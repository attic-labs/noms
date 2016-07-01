// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type NomsCommand struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(args []string) int

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'n help' output.
	Short string

	// Long is the long message shown in the 'n help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag *flag.FlagSet

	NumArgs int
}

// Name returns the command's name: the first word in the usage line.
func (nc *NomsCommand) Name() string {
	name := nc.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (nc *NomsCommand) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n\n", nc.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(nc.Long))
	if nc.Flag != nil {
		fmt.Fprintf(os.Stderr, "\nflags:\n")
		nc.Flag.PrintDefaults()
	}
	os.Exit(2)
}
