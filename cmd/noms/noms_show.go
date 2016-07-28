// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/outputpager"
)

var nomsShow = &nomsCommand{
	Run:       runShow,
	UsageLine: "show <object>",
	Short:     "Shows a serialization of a Noms object",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object argument.",
	Flags:     setupShowFlags,
	Nargs:     1,
}

func setupShowFlags() *flag.FlagSet {
	showFlagSet := flag.NewFlagSet("show", flag.ExitOnError)
	outputpager.RegisterOutputpagerFlags(showFlagSet)
	return showFlagSet
}

func runShow(args []string) int {
	database, value, err := spec.GetPath(args[0])
	d.CheckErrorNoUsage(err)
	defer database.Close()

	if value == nil {
		fmt.Fprintf(os.Stderr, "Object not found: %s\n", args[0])
		return 0
	}

	var w io.Writer
	if pager := outputpager.NewOrNil(); pager != nil {
		w = pager.Writer
		defer pager.Stop()
		go pager.RunAndExit()
	} else {
		bw := bufio.NewWriter(os.Stdout)
		defer bw.Flush()
		w = bw
	}

	types.WriteEncodedValueWithTags(w, value)
	w.Write([]byte{'\n'})
	return 0
}
