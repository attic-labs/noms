// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/progressreader"
	"github.com/attic-labs/noms/go/util/status"
	humanize "github.com/dustin/go-humanize"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <dataset> <file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	spec.RegisterDatabaseFlags(flag.CommandLine)
	flag.Parse()

	if len(flag.Args()) != 2 {
		d.CheckError(errors.New("expected dataset and file flags"))
	}

	var blob types.Blob
	path := flag.Arg(0)
	if db, val, err := spec.GetPath(path); err != nil {
		d.CheckErrorNoUsage(err)
	} else if val == nil {
		d.CheckErrorNoUsage(fmt.Errorf("No value at %s", path))
	} else if b, ok := val.(types.Blob); !ok {
		d.CheckErrorNoUsage(fmt.Errorf("Value at %s is not a blob", path))
	} else {
		defer db.Close()
		blob = b
	}

	filePath := flag.Arg(1)
	if filePath == "" {
		d.CheckErrorNoUsage(errors.New("Empty file path"))
	}

	// Note: overwrites any existing file.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	d.CheckErrorNoUsage(err)
	defer file.Close()

	progReader := progressreader.New(blob.Reader())

	ticker := status.NewTicker()
	go func() {
		expected := humanize.Bytes(blob.Len())
		start := time.Now()
		for range ticker.C {
			elapsed := time.Since(start).Seconds()
			seen := progReader.Seen()
			rate := uint64(float64(seen) / elapsed)
			status.Printf("%s of %s written in %ds (%s/s)...", humanize.Bytes(seen), expected, int(elapsed), humanize.Bytes(rate))
		}
	}()

	io.Copy(file, progReader)
	ticker.Stop()
	status.Done()
}
