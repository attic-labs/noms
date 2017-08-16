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

// CommandHandler - a callback passed to commands
type CommandHandler func() (exitCode int)

var (
	verboseFlag *bool
	quietFlag   *bool
)

// AddVerboseFlags adds --verbose and --quiet flags to the passed command
func AddVerboseFlags(cmd *kingpin.CmdClause) (verboseFlag *bool, quietFlag *bool) {
	verboseFlag = cmd.Flag("verbose", "show more").Short('v').Bool()
	quietFlag = cmd.Flag("quiet", "show less").Short('q').Bool()
	return
}

// ApplyVerbosity - run when commands are invoked to apply the verbosity arguments configured in AddVerboseFlags
func ApplyVerbosity() {
	verbose.SetVerbose(*verboseFlag)
	verbose.SetQuiet(*quietFlag)
}

// AddDatabaseArg adds a "database" arg to the passed command
func AddDatabaseArg(cmd *kingpin.CmdClause) (arg *string) {
	return cmd.Arg("database", "a noms database path").Required().String() // TODO: custom parser for noms db URL?
}

// Commands, in order of preference
var commands = []func(*kingpin.Application) (*kingpin.CmdClause, CommandHandler){
	NomsCommit,
	NomsConfig,
	NomsDiff,
	NomsDs,
	NomsLog,
	NomsMerge,
	NomsRoot,
	NomsServe,
	NomsShow,
	NomsSync,
	NomsVersion,
}

func main() {
	// allow short (-h) help
	kingpin.CommandLine.HelpFlag.Short('h')

	// TODO: is there a way to dynamically generate help text other than re-initializing this every time?
	i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(actions))
	noms := kingpin.New("noms", fmt.Sprintf(`Noms is a tool for %s Noms data.`, actions[i]))

	handlers := make(map[string]CommandHandler)

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
