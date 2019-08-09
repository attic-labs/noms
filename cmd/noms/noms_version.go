// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/constants"
)

func nomsVersion(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("version", "Displays the Noms version understood by this command.")

	return cmd, func(_ string) int {
		fmt.Fprintf(os.Stdout, "format version: %v\n", constants.NomsVersion)
		fmt.Fprintf(os.Stdout, "built from %v\n", constants.NomsGitSHA)
		return 0
	}
}
