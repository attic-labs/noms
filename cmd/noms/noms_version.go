// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/attic-labs/noms/go/constants"
)

var nomsVersion = &NomsCommand{
	Run:       runVersion,
	UsageLine: "version ",
	Short:     "Display noms version",
	Long: `
		version prints the noms data version and build identifier 
	`,
	Flag:    flag.NewFlagSet("version", flag.ExitOnError),
	NumArgs: 0,
}

func runVersion(args []string) int {
	fmt.Fprintf(os.Stdout, "version: %v\n", constants.NomsVersion)
	fmt.Fprintf(os.Stdout, "built from %v\n", constants.NomsGitSHA)
	return 0
}
