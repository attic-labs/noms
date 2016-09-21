// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/noms/cmd/util"
	flag "github.com/juju/gnuflag"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/d"
)

var nomsConfig = &util.Command{
	Run:       runConfig,
	UsageLine: "config ",
	Short:     "Display noms config info",
	Long:      "if a .nom/config file is present, config prints the active settings",
	Flags:     setupConfigFlags,
	Nargs:     0,
}

func setupConfigFlags() *flag.FlagSet {
	return flag.NewFlagSet("config", flag.ExitOnError)
}

func runConfig(args []string) int {
	c, err := spec.FindNomsConfig()
	if err == spec.NoConfig {
		fmt.Fprintf(os.Stdout, "no config active\n")
	} else {
		d.CheckError(err)
		fmt.Fprintf(os.Stdout, "%s\n", c.String())
	}
	return 0
}
