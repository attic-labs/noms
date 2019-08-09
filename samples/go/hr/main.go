// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
)

func main() {
	app := kingpin.New("hr", "")
	var dsStr = app.Flag("ds", "noms dataset to read/write from").Required().String()

	addCmd := app.Command("add-person", "Add a new person")
	id := addCmd.Arg("id", "unique person id").Required().Uint64()
	name := addCmd.Arg("name", "person name").Required().String()
	title := addCmd.Arg("title", "person title").Required().String()

	app.Command("list-persons", "list current persons")

	verbose.RegisterVerboseFlags(app)
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	cfg := config.NewResolver()
	db, ds, err := cfg.GetDataset(*dsStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create dataset: %s\n", err)
		return
	}
	defer db.Close()

	switch cmd {
	case "add-person":
		addPerson(db, ds, *id, *name, *title)
	case "list-persons":
		listPersons(ds)
	}
}

type Person struct {
	Name, Title string
	Id          uint64
}

func addPerson(db datas.Database, ds datas.Dataset, id uint64, name, title string) {
	np, err := marshal.Marshal(db, Person{name, title, id})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	_, err = db.CommitValue(ds, getPersons(ds).Edit().Set(types.Number(id), np).Map())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error committing: %s\n", err)
		return
	}
}

func listPersons(ds datas.Dataset) {
	d := getPersons(ds)
	if d.Empty() {
		fmt.Println("No people found")
		return
	}

	d.IterAll(func(k, v types.Value) {
		var p Person
		err := marshal.Unmarshal(v, &p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("%s (id: %d, title: %s)\n", p.Name, p.Id, p.Title)
	})
}

func getPersons(ds datas.Dataset) types.Map {
	hv, ok := ds.MaybeHeadValue()
	if ok {
		return hv.(types.Map)
	}
	return types.NewMap(ds.Database())
}
