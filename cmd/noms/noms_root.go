// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	flag "github.com/juju/gnuflag"
)

var nomsRoot = &util.Command{
	Run:       runRoot,
	UsageLine: "root <db-spec>",
	Short:     "Get or set the current root hash of the entire database",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.",
	Flags:     setupRootFlags,
	Nargs:     1,
}

var updateRoot = ""

func setupRootFlags() *flag.FlagSet {
	flagSet := flag.NewFlagSet("root", flag.ExitOnError)
	flagSet.StringVar(&updateRoot, "update", "", "Replaces the entire database with the one with the given hash")
	return flagSet
}

func runRoot(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Not enough arguments")
		return 0
	}

	cfg := config.NewResolver()
	db, err := cfg.GetDatabase(args[0])
	d.CheckErrorNoUsage(err)
	defer db.Close()

	currRoot := db.ValidatingBatchStore().Root()

	if updateRoot == "" {
		fmt.Println(currRoot)
		return 1
	}

	fmt.Println(`WARNING

This operation replaces the entire database with the instance having the given
hash. The old database becomes eligible for GC.

ANYTHING NOT SAVED WILL BE LOST

Continue?
`)
	fmt.Scanln()

	if updateRoot[0] == '#' {
		updateRoot = updateRoot[1:]
	}
	h, ok := hash.MaybeParse(updateRoot)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Hash %s does not exist in database\n", h.String())
		return 1
	}
	ok = db.ValidatingBatchStore().UpdateRoot(h, currRoot)
	if !ok {
		fmt.Fprintln(os.Stderr, "Optimistic concurrency failure")
		return 1
	}

	fmt.Printf("Success. Previous root was: %s\n", currRoot)
	return 0
}
