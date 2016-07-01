// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"os"
)

var commands = []*nomsCommand{
	nomsDiff,
	nomsDs,
	nomsLog,
	nomsServe,
	nomsShow,
	nomsSync,
	nomsVersion,
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.Flag.Usage = func() { cmd.Usage() }

			cmd.Flag.Parse(args[1:])
			args = cmd.Flag.Args()
			if cmd.Nargs != 0 && len(args) < cmd.Nargs {
				cmd.Usage()
			}
			os.Exit(cmd.Run(args))
		}
	}

	fmt.Fprintf(os.Stderr, "noms: unknown command %q\n", args[0])
	usage()
}
