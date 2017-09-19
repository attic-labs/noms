// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
)

func main() {
	sp, err := spec.ForDataset(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse spec: %s, error: %s\n", sp, err)
		os.Exit(1)
	}
	defer sp.Close()

	db := sp.GetDatabase()
	if headValue, ok := sp.GetDataset().MaybeHeadValue(); !ok {
		data := types.NewList(sp.GetDatabase(),
			newPerson("Rickon", true),
			newPerson("Bran", true),
			newPerson("Arya", false),
			newPerson("Sansa", false),
		)

		fmt.Fprintf(os.Stdout, "data type: %v\n", types.TypeOf(data).Describe())
		_, err = db.CommitValue(sp.GetDataset(), data)
		if err != nil {
			fmt.Fprint(os.Stderr, "Error commiting: %s\n", err)
			os.Exit(1)
		}
	} else {
		// type assertion to convert Head to List
		personList := headValue.(types.List)
		// type assertion to convert List Value to Struct
		personStruct := personList.Get(0).(types.Struct)
		// prints: Rickon
		fmt.Fprintf(os.Stdout, "given: %v\n", personStruct.Get("given"))
	}
}

func newPerson(givenName string, male bool) types.Struct {
	return types.NewStruct("Person", types.StructData{
		"given": types.String(givenName),
		"male":  types.Bool(male),
	})
}
