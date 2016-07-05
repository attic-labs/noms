// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/attic-labs/noms/cmd/noms/diff"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/util/outputpager"
)

const (
	addPrefix = "+   "
	subPrefix = "-   "
)

var nomsDiff = &nomsCommand{
	Run:       runDiff,
	UsageLine: "diff <object1> <object2>",
	Short:     "Shows the difference between two objects",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object arguments.",
	Flag:      flag.NewFlagSet("diff", flag.ExitOnError),
	Nargs:     2,
}

func runDiff(args []string) int {
	db1, value1, err := spec.GetPath(args[0])
	d.CheckErrorNoUsage(err)
	if value1 == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", args[0]))
	}
	defer db1.Close()

	db2, value2, err := spec.GetPath(args[1])
	d.CheckErrorNoUsage(err)
	if value2 == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", args[1]))
	}
	defer db2.Close()

	waitChan := outputpager.PageOutput(!*outputpager.NoPager)

	w := bufio.NewWriter(os.Stdout)
	diff.Diff(w, value1, value2)
	fmt.Fprintf(w, "\n")
	w.Flush()

	if waitChan != nil {
		os.Stdout.Close()
		<-waitChan
	}
	return 0
}
