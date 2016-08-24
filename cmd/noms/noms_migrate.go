// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/noms/cmd/noms/migrate"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/d"
	v7datas "github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	v7spec "github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	flag "github.com/juju/gnuflag"
)

var nomsMigrate = &util.Command{
	Run:       runMigrate,
	Flags:     setupMigrateFlags,
	UsageLine: "migrate [options] <source-object> <dest-dataset>",
	Short:     "Migrates between versions of Noms",
	Long:      "",
	Nargs:     2,
}

func setupMigrateFlags() *flag.FlagSet {
	return flag.NewFlagSet("migrate", flag.ExitOnError)
}

func runMigrate(args []string) int {
	// TODO: verify source store is expected version
	// TODO: support multiple source versions
	// TODO: parallelize
	// TODO: incrementalize

	sourceStore, sourceObj, err := v7spec.GetPath(args[0])
	d.CheckError(err)
	defer sourceStore.Close()

	if sourceObj == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", args[0]))
	}

	isCommit := v7datas.IsCommitType(sourceObj.Type())

	sinkDataset, err := spec.GetDataset(args[1])
	d.CheckError(err)
	defer sinkDataset.Database().Close()

	sinkObj, err := migrate.Value(sourceObj, sourceStore, sinkDataset.Database())
	d.CheckError(err)

	if isCommit {
		// Commit will assert that we got a Commit struct.
		_, err = sinkDataset.Database().Commit(sinkDataset.ID(), sinkObj.(types.Struct))
	} else {
		_, err = sinkDataset.CommitValue(sinkObj)
	}
	d.CheckError(err)

	return 0
}
