// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

const memTableSize = 128 * humanize.MiByte

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <noms-path>\n", os.Args[0])
		flag.PrintDefaults()
	}

	profile.RegisterProfileFlags(flag.CommandLine)
	flag.Parse(true)

	if flag.NArg() != 1 {
		flag.Usage()
		return
	}
	path := flag.Arg(0)
	parts := strings.SplitN(path, spec.Separator, 2) // db [, path]?

	cfg := config.NewResolver()
	cs, err := cfg.GetChunkStore(parts[0])
	d.CheckErrorNoUsage(err)

	var store *nbs.NomsBlockStore
	var ok bool
	if store, ok = cs.(*nbs.NomsBlockStore); !ok {
		fmt.Fprintf(os.Stderr, "%s is only supported for NBS stores.\n", os.Args[0])
		return
	}
	db := datas.NewDatabase(store)
	defer db.Close()

	var value types.Value
	if len(parts) == 1 {
		value = db.Datasets()
	} else {
		path, err := spec.NewAbsolutePath(parts[1])
		d.CheckErrorNoUsage(err)
		value = path.Resolve(db)
		d.Chk.NotNil(value)
	}

	defer profile.MaybeStartProfile().Stop()

	type record struct {
		count, allCalc, novelCalc, novel int
		allSplit, novelSplit             bool
	}

	concurrency := 32
	childSets := make(chan types.RefSlice, concurrency)
	numbers := make(chan record, concurrency)
	wg := sync.WaitGroup{}
	mu := sync.RWMutex{}
	visitedNodes := hash.HashSet{}

	for i := 0; i < concurrency; i++ {
		go func() {
			for chirren := range childSets {
				hashes, visitedHashes := hash.HashSlice{}, hash.HashSlice{}
				for _, child := range chirren {
					mu.Lock()
					visited := visitedNodes.Has(child.TargetHash())
					visitedNodes.Insert(child.TargetHash())
					mu.Unlock()

					childV := child.TargetValue(db)
					d.Chk.NotNil(childV)
					grandkids, grandkidHashes := getChildren(childV)

					hashes = append(hashes, grandkidHashes...)
					if visited {
						visitedHashes = append(visitedHashes, grandkidHashes...)
					}

					if num := len(grandkids); num > 0 {
						wg.Add(num)
						go func() {
							childSets <- grandkids
						}()
					}
				}
				if len(hashes) > 0 {
					set := hashes.HashSet()
					allReads, allSplit := store.CalcReads(set, 0)
					for _, h := range visitedHashes {
						set.Remove(h)
					}
					novelReads, novelSplit := store.CalcReads(set, 0)
					numbers <- record{count: 1, allCalc: allReads, allSplit: allSplit, novelCalc: novelReads, novelSplit: novelSplit, novel: len(hashes)}
				}
				wg.Add(-len(chirren))
			}
		}()
	}

	chirren, hashes := getChildren(value)
	wg.Add(len(chirren))
	childSets <- chirren

	go func() {
		wg.Wait()
		close(childSets)
		close(numbers)
	}()

	count := 1 // To account for reading children of |value|
	calc, splits, novel := 0, 0, 0
	calc, split := store.CalcReads(hashes.HashSet(), 0)
	if split {
		splits++
	}
	novelCalc, novelSplits := calc, splits
	for rec := range numbers {
		count += rec.count
		calc += rec.allCalc
		if rec.allSplit {
			splits++
		}
		novelCalc += rec.novelCalc
		if rec.novelSplit {
			novelSplits++
		}
		novel += rec.novel
	}

	fmt.Println("calculated optimal Reads", count)
	fmt.Printf("calculated actual Reads %d, including %d splits across tables\n", calc, splits)
	fmt.Printf("Reading %s requires %.01fx optimal number of reads\n", path, float64(calc)/float64(count))
	fmt.Println()
	fmt.Println("visited novel nodes:", novel)
	fmt.Printf("calculated novel Reads %d, including %d splits across tables\n", novelCalc, novelSplits)
	fmt.Printf("Reading each chunk of %s once requires %.01fx optimal number of reads\n", path, float64(novelCalc)/float64(count))
}

func getChildren(v types.Value) (children types.RefSlice, hashes hash.HashSlice) {
	v.WalkRefs(func(r types.Ref) {
		hashes = append(hashes, r.TargetHash())
		if r.Height() > 1 { // leaves are uninteresting, so skip them.
			children = append(children, r)
		}
	})
	return
}
