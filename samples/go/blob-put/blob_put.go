// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"os"

	"runtime"

	"io"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/verbose"
	flag "github.com/juju/gnuflag"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <file> <dataset>\n", os.Args[0])
		flag.PrintDefaults()
	}

	var concurrencyArg = flag.Int("concurrency", runtime.NumCPU(), "number of concurrent HTTP calls to retrieve remote resources")

	spec.RegisterDatabaseFlags(flag.CommandLine)
	verbose.RegisterVerboseFlags(flag.CommandLine)
	profile.RegisterProfileFlags(flag.CommandLine)

	flag.Parse(true)

	if len(flag.Args()) != 2 {
		d.CheckError(errors.New("expected file and dataset flags"))
	}

	filePath := flag.Arg(0)
	if filePath == "" {
		d.CheckErrorNoUsage(errors.New("Empty file path"))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		d.CheckError(errors.New("couldn't stat file"))
	}

	defer profile.MaybeStartProfile().Stop()

	fileSize := info.Size()
	chunkSize := fileSize / int64(*concurrencyArg)
	if chunkSize < (1 << 20) {
		chunkSize = 1 << 20
	}

	readers := make([]io.Reader, fileSize/chunkSize)
	for i := 0; i < len(readers); i++ {
		r, err := os.Open(filePath)
		d.CheckErrorNoUsage(err)
		defer r.Close()
		r.Seek(int64(i)*chunkSize, 0)
		limit := chunkSize
		if i == len(readers)-1 {
			limit += fileSize % chunkSize
		}
		lr := io.LimitReader(r, limit) // lastCh
		readers[i] = lr
	}

	cfg := config.NewResolver()
	db, ds, err := cfg.GetDataset(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create dataset: %s\n", err)
		return
	}
	defer db.Close()

	profile.RegisterProfileFlags(flag.CommandLine)

	blob := types.NewStreamingBlob(db, readers...)

	_, err = db.CommitValue(ds, blob)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error committing: %s\n", err)
		return
	}
}
