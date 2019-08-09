// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"os"

	"github.com/attic-labs/kingpin"
	"github.com/dustin/go-humanize"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/nbs/benchmarks/gen"
)

const (
	KB               = uint64(1 << 10)
	MB               = uint64(1 << 20)
	averageChunkSize = 4 * KB
)

var (
	genSize    = kingpin.Flag("gen", "MiB of data to generate and chunk").Default("1024").Uint64()
	chunkInput = kingpin.Flag("chunk", "Treat arg as data file to chunk").Bool()
	fileName   = kingpin.Arg("file", "filename").String()
)

func main() {
	kingpin.Parse()

	var fd *os.File
	var err error
	if *chunkInput {
		fd, err = os.Open(*fileName)
		d.Chk.NoError(err)
		defer fd.Close()
	} else {
		fd, err = gen.OpenOrGenerateDataFile(*fileName, (*genSize)*humanize.MiByte)
		d.Chk.NoError(err)
		defer fd.Close()
	}

	cm := gen.OpenOrBuildChunkMap(*fileName+".chunks", fd)
	defer cm.Close()

	return
}
