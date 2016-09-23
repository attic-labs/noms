// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/walk"
	flag "github.com/juju/gnuflag"
)

func main() {
	if !index() {
		os.Exit(1)
	}
}

func index() (win bool) {
	var dbStr = flag.String("db", "", "input database spec")
	var outDSStr = flag.String("out-ds", "", "output dataset to write to - if empty, defaults to input dataset")

	flag.Usage = usage
	flag.Parse(false)

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Need at least one dataset to index")
		return
	}

	db, err := spec.GetDatabase(*dbStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid input database '%s': %s\n", flag.Arg(0), err)
		return
	}
	defer db.Close()

	var outDS dataset.Dataset
	if !dataset.DatasetFullRe.MatchString(*outDSStr) {
		fmt.Fprintf(os.Stderr, "Invalid output dataset name: %s\n", *outDSStr)
		return
	} else {
		outDS = dataset.NewDataset(db, *outDSStr)
	}

	inputs := []types.Value{}
	for i := 0; i < flag.NArg(); i++ {
		if dataset.DatasetFullRe.MatchString(flag.Arg(i)) {
			ds := dataset.NewDataset(db, flag.Arg(i))
			v, ok := ds.MaybeHeadValue()
			if ok {
				inputs = append(inputs, v)
				continue
			}
		}
		fmt.Fprintf(os.Stderr, "Could not load dataset '%s', error: %s\n", flag.Arg(i), err)
		return
	}

	st := types.MakeStructType("",
		[]string{"height", "width"},
		[]*types.Type{types.NumberType, types.NumberType},
	)
	pt := types.MakeStructType("Photo",
		[]string{"sizes", "tags", "title"},
		[]*types.Type{
			types.MakeMapType(st, types.StringType),
			types.MakeSetType(types.StringType),
			types.StringType,
		})

	gb := types.NewGraphBuilder(db, types.MapKind, true)
	for _, v := range inputs {
		walk.SomeP(v, db, func(cv types.Value, _ *types.Ref) (stop bool) {
			if types.IsSubtype(pt, cv.Type()) {
				p := cv.(types.Struct)
				tags := p.Get("tags").(types.Set)
				tags.IterAll(func(t types.Value) {
					gb.SetInsert([]types.Value{t}, p)
				})
				// Can't be any photos inside photos, so we can save a little bit here.
				stop = true
			}
			return
		}, 12)
	}

	_, err = outDS.CommitValue(gb.Build())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not commit: %s\n", err)
		return
	}

	win = true
	return
}

func usage() {
	fmt.Fprintf(os.Stderr, "photo-index indexes photos by common attributes\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s -db=<db-spec> -out-ds=<name> [input-datasets...]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  <db>             : Database to work with\n")
	fmt.Fprintf(os.Stderr, "  <out-ds>         : Dataset to write index to\n")
	fmt.Fprintf(os.Stderr, "  <input-datasets> : Input datasets to crawl\n\n")
	fmt.Fprintln(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}
