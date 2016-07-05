// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
)

var (
	dsFlagSet = flag.NewFlagSet("serve", flag.ExitOnError)
	toDelete  = dsFlagSet.String("d", "", "dataset to delete")
)

var nomsDs = &nomsCommand{
	Run:       runDs,
	UsageLine: "ds [<database> | -d <dataset>]",
	Short:     "Noms dataset management",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database and dataset arguments.",
	Flag:      dsFlagSet,
}

func runDs(args []string) int {
	if *toDelete != "" {
		set, err := spec.GetDataset(*toDelete)
		d.CheckError(err)

		oldCommitRef, errBool := set.MaybeHeadRef()
		if !errBool {
			d.CheckError(fmt.Errorf("Dataset %v not found", set.ID()))
		}

		store, err := set.Database().Delete(set.ID())
		d.CheckError(err)
		defer store.Close()

		fmt.Printf("Deleted dataset %v (was %v)\n\n", set.ID(), oldCommitRef.TargetHash().String())
	} else {
		if len(args) != 1 {
			return 0
		}

		store, err := spec.GetDatabase(args[0])
		d.CheckError(err)
		defer store.Close()

		store.Datasets().IterAll(func(k, v types.Value) {
			fmt.Println(k)
		})
	}
	return 0
}
