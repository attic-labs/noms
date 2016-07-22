// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/jsontonoms"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/clbanning/mxj"
	flag "github.com/ogier/pflag"
)

var (
	noIO        = flag.Bool("benchmark", false, "Run in 'benchmark' mode, without file-IO")
	customUsage = func() {
		fmtString := `%s walks the given directory, looking for .xml files. When it finds one, the entity inside is parsed into nested Noms maps/lists and committed to the dataset indicated on the command line.`
		fmt.Fprintf(os.Stderr, fmtString, os.Args[0])
		fmt.Fprintf(os.Stderr, "\n\nUsage: %s [options] <dataset> <path/to/root/directory>\n", os.Args[0])
		flag.PrintDefaults()
	}
)

type fileIndex struct {
	path  string
	index int
}

type refIndex struct {
	ref   types.Ref
	index int
}

type refIndexList []refIndex

func (a refIndexList) Len() int           { return len(a) }
func (a refIndexList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a refIndexList) Less(i, j int) bool { return a[i].index < a[j].index }

func main() {
	err := d.Try(func() {
		spec.RegisterDatabaseFlags(flag.CommandLine)
		profile.RegisterProfileFlags(flag.CommandLine)
		flag.Usage = customUsage
		flag.Parse()

		if flag.NArg() != 2 {
			d.CheckError(errors.New("Expected dataset followed by directory path"))
		}
		dir := flag.Arg(1)
		ds, err := spec.GetDataset(flag.Arg(0))
		d.CheckError(err)

		defer profile.MaybeStartProfile().Stop()

		cpuCount := runtime.NumCPU()

		filesChan := make(chan fileIndex, 1024)
		refsChan := make(chan refIndex, 1024)

		getFilePaths := func() {
			index := 0
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				d.PanicIfTrue(err != nil, "Cannot traverse directories")
				if !info.IsDir() && filepath.Ext(path) == ".xml" {
					filesChan <- fileIndex{path, index}
					index++
				}

				return nil
			})
			d.PanicIfError(err)
			close(filesChan)
		}

		wg := sync.WaitGroup{}
		importXML := func() {
			expectedType := types.NewMap()
			for f := range filesChan {
				file, err := os.Open(f.path)
				d.PanicIfTrue(err != nil, "Error getting XML")

				xmlObject, err := mxj.NewMapXmlReader(file)
				d.PanicIfTrue(err != nil, "Error decoding XML")
				object := xmlObject.Old()
				file.Close()

				nomsObj := jsontonoms.NomsValueFromDecodedJSON(object, false)
				d.Chk.IsType(expectedType, nomsObj)

				var r types.Ref
				if !*noIO {
					r = ds.Database().WriteValue(nomsObj)
				}

				refsChan <- refIndex{r, f.index}
			}

			wg.Done()
		}

		go getFilePaths()
		for i := 0; i < cpuCount*8; i++ {
			wg.Add(1)
			go importXML()
		}
		go func() {
			wg.Wait()
			close(refsChan) // done converting xml to noms
		}()

		refList := refIndexList{}
		for r := range refsChan {
			refList = append(refList, r)
		}
		sort.Sort(refList)

		refs := make([]types.Value, len(refList))
		for idx, r := range refList {
			refs[idx] = r.ref
		}

		rl := types.NewList(refs...)

		if !*noIO {
			_, err := ds.CommitValue(rl)
			d.PanicIfError(err)
		}
	})

	if err != nil {
		log.Fatal(err)
	}
}
