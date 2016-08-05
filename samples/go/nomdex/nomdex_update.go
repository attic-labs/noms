// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/status"
	"github.com/attic-labs/noms/go/walk"
	humanize "github.com/dustin/go-humanize"
	flag "github.com/tsuru/gnuflag"
)

var (
	inPathArg  = ""
	outDsArg   = ""
	relPathArg = ""
)

var update = &nomdexCommand{
	Run:       runUpdate,
	UsageLine: "up -in-path <path> -out-ds <dspath> -by <relativepath>",
	Short:     "Build/Update an index",
	Long:      "Traverse all values starting at root and add values found at 'relativePath' to a map found at 'out-ds'\n",
	Flags:     setupUpdateFlags,
	Nargs:     0,
}

func setupUpdateFlags() *flag.FlagSet {
	flagSet := flag.NewFlagSet("up", flag.ExitOnError)
	flagSet.StringVar(&inPathArg, "in-path", "", "a value to search for items to index within ")
	flagSet.StringVar(&outDsArg, "out-ds", "", "name of dataset to save the results to")
	flagSet.StringVar(&relPathArg, "by", "", "a path relative to all the items in <in-path> to index by")
	return flagSet
}

type IndexMap map[types.Value][]types.Value

type Index struct {
	m     IndexMap
	mutex sync.Mutex
}

func runUpdate(args []string) int {
	db, rootObject, err := spec.GetPath(inPathArg)
	d.Chk.NoError(err)

	outDs := dataset.NewDataset(db, outDsArg)
	relPath, err := types.ParsePath(relPathArg)
	if printError(err, "Error parsing -by value\n\t") {
		return 1
	}

	typeCacheMutex := sync.Mutex{}
	typeCache := map[*types.Type]bool{}

	index := Index{m: IndexMap{}}

	walk.AllP(rootObject, db, func(v types.Value, r *types.Ref) {
		typ := v.Type()
		typeCacheMutex.Lock()
		hasPath, ok := typeCache[typ]
		typeCacheMutex.Unlock()
		if !ok || hasPath {
			pathResolved := false
			tv := relPath.Resolve(v)
			if tv != nil {
				index.Add(tv, v)
				pathResolved = true
			}
			if !ok {
				typeCacheMutex.Lock()
				typeCache[typ] = pathResolved
				typeCacheMutex.Unlock()
			}
		}
	}, 4)

	status.Done()
	indexMap := writeToStreamingMap(db, index.m)
	outDs, err = outDs.Commit(indexMap, dataset.CommitOptions{})
	d.Chk.NoError(err)
	fmt.Printf("Committed index with %d entries to dataset: %s\n", indexMap.Len(), outDsArg)
	return 0
}

var cnt = int64(0)

func (idx *Index) Add(k, v types.Value) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	cnt++
	l := idx.m[k]
	l1 := append(l, v)
	idx.m[k] = l1
	status.Printf("Indexed %s objects", humanize.Comma(cnt))
}

func writeToStreamingMap(db datas.Database, indexMap IndexMap) types.Map {
	itemCnt := len(indexMap)
	writtenCnt := int64(0)
	indexedCnt := int64(0)
	kvChan := make(chan types.Value)
	mapChan := types.NewStreamingMap(db, kvChan)
	for k, v := range indexMap {
		s := types.NewSet(v...)
		kvChan <- k
		kvChan <- s
		indexedCnt += int64(len(v))
		delete(indexMap, k)
		writtenCnt++
		status.Printf("Wrote %s/%d keys, %s indexedObjects", humanize.Comma(writtenCnt), itemCnt, humanize.Comma(indexedCnt))
	}
	close(kvChan)
	status.Done()
	return <-mapChan
}
