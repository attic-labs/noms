// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
)

// NomsConfig - the noms config command
func NomsConfig(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	configCmd := noms.Command("config", "Prints the active configuration if a .nomsconfig file is present")

	return configCmd, func() int {
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
