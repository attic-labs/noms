// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	flag "github.com/juju/gnuflag"
)

var datasetRe = regexp.MustCompile("^" + dataset.DatasetRe.String() + "$")

func main() {
	if err := nomsMerge(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func nomsMerge() error {
	outDSStr := flag.String("out-ds-name", "", "output dataset to write to - if empty, defaults to <right-ds-name>")
	parentStr := flag.String("parent", "", "common ancestor of <left-ds-name> and <right-ds-name> (currently required; soon to be optional)")
	flag.Usage = usage

	return d.Unwrap(d.Try(func() {
		flag.Parse(false)

		if flag.NArg() == 0 {
			flag.Usage()
			d.PanicIfTrue(true, "")
		}

		d.PanicIfTrue(flag.NArg() != 3, "Incorrect number of arguments\n")
		d.PanicIfTrue(*parentStr == "", "--parent is currently required\n")

		db, err := spec.GetDatabase(flag.Arg(0))
		defer db.Close()
		d.PanicIfError(err)

		makeDS := func(dsName string) dataset.Dataset {
			d.PanicIfTrue(!datasetRe.MatchString(dsName), "Invalid dataset %s, must match %s\n", dsName, dataset.DatasetRe.String())
			return dataset.NewDataset(db, dsName)
		}

		leftDS := makeDS(flag.Arg(1))
		rightDS := makeDS(flag.Arg(2))
		parentDS := makeDS(*parentStr)

		parent, ok := parentDS.MaybeHeadValue()
		d.PanicIfTrue(!ok, "Dataset %s has no data\n", *parentStr)
		left, ok := leftDS.MaybeHeadValue()
		d.PanicIfTrue(!ok, "Dataset %s has no data\n", flag.Arg(1))
		right, ok := rightDS.MaybeHeadValue()
		d.PanicIfTrue(!ok, "Dataset %s has no data\n", flag.Arg(2))

		outDS := rightDS
		if *outDSStr != "" {
			outDS = makeDS(*outDSStr)
		}

		merged, err := merge.ThreeWay(left, right, parent, db)
		d.PanicIfError(err)

		_, err = outDS.Commit(merged, dataset.CommitOptions{Parents: types.NewSet(leftDS.HeadRef(), rightDS.HeadRef())})
		d.PanicIfError(err)
	}))
}

func usage() {
	fmt.Fprintf(os.Stderr, "Attempts to merge the two datasets in the provided database and commit the merge to either <right-ds-name> or another dataset of your choice.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [--out-ds-name=<name>] [--parent=<name>] <db-spec> <left-ds-name> <right-ds-name>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  <db-spec>       : database in which named datasets live\n")
	fmt.Fprintf(os.Stderr, "  <left-ds-name>  : name of a dataset descending from <parent>\n")
	fmt.Fprintf(os.Stderr, "  <right-ds-name> : name of another dataset descending from <parent>\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n\n")
	flag.PrintDefaults()
}
