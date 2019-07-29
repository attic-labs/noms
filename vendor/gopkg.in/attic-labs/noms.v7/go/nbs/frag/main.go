// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/datas"
	"gopkg.in/attic-labs/noms.v7/go/hash"
	"gopkg.in/attic-labs/noms.v7/go/nbs"
	"gopkg.in/attic-labs/noms.v7/go/types"
	"gopkg.in/attic-labs/noms.v7/go/util/profile"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

var (
	dir    = flag.String("dir", "", "Write to an NBS store in the given directory")
	table  = flag.String("table", "", "Write to an NBS store in AWS, using this table")
	bucket = flag.String("bucket", "", "Write to an NBS store in AWS, using this bucket")
	dbName = flag.String("db", "", "Write to an NBS store in AWS, using this db name")
)

const memTableSize = 128 * humanize.MiByte

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	profile.RegisterProfileFlags(flag.CommandLine)
	flag.Parse(true)

	if flag.NArg() != 0 {
		flag.Usage()
		return
	}

	var store *nbs.NomsBlockStore
	if *dir != "" {
		store = nbs.NewLocalStore(*dir, memTableSize)
		*dbName = *dir
	} else if *table != "" && *bucket != "" && *dbName != "" {
		sess := session.Must(session.NewSession(aws.NewConfig().WithRegion("us-west-2")))
		store = nbs.NewAWSStore(*table, *dbName, *bucket, s3.New(sess), dynamodb.New(sess), memTableSize)
	} else {
		log.Fatalf("Must set either --dir or ALL of --table, --bucket and --db\n")
	}

	db := datas.NewDatabase(store)
	defer db.Close()

	defer profile.MaybeStartProfile().Stop()

	type record struct {
		count, calc int
		split       bool
	}

	concurrency := 32
	refs := make(chan types.Ref, concurrency)
	numbers := make(chan record, concurrency)
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	visitedRefs := map[hash.Hash]bool{}

	for i := 0; i < concurrency; i++ {
		go func() {
			for ref := range refs {
				mu.Lock()
				visited := visitedRefs[ref.TargetHash()]
				visitedRefs[ref.TargetHash()] = true
				mu.Unlock()

				if !visited {
					v := ref.TargetValue(db)
					d.Chk.NotNil(v)

					children := types.RefSlice{}
					hashes := hash.HashSlice{}
					v.WalkRefs(func(r types.Ref) {
						hashes = append(hashes, r.TargetHash())
						if r.Height() > 1 { // leaves are uninteresting, so skip them.
							children = append(children, r)
						}
					})

					reads, split := store.CalcReads(hashes.HashSet(), 0)
					numbers <- record{count: 1, calc: reads, split: split}

					wg.Add(len(children))
					go func() {
						for _, r := range children {
							refs <- r
						}
					}()
				}
				wg.Done()
			}
		}()
	}

	wg.Add(1)
	refs <- types.NewRef(db.Datasets())
	go func() {
		wg.Wait()
		close(refs)
		close(numbers)
	}()

	count, calc, splits := 0, 0, 0
	for rec := range numbers {
		count += rec.count
		calc += rec.calc
		if rec.split {
			splits++
		}
	}

	fmt.Println("calculated optimal Reads", count)
	fmt.Printf("calculated actual Reads %d, including %d splits across tables\n", calc, splits)
	fmt.Printf("Reading DB %s requires %.01fx optimal number of reads\n", *dbName, float64(calc)/float64(count))
}
