// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/verbose"
)

func main() {
	registerUpdate()
	registerFind()
	verbose.RegisterVerboseFlags(kingpin.CommandLine)
	profile.RegisterProfileFlags(kingpin.CommandLine)

	switch kingpin.Parse() {
	case "up":
		runUpdate()
	case "find":
		runFind()
	}
}

func printError(err error, msgAndArgs ...interface{}) bool {
	if err != nil {
		err := d.Unwrap(err)
		switch len(msgAndArgs) {
		case 0:
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		case 1:
			fmt.Fprintf(os.Stderr, "%s%s\n", msgAndArgs[0], err)
		default:
			format, ok := msgAndArgs[0].(string)
			if ok {
				s1 := fmt.Sprintf(format, msgAndArgs[1:]...)
				fmt.Fprintf(os.Stderr, "%s%s\n", s1, err)
			} else {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			}
		}
	}
	return err != nil
}
