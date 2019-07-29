// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
	"gopkg.in/attic-labs/noms.v7/go/config"
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/types"
	"gopkg.in/attic-labs/noms.v7/go/util/profile"
	"gopkg.in/attic-labs/noms.v7/go/util/progressreader"
	"gopkg.in/attic-labs/noms.v7/go/util/status"
	"gopkg.in/attic-labs/noms.v7/go/util/verbose"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <dataset> [<file>]\n", os.Args[0])
		flag.PrintDefaults()
	}

	verbose.RegisterVerboseFlags(flag.CommandLine)
	profile.RegisterProfileFlags(flag.CommandLine)

	flag.Parse(true)

	if flag.NArg() != 1 && flag.NArg() != 2 {
		d.CheckError(errors.New("expected dataset and optional file flag"))
	}

	cfg := config.NewResolver()
	var blob types.Blob
	path := flag.Arg(0)
	if db, val, err := cfg.GetPath(path); err != nil {
		d.CheckErrorNoUsage(err)
	} else if val == nil {
		d.CheckErrorNoUsage(fmt.Errorf("No value at %s", path))
	} else if b, ok := val.(types.Blob); !ok {
		d.CheckErrorNoUsage(fmt.Errorf("Value at %s is not a blob", path))
	} else {
		defer db.Close()
		blob = b
	}

	defer profile.MaybeStartProfile().Stop()

	filePath := flag.Arg(1)
	if filePath == "" {
		blob.Copy(os.Stdout)
		return
	}

	// Note: overwrites any existing file.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	d.CheckErrorNoUsage(err)
	defer file.Close()

	start := time.Now()
	expected := humanize.Bytes(blob.Len())

	// Create a pipe so that we can connect a progress reader
	preader, pwriter := io.Pipe()

	go func() {
		blob.Copy(pwriter)
		pwriter.Close()
	}()

	blobReader := progressreader.New(preader, func(seen uint64) {
		elapsed := time.Since(start).Seconds()
		rate := uint64(float64(seen) / elapsed)
		status.Printf("%s of %s written in %ds (%s/s)...", humanize.Bytes(seen), expected, int(elapsed), humanize.Bytes(rate))
	})

	io.Copy(file, blobReader)
	status.Done()
}
