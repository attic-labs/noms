// Copyright 2019 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/json"
)

func nomsJSON(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	// noms json in <db-spec> <file-or->
	// noms json out <path-spec> <file-or->
	jsonCmd := noms.Command("json", "Import or export JSON.")

	jsonIn := jsonCmd.Command("in", "imports data into Noms from JSON")
	structsIn := jsonIn.Flag("structs", "JSON objects will be imported to structs, otherwise maps").Bool()
	toDB := jsonIn.Arg("to", "Database spec to import to").Required().String()
	fromFile := jsonIn.Arg("from", "File to import from, or '@' to import from stdin").Required().String()

	jsonOut := jsonCmd.Command("out", "exports data from Noms to JSON")
	structsOut := jsonOut.Flag("structs", "Enable export of Noms structs (to JSON objects)").Default("false").Bool()
	fromPath := jsonOut.Arg("path", "Absolute path to value to export").Required().String()
	toFile := jsonOut.Arg("to", "File to export to, or '@' to export to stdout").Required().String()
	indent := jsonOut.Flag("indent", "Number of spaces to indent when pretty-printing").Default("\t").String()

	return jsonCmd, func(input string) int {
		switch input {
		case jsonIn.FullCommand():
			return nomsJSONIn(*fromFile, *toDB, json.FromOptions{Structs: *structsIn})
		case jsonOut.FullCommand():
			return nomsJSONOut(*fromPath, *toFile, json.ToOptions{Lists: true, Maps: true, Sets: true, Structs: *structsOut, Indent: *indent})
		}
		d.Panic("notreached")
		return 1
	}
}

func nomsJSONIn(from, to string, opts json.FromOptions) int {
	cfg := config.NewResolver()
	db, err := cfg.GetDatabase(to)
	d.CheckErrorNoUsage(err)

	var r io.ReadCloser
	if from == "@" {
		r = os.Stdin
	} else {
		r, err = os.Open(from)
		d.CheckErrorNoUsage(err)
	}
	defer r.Close()

	v, err := json.FromJSON(r, db, opts)
	d.CheckErrorNoUsage(err)

	ref := db.WriteValue(v)
	db.Flush()
	fmt.Printf("#%s\n", ref.TargetHash().String())
	return 0
}

func nomsJSONOut(from, to string, opts json.ToOptions) int {
	cfg := config.NewResolver()
	_, val, err := cfg.GetPath(from)
	d.CheckErrorNoUsage(err)

	var w io.WriteCloser
	if to == "@" {
		w = os.Stdout
	} else {
		w, err = os.Create(to)
		d.CheckErrorNoUsage(err)
	}
	defer w.Close()

	err = json.ToJSON(val, w, opts)
	d.CheckErrorNoUsage(err)
	return 0
}
