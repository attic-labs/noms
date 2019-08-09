// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
)

func nomsConfig(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cfg := noms.Command("config", "Display noms config info.")

	return cfg, func(input string) int {
		c, err := config.FindNomsConfig()
		if err == config.NoConfig {
			fmt.Fprintf(os.Stdout, "no config active\n")
		} else {
			d.CheckError(err)
			fmt.Fprintf(os.Stdout, "%s\n", c.String())
		}
		return 0
	}
}
