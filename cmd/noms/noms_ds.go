// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// NomsDs - the noms ds (dataset) command
func NomsDs(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	ds := noms.Command("ds", `Noms dataset management

See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
`)

	toDelete := ds.Flag("delete", "dataset to delete").Short('d').String()
	AddVerboseFlags(ds)
	// TODO: break out delete and use
	// database := AddDatabaseArg(ds)
	// instead
	database := ds.Arg("database", "a noms database path").String()

	return ds, func() int {
		ApplyVerbosity()
		cfg := config.NewResolver()
		if toDelete != nil {
			db, set, err := cfg.GetDataset(*toDelete)
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
			store, err := cfg.GetDatabase(*database)
			d.CheckError(err)
			defer store.Close()

			store.Datasets().IterAll(func(k, v types.Value) {
				fmt.Println(k)
			})
		}
		return 0
	}
}
