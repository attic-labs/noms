// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"os"

	flag "github.com/juju/gnuflag"
	"gopkg.in/attic-labs/noms.v7/go/config"
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/types"
	"gopkg.in/attic-labs/noms.v7/go/util/profile"
	"gopkg.in/attic-labs/noms.v7/go/util/verbose"
	"gopkg.in/attic-labs/noms.v7/samples/go/csv"
)

func main() {
	// Actually the delimiter uses runes, which can be multiple characters long.
	// https://blog.golang.org/strings
	delimiter := flag.String("delimiter", ",", "field delimiter for csv file, must be exactly one character long.")

	verbose.RegisterVerboseFlags(flag.CommandLine)
	profile.RegisterProfileFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: csv-export [options] dataset > filename")
		flag.PrintDefaults()
	}

	flag.Parse(true)

	if flag.NArg() != 1 {
		d.CheckError(errors.New("expected dataset arg"))
	}

	cfg := config.NewResolver()
	db, ds, err := cfg.GetDataset(flag.Arg(0))
	d.CheckError(err)

	defer db.Close()

	comma, err := csv.StringToRune(*delimiter)
	d.CheckError(err)

	err = d.Try(func() {
		defer profile.MaybeStartProfile().Stop()

		hv := ds.HeadValue()
		if l, ok := hv.(types.List); ok {
			structDesc := csv.GetListElemDesc(l, db)
			csv.WriteList(l, structDesc, comma, os.Stdout)
		} else if m, ok := hv.(types.Map); ok {
			structDesc := csv.GetMapElemDesc(m, db)
			csv.WriteMap(m, structDesc, comma, os.Stdout)
		} else {
			panic(fmt.Sprintf("Expected ListKind or MapKind, found %s", hv.Kind()))
		}
	})
	if err != nil {
		fmt.Println("Failed to export dataset as CSV:")
		fmt.Println(err)
	}
}
