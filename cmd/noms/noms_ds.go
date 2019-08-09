// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

func nomsDs(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("ds", "Dataset management.")
	del := cmd.Flag("delete", "delete a dataset").Short('d').Bool()
	name := cmd.Arg("name", "name of the database to list or dataset to delete - see Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").String()

	return cmd, func(input string) int {
		cfg := config.NewResolver()
		if *del {
			db, set, err := cfg.GetDataset(*name)
			d.CheckError(err)
			defer db.Close()

			oldCommitRef, errBool := set.MaybeHeadRef()
			if !errBool {
				d.CheckError(fmt.Errorf("Dataset %v not found", set.ID()))
			}

			_, err = set.Database().Delete(set)
			d.CheckError(err)

			fmt.Printf("Deleted %v (was #%v)\n", *name, oldCommitRef.TargetHash().String())
		} else {
			store, err := cfg.GetDatabase(*name)
			d.CheckError(err)
			defer store.Close()

			store.Datasets().IterAll(func(k, v types.Value) {
				fmt.Println(k)
			})
		}
		return 0
	}
}
