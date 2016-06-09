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
	"runtime"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/flags"
	"github.com/attic-labs/noms/samples/go/util"
)

var (
	p         = flag.Uint("p", 512, "parallelism")
	clearLine = "\x1b[2K\r"
)

func main() {
	cpuCount := runtime.NumCPU()
	runtime.GOMAXPROCS(cpuCount)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Moves datasets between or within databases\n")
		fmt.Fprintln(os.Stderr, "noms sync [options] <source-object> <dest-dataset>\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor detailed information on spelling objects and datasets, see: at https://github.com/attic-labs/noms/blob/master/doc/spelling.md.\n\n")
	}

	flags.RegisterDatabaseFlags()
	flag.Parse()

	if flag.NArg() != 2 {
		util.CheckError(errors.New("expected a source object and destination dataset"))
	}

	sourceSpec, err := flags.ParsePathSpec(flag.Arg(0))
	util.CheckError(err)
	sourceStore, sourceObj, err := sourceSpec.Value()
	util.CheckError(err)
	defer sourceStore.Close()

	sinkSpec, err := flags.ParseDatasetSpec(flag.Arg(1))
	util.CheckError(err)

	sinkDataset, err := sinkSpec.Dataset()
	util.CheckError(err)
	defer sinkDataset.Database().Close()

	var count, total uint64

	err = d.Try(func() {
		if util.MaybeStartCPUProfile() {
			defer util.StopCPUProfile()
		}

		progressCallback := func(sofar, expect uint64) {
			if total > 0 {
				count += sofar
			}
			total += expect

			fmt.Printf("%s%d/%d", clearLine, count, total)
		}

		var err error
		sinkDataset, err = sinkDataset.Pull(sourceStore, types.NewRef(sourceObj), int(*p), progressCallback)
		fmt.Printf("\n")

		util.MaybeWriteMemProfile()
		d.Exp.NoError(err)
	})

	if err != nil {
		log.Fatal(err)
	}
}
