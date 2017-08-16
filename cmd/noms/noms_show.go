// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/outputpager"
)

// NomsShow - the noms show command
func NomsShow(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	show := noms.Command("show", `Shows a serialization of a Noms object

See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object argument.
`)

	raw := show.Flag("raw", "If true, dumps the raw binary version of the data").Bool()
	stats := show.Flag("stats", "If true, reports statistics related to the value").Bool()
	AddVerboseFlags(show)
	object := show.Arg("object", "a noms object").Required().String()

	// TODO: outputpager stuff?

	return show, func() int {
		showRaw := *raw
		showStats := *stats
		ApplyVerbosity()
		cfg := config.NewResolver()
		database, value, err := cfg.GetPath(*object)
		d.CheckErrorNoUsage(err)
		defer database.Close()

		if value == nil {
			fmt.Fprintf(os.Stderr, "Object not found: %s\n", *object)
			return 0
		}

		if showRaw && showStats {
			fmt.Fprintln(os.Stderr, "--raw and --stats are mutually exclusive")
			return 0
		}

		if showRaw {
			ch := types.EncodeValue(value)
			buf := bytes.NewBuffer(ch.Data())
			_, err = io.Copy(os.Stdout, buf)
			d.CheckError(err)
			return 0
		}

		if showStats {
			types.WriteValueStats(os.Stdout, value, database)
			return 0
		}

		pgr := outputpager.Start()
		defer pgr.Stop()

		types.WriteEncodedValue(pgr.Writer, value)
		fmt.Fprintln(pgr.Writer)
		return 0
	}
}
