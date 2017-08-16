// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"runtime"
	"strconv"

	"github.com/attic-labs/noms/go/util/profile"
	"gopkg.in/alecthomas/kingpin.v2"
)

func nomsBlob(noms *kingpin.Application) (*kingpin.CmdClause, kCommandHandler) {
	blob := noms.Command("blob", "interact with blobs in a dataset")

	blobImport := blob.Command("import", "imports a blob to a dataset")
	addVerboseFlags(blobImport)
	profile.KAddProfileFlags(blobImport)
	concurrency := blobImport.Flag("concurrency", "number of concurrent HTTP calls to retrieve remote resources").Default(strconv.Itoa(runtime.NumCPU())).Int()
	importFile := blobImport.Arg("url-or-file", "a url or file to import").Required().String()
	importDs := blobImport.Arg("dataset", "the path to import to").Required().String()

	blobExport := blob.Command("export", "exports a blob from a dataset")
	exportDs := blobExport.Arg("dataset", "the dataset to export").Required().String()
	exportPath := blobExport.Arg("file", "an optional file to save the blob to").String()
	addVerboseFlags(blobExport)
	profile.KAddProfileFlags(blobExport)

	return blob, func(input string) int {
		applyVerbosity()
		profile.KApplyProfileFlags()
		switch input {
		case blobImport.FullCommand():
			return nomsBlobImport(*importFile, *importDs, *concurrency)
		case blobExport.FullCommand():
			return nomsBlobExport(*exportDs, *exportPath)
		}
		// unreachable
		return 1
	}
}
