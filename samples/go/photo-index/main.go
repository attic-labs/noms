// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/exit"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/attic-labs/noms/go/walk"
	flag "github.com/juju/gnuflag"
)

func main() {
	if !index() {
		exit.Fail()
	}
}

func index() (win bool) {
	var dbStr = flag.String("db", "", "input database spec")
	var outDSStr = flag.String("out-ds", "", "output dataset to write to - if empty, defaults to input dataset")
	verbose.RegisterVerboseFlags(flag.CommandLine)

	flag.Usage = usage
	flag.Parse(false)

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	cfg := config.NewResolver()
	db, err := cfg.GetDatabase(*dbStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid input database '%s': %s\n", flag.Arg(0), err)
		return
	}
	defer db.Close()

	var outDS datas.Dataset
	if !datas.IsValidDatasetName(*outDSStr) {
		fmt.Fprintf(os.Stderr, "Invalid output dataset name: %s\n", *outDSStr)
		return
	} else {
		outDS = db.GetDataset(*outDSStr)
	}

	inputs, err := spec.ReadAbsolutePaths(db, flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	sizeType := types.MakeStructTypeFromFields("", types.FieldMap{
		"width":  types.NumberType,
		"height": types.NumberType,
	})
	dateType := types.MakeStructTypeFromFields("Date", types.FieldMap{
		"nsSinceEpoch": types.NumberType,
	})
	faceType := types.MakeStructTypeFromFields("", types.FieldMap{
		"name": types.StringType,
		"x":    types.NumberType,
		"y":    types.NumberType,
		"w":    types.NumberType,
		"h":    types.NumberType,
	})
	photoType := types.MakeStructTypeFromFields("Photo", types.FieldMap{
		"id":    types.StringType,
		"sizes": types.MakeMapType(sizeType, types.StringType),
	})

	withTags := types.MakeStructTypeFromFields("", types.FieldMap{
		"tags": types.MakeSetType(types.StringType),
	})
	withFaces := types.MakeStructTypeFromFields("", types.FieldMap{
		"faces": types.MakeSetType(faceType),
	})
	withDateTaken := types.MakeStructTypeFromFields("", types.FieldMap{
		"dateTaken": dateType,
	})
	withDatePublished := types.MakeStructTypeFromFields("", types.FieldMap{
		"datePublished": dateType,
	})
	withDateUpdated := types.MakeStructTypeFromFields("", types.FieldMap{
		"dateUpdated": dateType,
	})

	byDate := types.NewGraphBuilder(db, types.MapKind, true)
	byTag := types.NewGraphBuilder(db, types.MapKind, true)
	byFace := types.NewGraphBuilder(db, types.MapKind, true)
	byYear := types.NewGraphBuilder(db, types.MapKind, true)

	tagCounts := map[types.String]int{}
	faceCounts := map[types.String]int{}
	countsMtx := sync.Mutex{}

	for _, v := range inputs {
		walk.WalkValues(v, db, func(cv types.Value) (stop bool) {
			if types.IsSubtype(photoType, cv.Type()) {
				s := cv.(types.Struct)

				// None of the date fields are required, but they are usually available.
				var ds types.Value
				if types.IsSubtype(withDateTaken, cv.Type()) {
					ds = s.Get("dateTaken")
				} else if types.IsSubtype(withDatePublished, cv.Type()) {
					ds = s.Get("datePublished")
				} else if types.IsSubtype(withDateUpdated, cv.Type()) {
					ds = s.Get("dateUpdated")
				}

				d := types.Number(float64(math.MaxFloat64))
				if ds != nil {
					// Sort by most recent by negating the timestamp.
					d = ds.(types.Struct).Get("nsSinceEpoch").(types.Number)
					d = types.Number(-float64(d))
				}

				// Index by date
				byDate.SetInsert([]types.Value{d}, cv)

				t := time.Unix(0, int64(-d))
				byYear.SetInsert([]types.Value{types.Number(t.Year()),
					types.Number(t.Month()), types.Number(t.Day()), d}, cv)

				// Index by tag, then date
				moreTags := map[types.String]int{}
				if types.IsSubtype(withTags, cv.Type()) {
					s.Get("tags").(types.Set).IterAll(func(t types.Value) {
						byTag.SetInsert([]types.Value{t, d}, cv)
						moreTags[t.(types.String)]++
					})
				}

				// Index by face, then date
				moreFaces := map[types.String]int{}
				if types.IsSubtype(withFaces, cv.Type()) {
					s.Get("faces").(types.Set).IterAll(func(t types.Value) {
						name := t.(types.Struct).Get("name").(types.String)
						byFace.SetInsert([]types.Value{name, d}, cv)
						moreFaces[name]++
					})
				}

				countsMtx.Lock()
				for tag, count := range moreTags {
					tagCounts[tag] += count
				}
				for face, count := range moreFaces {
					faceCounts[face] += count
				}
				countsMtx.Unlock()

				// Can't be any photos inside photos, so we can save a little bit here.
				stop = true
			}
			return
		})
	}

	outDS, err = db.Commit(outDS, types.NewStruct("", types.StructData{
		"byDate":       byDate.Build(),
		"byTag":        byTag.Build(),
		"byFace":       byFace.Build(),
		"byYear":       byYear.Build(),
		"tagsByCount":  stringsByCount(db, tagCounts),
		"facesByCount": stringsByCount(db, faceCounts),
	}), datas.CommitOptions{
		Meta: types.NewStruct("", types.StructData{
			"date": types.String(time.Now().Format(time.RFC3339)),
		}),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not commit: %s\n", err)
		return
	}

	win = true
	return
}

func stringsByCount(db datas.Database, strings map[types.String]int) types.Map {
	b := types.NewGraphBuilder(db, types.MapKind, true)
	for s, count := range strings {
		// Sort by largest count by negating.
		b.SetInsert([]types.Value{types.Number(-count)}, s)
	}
	return b.Build().(types.Map)
}

func usage() {
	fmt.Fprintf(os.Stderr, "photo-index indexes photos by common attributes\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s -db=<db-spec> -out-ds=<name> [input-paths...]\n\n", path.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  <db>             : Database to work with\n")
	fmt.Fprintf(os.Stderr, "  <out-ds>         : Dataset to write index to\n")
	fmt.Fprintf(os.Stderr, "  [input-paths...] : One or more paths within <db-spec> to crawl\n\n")
	fmt.Fprintln(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}
