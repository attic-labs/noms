// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	flag "github.com/juju/gnuflag"
)

var toDelete string
var verbose bool

var nomsDs = &util.Command{
	Run:       runDs,
	UsageLine: "ds [<database> | -d <dataset>]",
	Short:     "Noms dataset management",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database and dataset arguments.",
	Flags:     setupDsFlags,
	Nargs:     0,
}

func setupDsFlags() *flag.FlagSet {
	dsFlagSet := flag.NewFlagSet("ds", flag.ExitOnError)
	dsFlagSet.StringVar(&toDelete, "d", "", "dataset to delete")
	spec.RegisterVerboseFlags(dsFlagSet)
	return dsFlagSet
}

func runDs(args []string) int {
	resolver := spec.NewResolver()
	if toDelete != "" {
		db, set, err := resolver.GetDataset(toDelete) // append local if no local given, check for aliases
		d.CheckError(err)
		defer db.Close()

		oldCommitRef, errBool := set.MaybeHeadRef()
		if !errBool {
			d.CheckError(fmt.Errorf("Dataset %v not found", set.ID()))
		}

		_, err = set.Database().Delete(set)
		d.CheckError(err)

		fmt.Printf("Deleted %v (was #%v)\n", toDelete, oldCommitRef.TargetHash().String())
	} else {
		dbSpec := ""
		if len(args) >= 1 {
			dbSpec = args[0]
		}
		store, err := resolver.GetDatabase(dbSpec)
		d.CheckError(err)
		defer store.Close()

		store.Datasets().IterAll(func(k, v types.Value) {
			fmt.Println(k)
		})
	}
	return 0
}
