// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package verbose

import (
	"log"
)

var (
	verbose bool
	quiet   bool
)

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
