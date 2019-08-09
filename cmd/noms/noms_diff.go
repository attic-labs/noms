// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/diff"
	"github.com/attic-labs/noms/go/util/outputpager"
)

func nomsDiff(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("diff", "Shows the difference between two values.")
	stat := cmd.Flag("stat", "writes a summary of the changes instead").Bool()
	o1 := cmd.Arg("val1", "first value - see Spelling Values at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()
	o2 := cmd.Arg("val2", "second value - see Spelling Values at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()
	outputpager.RegisterOutputpagerFlags(cmd)

	return cmd, func(input string) int {
		cfg := config.NewResolver()
		db1, value1, err := cfg.GetPath(*o1)
		d.CheckErrorNoUsage(err)
		if value1 == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Value not found: %s", *o1))
		}
		defer db1.Close()

		db2, value2, err := cfg.GetPath(*o2)
		d.CheckErrorNoUsage(err)
		if value2 == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Value not found: %s", *o2))
		}
		defer db2.Close()

		if *stat {
			diff.Summary(value1, value2)
			return 0
		}

		pgr := outputpager.Start()
		defer pgr.Stop()

		diff.PrintDiff(pgr.Writer, value1, value2, false)
		return 0
	}
}
