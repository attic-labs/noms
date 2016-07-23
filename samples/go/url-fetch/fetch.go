// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/progressreader"
	"github.com/attic-labs/noms/go/util/status"
	human "github.com/dustin/go-humanize"
)

func main() {
	comment := flag.String("comment", "", "comment to add to commit's meta data")
	spec.RegisterDatabaseFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Fetches a URL (or file) into a noms blob\n\nUsage: %s <dataset> <url-or-local-path>:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		d.CheckErrorNoUsage(errors.New("expected dataset and url arguments"))
	}

	ds, err := spec.GetDataset(flag.Arg(0))
	d.CheckErrorNoUsage(err)
	defer ds.Database().Close()

	src := flag.Arg(1)

	var fileOrUrl string
	var body io.Reader
	var contentLength int64

	if strings.HasPrefix(src, "http") {
		resp, err := http.Get(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not fetch url %s, error: %s\n", src, err)
			return
		}

		switch resp.StatusCode / 100 {
		case 4, 5:
			fmt.Fprintf(os.Stderr, "Could not fetch url %s, error: %d (%s)\n", src, resp.StatusCode, resp.Status)
			return
		}

		body = resp.Body
		contentLength = resp.ContentLength
		fileOrUrl = "url"
	} else {
		// assume it's a file
		f, err := os.Open(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid URL %s - does not start with 'http' and isn't local file either. fopen error: %s", src, err)
			return
		}

		s, err := f.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not stat file %s: %s", src, err)
			return
		}

		body = f
		contentLength = int64(s.Size())
		fileOrUrl = "file"
	}

	pr := progressreader.New(body)

	ticker := status.NewTicker()
	go func() {
		var expected string
		if contentLength < 0 {
			expected = "(unknown)"
		} else {
			expected = human.Bytes(uint64(contentLength))
		}
		start := time.Now()
		for range ticker.C {
			printProgress(start, pr.Seen(), expected)
		}
	}()

	b := types.NewStreamingBlob(pr, ds.Database())
	ticker.Stop()
	status.Done()

	mi := metaInfoForCommit(fileOrUrl, src, *comment)
	ds, err = ds.Commit(b, dataset.CommitOptions{Meta: mi})
	if err != nil {
		d.Chk.Equal(datas.ErrMergeNeeded, err)
		fmt.Fprintf(os.Stderr, "Could not commit, optimistic concurrency failed.")
	} else {
		fmt.Println("Done")
	}
}

func metaInfoForCommit(fileOrUrl, source, comment string) types.Struct {
	date := time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	metaValues := map[string]types.Value{
		"date":    types.String(date),
		fileOrUrl: types.String(source),
	}
	if comment != "" {
		metaValues["comment"] = types.String(comment)
	}
	return types.NewStruct("Meta", metaValues)
}

func printProgress(start time.Time, seen uint64, expected string) {
	elapsed := time.Now().Sub(start).Seconds()
	rate := uint64(float64(seen) / elapsed)
	status.Printf("%s of %s written in %ds (%s/s)...", human.Bytes(seen), expected, int(elapsed), human.Bytes(rate))
}
