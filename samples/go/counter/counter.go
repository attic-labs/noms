// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
)

func main() {
	app := kingpin.New("counter", "")
	dsStr := app.Arg("ds", "dataset to count in").Required().String()
	verbose.RegisterVerboseFlags(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	cfg := config.NewResolver()
	db, ds, err := cfg.GetDataset(*dsStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create dataset: %s\n", err)
		return
	}
	defer db.Close()

	newVal := uint64(1)
	if lastVal, ok := ds.MaybeHeadValue(); ok {
		newVal = uint64(lastVal.(types.Number)) + 1
	}

	_, err = db.CommitValue(ds, types.Number(newVal))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error committing: %s\n", err)
		return
	}

	fmt.Println(newVal)
}
