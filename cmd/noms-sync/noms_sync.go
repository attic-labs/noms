// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/status"
)

var (
	p = flag.Uint("p", 512, "parallelism")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Moves datasets between or within databases\n\n")
		fmt.Fprintf(os.Stderr, "noms sync [options] <source-object> <dest-dataset>\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor detailed information on spelling objects and datasets, see: at https://github.com/attic-labs/noms/blob/master/doc/spelling.md.\n\n")
	}

	spec.RegisterDatabaseFlags()
	flag.Parse()

	if flag.NArg() != 2 {
		d.CheckError(errors.New("expected a source object and destination dataset"))
	}

	sourceStore, sourceObj, err := spec.GetPath(flag.Arg(0))
	d.CheckError(err)
	defer sourceStore.Close()

	sinkDataset, err := spec.GetDataset(flag.Arg(1))
	d.CheckError(err)
	defer sinkDataset.Database().Close()

	progressCh := make(chan datas.PullProgress)
	lastProgressCh := make(chan datas.PullProgress)

	go func() {
		var last datas.PullProgress

		for info := range progressCh {
			if info.KnownCount == 1 {
				// It's better to print "up to date" than "0% (0/1); 100% (1/1)".
				continue
			}

			last = info
			if status.WillPrint() {
				pct := 100.0 * float64(info.DoneCount) / float64(info.KnownCount)
				status.Printf("Syncing - %.2f%% (%d/%d chunks)", pct, info.DoneCount, info.KnownCount)
			}
		}

		lastProgressCh <- last
	}()

	start := time.Now()
	err = d.Try(func() {
		defer profile.MaybeStartProfile().Stop()

		var err error
		sinkDataset, err = sinkDataset.Pull(sourceStore, types.NewRef(sourceObj), int(*p), progressCh)
		d.PanicIfError(err)
	})

	if err != nil {
		log.Fatal(err)
	}

	close(progressCh)
	if last := <-lastProgressCh; last.DoneCount > 0 {
		status.Printf("Done - Synced %d chunks in %s", last.DoneCount, time.Since(start).String())
		status.Done()
	} else {
		fmt.Println(flag.Arg(1), "is up to date.")
	}
}
