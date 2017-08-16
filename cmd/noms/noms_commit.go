// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
)

func nomsCommit(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	commit := noms.Command("commit", `Commits a specified value as head of the dataset

If absolute-path is not provided, then it is read from stdin. See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the dataset and absolute-path arguments.
`)
	allowDupe := commit.Flag("allow-dupe", "creates a new commit, even if it would be identical (modulo metadata and parents) to the existing HEAD.").Default("false").Bool()
	AddVerboseFlags(commit)
	database := AddDatabaseArg(commit)
	absolutePath := commit.Arg("absolute-path", "the path to read data from").String()

	return commit, func() int {
		ApplyVerbosity()
		cfg := config.NewResolver()
		db, ds, err := cfg.GetDataset(*database)
		d.CheckError(err)
		defer db.Close()

		var path string
		if absolutePath != nil {
			path = *absolutePath
		} else {
			readPath, _, err := bufio.NewReader(os.Stdin).ReadLine()
			d.CheckError(err)
			path = string(readPath)
		}
		absPath, err := spec.NewAbsolutePath(path)
		d.CheckError(err)

		value := absPath.Resolve(db)
		if value == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Error resolving value: %s", path))
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
