// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/attic-labs/noms/go/util/verbose"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var actions = []string{
	"interacting with",
	"poking at",
	"goofing with",
	"dancing with",
	"playing with",
	"contemplation of",
	"showing off",
	"jiggerypokery of",
	"singing to",
	"nomming on",
}

type CommandHandler func() (exitCode int)

var (
	verboseFlag *bool
	quietFlag   *bool
)

// addVerboseFlags adds --verbose and --quiet flags to the passed command
func addVerboseFlags(cmd *kingpin.CmdClause) (verboseFlag *bool, quietFlag *bool) {
	verboseFlag = cmd.Flag("verbose", "show more").Short('v').Bool()
	quietFlag = cmd.Flag("quiet", "show less").Short('q').Bool()
	return
}

// applyVerbosity - run when commands are invoked to apply the verbosity arguments configured in addVerboseFlags
func applyVerbosity() {
	verbose.SetVerbose(*verboseFlag)
	verbose.SetQuiet(*quietFlag)
}

// AddDatabaseArg adds a "database" arg to the passed command
func AddDatabaseArg(cmd *kingpin.CmdClause) (arg *string) {
	return cmd.Arg("database", "a noms database path").Required().String() // TODO: custom parser for noms db URL?
}

type NomsCommand func(*kingpin.Application) (*kingpin.CmdClause, CommandHandler)

// Commands, in order of preference
var commands = []NomsCommand{
	nomsCommit,
	nomsConfig,
	nomsDiff,
	nomsDs,
	nomsLog,
	nomsMerge,
	nomsRoot,
	nomsServe,
	nomsShow,
	nomsSync,
	nomsVersion,
}

func main() {
	// allow short (-h) help
	kingpin.CommandLine.HelpFlag.Short('h')

	i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(actions))
	noms := kingpin.New("noms", fmt.Sprintf(`Noms is a tool for %s Noms data.`, actions[i]))

	handlers := map[string]CommandHandler{}

	// install handlers
	for _, cmdFunction := range commands {
		command, handler := cmdFunction(noms)
		handlers[command.FullCommand()] = handler
	}

	// parse our input
	input := kingpin.MustParse(noms.Parse(os.Args[1:]))

	// Don't prefix log messages with timestamp when running interactively
	log.SetFlags(0)

	if handler := handlers[input]; handler != nil {
		handler()
	}
}
