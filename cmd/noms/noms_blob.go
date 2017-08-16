// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"runtime"
	"strconv"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/profile"
	"gopkg.in/alecthomas/kingpin.v2"
)

func nomsBlob(noms *kingpin.Application) (*kingpin.CmdClause, commandHandler) {
	blob := noms.Command("blob", "interact with blobs in a dataset")

	blobPut := blob.Command("put", "imports a blob to a dataset")
	putVerbose, putQuiet := addVerboseFlags(blobPut)
	profile.AddProfileFlags(blobPut)
	concurrency := blobPut.Flag("concurrency", "number of concurrent HTTP calls to retrieve remote resources").Default(strconv.Itoa(runtime.NumCPU())).Int()
	putFile := blobPut.Arg("url-or-file", "a url or file to import").Required().String()
	putDs := blobPut.Arg("dataset", "the path to import to").Required().String()

	blobGet := blob.Command("export", "exports a blob from a dataset")
	getDs := blobGet.Arg("dataset", "the dataset to export").Required().String()
	getPath := blobGet.Arg("file", "an optional file to save the blob to").String()
	getVerbose, getQuiet := addVerboseFlags(blobGet)
	profile.AddProfileFlags(blobGet)

	return blob, func(input string) int {
		profile.ApplyProfileFlags()
		switch input {
		case blobPut.FullCommand():
			applyVerbosity(putVerbose, putQuiet)
			return nomsBlobPut(*putFile, *putDs, *concurrency)
		case blobGet.FullCommand():
			applyVerbosity(getVerbose, getQuiet)
			return nomsBlobGet(*getDs, *getPath)
		}
		d.Panic("notreached")
		return 1
	}
}
