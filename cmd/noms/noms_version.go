// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/constants"
)

func nomsVersion(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	version := noms.Command("version", "Print the noms version")

	return version, func() int {
		fmt.Fprintf(os.Stdout, "format version: %v\n", constants.NomsVersion)
		fmt.Fprintf(os.Stdout, "built from %v\n", constants.NomsGitSHA)
		return 0
	}
}
