// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/attic-labs/noms/go/util/outputpager"
)

func nomsShow(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("show", "Print Noms values.")
	showRaw := cmd.Flag("raw", "dump the value in binary format").Bool()
	showStats := cmd.Flag("stats", "report statics related to the value").Bool()
	tzName := cmd.Flag("tz", "display formatted date comments in specified timezone, must be: local or utc").Default("local").String()
	path := cmd.Arg("path", "value to display - see Spelling Values at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()

	return cmd, func(_ string) int {
		cfg := config.NewResolver()
		database, value, err := cfg.GetPath(*path)
		d.CheckErrorNoUsage(err)
		defer database.Close()

		if value == nil {
			fmt.Fprintf(os.Stderr, "Value not found: %s\n", *path)
			return 0
		}

		if *showRaw && *showStats {
			fmt.Fprintln(os.Stderr, "--raw and --stats are mutually exclusive")
			return 0
		}

		if *showRaw {
			ch := types.EncodeValue(value)
			buf := bytes.NewBuffer(ch.Data())
			_, err = io.Copy(os.Stdout, buf)
			d.CheckError(err)
			return 0
		}

		if *showStats {
			types.WriteValueStats(os.Stdout, value, database)
			return 0
		}

		tz, _ := locationFromTimezoneArg(*tzName, nil)
		datetime.RegisterHRSCommenter(tz)

		pgr := outputpager.Start()
		defer pgr.Stop()

		types.WriteEncodedValue(pgr.Writer, value)
		fmt.Fprintln(pgr.Writer)
		return 0
	}
}
