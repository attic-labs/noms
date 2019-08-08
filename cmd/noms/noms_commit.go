// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
)

func nomsCommit(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	commit := noms.Command("commit", "commits a value to a dataset")
	allowDupe := commit.Flag("allow-dupe", "creates a new commit, even if it would be identical (modulo metadata and parents) to the existing HEAD").Bool()
	path := commit.Arg("absolute-path", "absolute path to value to commit - see See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()
	ds := commit.Arg("dataset", "dataset spec to commit to - see Spelling Datasets at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()

	return commit, func(input string) int {
		cfg := config.NewResolver()
		db, ds, err := cfg.GetDataset(*ds)
		d.CheckError(err)
		defer db.Close()

		absPath, err := spec.NewAbsolutePath(*path)
		d.CheckError(err)

		value := absPath.Resolve(db)
		if value == nil {
			d.CheckErrorNoUsage(errors.New(fmt.Sprintf("Error resolving value: %s", path)))
		}

		oldCommitRef, oldCommitExists := ds.MaybeHeadRef()
		if oldCommitExists {
			head := ds.HeadValue()
			if head.Hash() == value.Hash() && !*allowDupe {
				fmt.Fprintf(os.Stdout, "Commit aborted - allow-dupe is set to off and this commit would create a duplicate\n")
				return 0
			}
		}

		meta, err := spec.CreateCommitMetaStruct(db, "", "", nil, nil)
		d.CheckErrorNoUsage(err)

		ds, err = db.Commit(ds, value, datas.CommitOptions{Meta: meta})
		d.CheckErrorNoUsage(err)

		if oldCommitExists {
			fmt.Fprintf(os.Stdout, "New head #%v (was #%v)\n", ds.HeadRef().TargetHash().String(), oldCommitRef.TargetHash().String())
		} else {
			fmt.Fprintf(os.Stdout, "New head #%v\n", ds.HeadRef().TargetHash().String())
		}
		return 0
	}
}
