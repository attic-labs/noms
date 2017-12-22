// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"runtime"
	"strconv"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/d"
	"gopkg.in/alecthomas/kingpin.v2"
)

func nomsBlob(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	blob := noms.Command("blob", "interact with blobs")

	blobPut := blob.Command("put", "imports a blob to a dataset")
	concurrency := blobPut.Flag("concurrency", "number of concurrent HTTP calls to retrieve remote resources").Default(strconv.Itoa(runtime.NumCPU())).Int()
	putFile := blobPut.Arg("url-or-file", "a url or file to import").Required().String()
	putDs := blobPut.Arg("dataset", "the path to import to").Required().String()

	blobGet := blob.Command("export", "exports a blob from a dataset")
	getDs := blobGet.Arg("dataset", "the dataset to export").Required().String()
	getPath := blobGet.Arg("file", "an optional file to save the blob to").String()

	return blob, func(input string) int {
		switch input {
		case blobPut.FullCommand():
			return nomsBlobPut(*putFile, *putDs, *concurrency)
		case blobGet.FullCommand():
			return nomsBlobGet(*getDs, *getPath)
		}
		d.Panic("notreached")
		return 1
	}
}
