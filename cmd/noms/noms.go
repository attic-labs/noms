// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/cmd/noms/splore"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/verbose"
)

var kingpinCommands = []util.KingpinCommand{
	nomsBlob,
	nomsCommit,
	nomsConfig,
	nomsDiff,
	nomsDs,
	nomsList,
	nomsLog,
	nomsMerge,
	nomsJSON,
	nomsMap,
	nomsRoot,
	nomsServe,
	nomsSet,
	nomsShow,
	nomsStats,
	nomsStruct,
	nomsSync,
	splore.Cmd,
	nomsVersion,
}

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

func usageString() string {
	i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(actions))
	return fmt.Sprintf(`Noms is a tool for %s Noms data.`, actions[i])
}

func main() {
	// allow short (-h) help
	kingpin.EnableFileExpansion = false
	kingpin.CommandLine.HelpFlag.Short('h')
	noms := kingpin.New("noms", usageString())

	// global flags
	profile.RegisterProfileFlags(noms)
	verbose.RegisterVerboseFlags(noms)

	handlers := map[string]util.KingpinHandler{}

	// install kingpin handlers
	for _, cmdFunction := range kingpinCommands {
		command, handler := cmdFunction(noms)
		handlers[command.FullCommand()] = handler
	}

	input := kingpin.MustParse(noms.Parse(os.Args[1:]))

	if handler := handlers[strings.Split(input, " ")[0]]; handler != nil {
		handler(input)
	}
}
