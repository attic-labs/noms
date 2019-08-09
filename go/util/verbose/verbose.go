// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package verbose

import (
	"log"

	"github.com/attic-labs/kingpin"
)

var (
	verbose bool
	quiet   bool
)

// RegisterVerboseFlags registers -v|--verbose flags for general usage
func RegisterVerboseFlags(app *kingpin.Application) {
	// Must reset globals because under test this can get called multiple times.
	verbose = false
	quiet = false
	app.Flag("verbose", "show more").Short('v').BoolVar(&verbose)
	app.Flag("quite", "show less").Short('q').BoolVar(&quiet)
}

// Verbose returns True if the verbose flag was set
func Verbose() bool {
	return verbose
}

func SetVerbose(v bool) {
	verbose = v
}

// Quiet returns True if the verbose flag was set
func Quiet() bool {
	return quiet
}

func SetQuiet(q bool) {
	quiet = q
}

// Log calls Printf(format, args...) iff Verbose() returns true.
func Log(format string, args ...interface{}) {
	if Verbose() {
		if len(args) > 0 {
			log.Printf(format+"\n", args...)
		} else {
			log.Println(format)
		}
	}
}
