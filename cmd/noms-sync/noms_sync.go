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

	"github.com/attic-labs/noms/go/d"
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

	hasStatus := false
	printProgress := func(doneCount, knownCount uint64) {
		if knownCount == 1 {
			// It's better to print "up to date" than "0% (0/1); 100% (1/1)".
			return
		}
		pct := 100.0 * (float64(doneCount) / float64(knownCount))
		status.Printf("Preparing Commit: %.2f%% (%d/%d chunks)", pct, doneCount, knownCount)
		hasStatus = true
	}

	err = d.Try(func() {
		defer profile.MaybeStartProfile().Stop()

		var err error
		sinkDataset, err = sinkDataset.Pull(sourceStore, types.NewRef(sourceObj), int(*p), printProgress)
		d.PanicIfError(err)
	})

	if err != nil {
		log.Fatal(err)
	} else if hasStatus {
		status.Done()
	} else {
		fmt.Println(flag.Arg(1), "is up to date.")
	}
}
