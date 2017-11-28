// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/attic-labs/noms/go/nbs"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/verbose"
	flag "github.com/juju/gnuflag"
)

func main() {
	t1 := time.Now()
	fmt.Println("Started at: ", t1)

	verbose.RegisterVerboseFlags(flag.CommandLine)
	profile.RegisterProfileFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sum-list path.to.list")
		flag.PrintDefaults()
	}

	flag.Parse(true)

	if flag.NArg() != 1 {
		d.CheckError(errors.New("expected path arg"))
	}

	cfg := config.NewResolver()
	db, v, err := cfg.GetPath(flag.Arg(0))
	d.CheckError(err)

	defer db.Close()

	stats1 := db.Stats().(nbs.Stats)
	fmt.Println(stats1)

	t2 := time.Now()

	l, ok := v.(types.List)
	if !ok {
		d.CheckError(errors.New("expected list"))
	}

	fmt.Println("Opened db", t2.Sub(t1))

	sum := float64(0)

	l.IterAll(func(v types.Value, i uint64) {
		sum += float64(v.(types.Number))
		if i%100000 == 0 {
			fmt.Println(sum, i)
		}
	})

	fmt.Println("Finsihed", sum, time.Since(t2))
	stats2 := db.Stats().(nbs.Stats).Delta(stats1)
	fmt.Println(stats2)
}
