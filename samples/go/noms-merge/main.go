// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	flag "github.com/juju/gnuflag"
)

var datasetRe = regexp.MustCompile("^" + dataset.DatasetRe.String() + "$")

func main() {
	var outDSStr = flag.String("out-ds-name", "", "output dataset to write to - if empty, defaults to <right-ds-name>")
	var parentStr = flag.String("parent", "", "common ancestor of <left-ds-name> and <right-ds-name> (currently required; soon to be optional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Attempts to merge the two datasets in the provided database and commit the merge to either <right-ds-name> or another dataset of your choice.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [--out-ds-name=<name>] [--parent=<name>] <db-spec> <left-ds-name> <right-ds-name>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  <db-spec>       : database in which named datasets live\n")
		fmt.Fprintf(os.Stderr, "  <left-ds-name>  : name of a dataset descending from <parent>\n")
		fmt.Fprintf(os.Stderr, "  <right-ds-name> : name of another dataset descending from <parent>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n\n")
		flag.PrintDefaults()
	}

	flag.Parse(false)

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	if flag.NArg() != 3 {
		log.Fatalln("Incorrect number of arguments")
	}

	db, err := spec.GetDatabase(flag.Arg(0))
	if err != nil {
		log.Fatalf("Invalid database '%s': %s\n", flag.Arg(0), err)
	}
	defer db.Close()

	makeDS := func(dsName string) dataset.Dataset {
		if !datasetRe.MatchString(dsName) {
			log.Fatalf("Invalid dataset %s, must match %s\n", dsName, dataset.DatasetRe.String())
		}
		return dataset.NewDataset(db, dsName)
	}

	leftDS := makeDS(flag.Arg(1))
	rightDS := makeDS(flag.Arg(2))
	parentDS := makeDS(*parentStr)

	parent, ok := parentDS.MaybeHeadValue()
	if !ok {
		log.Fatalln("Parent dataset has no data")
	}
	left, ok := leftDS.MaybeHeadValue()
	if !ok {
		log.Fatalln("left dataset has no data")
	}
	right, ok := rightDS.MaybeHeadValue()
	if !ok {
		log.Fatalln("right dataset has no data")
	}

	outDS := rightDS
	if *outDSStr != "" {
		outDS = makeDS(*outDSStr)
	}

	merged, err := merge.ThreeWay(left, right, parent, db)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = outDS.Commit(merged, dataset.CommitOptions{Parents: types.NewSet(leftDS.HeadRef(), rightDS.HeadRef())})
	if err != nil {
		log.Fatalln(err)
	}
}
