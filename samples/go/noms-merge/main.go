// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/status"
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
	quiet := flag.Bool("quiet", false, "silence progress output")
	flag.Usage = usage

	return d.Unwrap(d.Try(func() {
		flag.Parse(false)

		if flag.NArg() == 0 {
			flag.Usage()
			d.PanicIfTrue(true, "")
		}

		d.PanicIfTrue(flag.NArg() != 3, "Incorrect number of arguments\n")
		d.PanicIfTrue(*parentStr == "", "--parent is required\n")

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

		pc := make(chan struct{}, 128)
		go func() {
			count := 0
			for _ = range pc {
				if !*quiet {
					count++
					status.Printf("Applied %d changes...", count)
				}
			}
		}()
		rand.Seed(time.Now().UnixNano())
		merged, err := merge.ThreeWay(left, right, parent, db, resolve, pc)
		d.PanicIfError(err)

		_, err = outDS.Commit(merged, dataset.CommitOptions{
			Parents: types.NewSet(leftDS.HeadRef(), rightDS.HeadRef()),
			Meta:    parentDS.Head().Get(datas.MetaField).(types.Struct),
		})
		d.PanicIfError(err)
		if !*quiet {
			status.Printf("Done")
			status.Done()
		}
	}))
}

func resolve(aChange, bChange types.ValueChanged, a, b types.Value, path types.Path) (change types.ValueChanged, merged types.Value, ok bool) {
	stringer := func(v types.Value) (s string, success bool) {
		switch v := v.(type) {
		case types.Bool, types.Number, types.String:
			return fmt.Sprintf("%v", v), true
		}
		return "", false
	}
	left, lOk := stringer(a)
	right, rOk := stringer(b)
	if !lOk || !rOk {
		return change, merged, false
	}

	// TODO: Handle removes as well.
	fmt.Printf("\nConflict at: %s\n", path.String())
	fmt.Printf("Left:  %s\nRight: %s\n\n", left, right)
	var choice rune
	for {
		fmt.Println("Enter 'l' to accept the left value, 'r' to accept the right value, or 'm' to mash them together")
		_, err := fmt.Scanf("%c\n", &choice)
		d.PanicIfError(err)
		switch choice {
		case 'l', 'L':
			return aChange, a, true
		case 'r', 'R':
			return bChange, b, true
		case 'm', 'M':
			if !a.Type().Equals(b.Type()) {
				fmt.Printf("Sorry, can't smush a %s with a %s\n", a.Type().Describe(), b.Type().Describe())
			}
			switch a := a.(type) {
			case types.Bool:
				return aChange, types.Bool(bool(a) || bool(b.(types.Bool))), true
			case types.Number:
				return aChange, types.Number(float64(a) + float64(b.(types.Number))), true
			case types.String:
				mashed := mash(a, b.(types.String))
				fmt.Println("Replacing with", mashed)
				return aChange, mashed, true
			}
		}
	}
}

func mash(a, b types.String) types.String {
	out := append([]byte(a), []byte(b)...)
	for i := range out {
		j := rand.Intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return types.String(string(out))
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
