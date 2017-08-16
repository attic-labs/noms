// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/diff"
	"github.com/attic-labs/noms/go/util/outputpager"
)

func nomsDiff(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	diffCmd := noms.Command("diff", `Shows the difference between two objects

See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object arguments.
`)
	summarize := diffCmd.Flag("summarize", "Writes a summary of the changes instead").Short('s').Bool()
	AddVerboseFlags(diffCmd)
	// TODO: deal with paged output
	// outputpager.RegisterOutputpagerFlags(diffFlagSet)

	object1 := diffCmd.Arg("object1", "").Required().String()
	object2 := diffCmd.Arg("object2", "").Required().String()

	return diffCmd, func() int {
		ApplyVerbosity()
		cfg := config.NewResolver()
		db1, value1, err := cfg.GetPath(*object1)
		d.CheckErrorNoUsage(err)
		if value1 == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", *object1))
		}
		defer db1.Close()

		db2, value2, err := cfg.GetPath(*object2)
		d.CheckErrorNoUsage(err)
		if value2 == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", *object2))
		}
		defer db2.Close()

		if *summarize {
			diff.Summary(value1, value2)
			return 0
		}

		pgr := outputpager.Start()
		defer pgr.Stop()

		diff.PrintDiff(pgr.Writer, value1, value2, false)
		return 0
	}
}
