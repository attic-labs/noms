// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"sync"

	"github.com/stretchr/testify/assert"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	jsontonoms "github.com/attic-labs/noms/go/util/json"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/verbose/verboseflags"
	"github.com/clbanning/mxj"
)

var (
	noIO          = kingpin.Flag("benchmark", "Run in 'benchmark' mode: walk directories and parse XML files but do not write to Noms").Bool()
	performCommit = kingpin.Flag("commit", "commit the data to head of the dataset (otherwise only write the data to the dataset)").Default("true").Bool()
	rootDir       = kingpin.Arg("dir", "directory to find for xml files in").Required().String()
	dataset       = kingpin.Arg("dataset", "dataset to write to").Required().String()
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
		verboseflags.Register(kingpin.CommandLine)
		profile.RegisterProfileFlags(kingpin.CommandLine)
		kingpin.Parse()

		cfg := config.NewResolver()
		db, ds, err := cfg.GetDataset(*dataset)
		d.CheckError(err)
		defer db.Close()

		defer profile.MaybeStartProfile().Stop()

		cpuCount := runtime.NumCPU()

		filesChan := make(chan fileIndex, 1024)
		refsChan := make(chan refIndex, 1024)

		getFilePaths := func() {
			index := 0
			err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					d.Panic("Cannot traverse directories")
				}
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
			expectedType := types.NewMap(db)
			for f := range filesChan {
				file, err := os.Open(f.path)
				if err != nil {
					d.Panic("Error getting XML")
				}

				xmlObject, err := mxj.NewMapXmlReader(file)
				if err != nil {
					d.Panic("Error decoding XML")
				}
				object := xmlObject.Old()
				file.Close()

				nomsObj := jsontonoms.NomsValueFromDecodedJSON(db, object, false)
				d.Chk.True(assert.ObjectsAreEqual(
					reflect.TypeOf(expectedType), reflect.TypeOf(nomsObj)))

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

		rl := types.NewList(db, refs...)

		if !*noIO {
			if *performCommit {
				additionalMetaInfo := map[string]string{"inputDir": *rootDir}
				meta, err := spec.CreateCommitMetaStruct(ds.Database(), "", "", additionalMetaInfo, nil)
				d.CheckErrorNoUsage(err)
				_, err = db.Commit(ds, rl, datas.CommitOptions{Meta: meta})
				d.PanicIfError(err)
			} else {
				ref := db.WriteValue(rl)
				fmt.Fprintf(os.Stdout, "#%s\n", ref.TargetHash().String())
			}
		}
	})
	if err != nil {
		log.Fatal(err)
	}
}
