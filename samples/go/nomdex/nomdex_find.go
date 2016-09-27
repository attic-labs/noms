// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/outputpager"
	flag "github.com/juju/gnuflag"
)

var longHelp = `Find retrieves and prints objects that satisfy the 'query' argument.

Indexes are built using the 'nomdex up' command. Once built, the indexes can be referenced
in the 'query' arg to select objects matching certain criteria. For example, if there are
objects in the database that contain a personId and a gender field, 'nomdex up' can scan all
the objects in a given dataset and build an index on the specified field with the following
commands:
   nomdex up --by gender --in-path <dsSpec> --out-ds gender-index
   nomdex up --by personId --in-path <dsSpec> --out-ds personId-index

Once these indexes are built, objects can be retrieved quickly and efficiently using the
nomdex query language. For example, the followign query could be used to find all people
with with a personId between 1 and 2000 and who are female:
    nomdex find '(personId >= 0 and personId <= 2000) and gender = "female"

The next command would retrieve all people objects that were either male or had an personId
greater than 2000:
    nomdex find 'gender = "male" or personId > 2000'

The query language is simple. It currently supports the following relational operators:
    <, <=, >, >=, =, !=
Relational expressions are always of the form:
    <index> <relational operator> <constant>   e.g. personId >= 2000.

Indexes are the name given by the --out-ds argument in the 'nomdex up' command. Constants are
either "strings" (in quotes) or numbers (e.g. 3, 3000, -2, -2.5, 3.147, etc).

Relational expressions can be combined using the "and" and "or" operators. Parentheses can
be used to ensure that the evaluation is done in the desired order.
`

var find = &util.Command{
	Run:       runFind,
	UsageLine: "find --db <database spec> <query>",
	Short:     "Print objects in index that satisfy 'query'",
	Long:      longHelp,
	Flags:     setupFindFlags,
	Nargs:     1,
}

var dbPath = ""

func setupFindFlags() *flag.FlagSet {
	flagSet := flag.NewFlagSet("find", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "db", "", "database containing index")
	outputpager.RegisterOutputpagerFlags(flagSet)
	return flagSet
}

func runFind(args []string) int {
	query := args[0]
	if dbPath == "" {
		fmt.Fprintf(os.Stderr, "Missing required 'index' arg\n")
		flag.Usage()
		return 1
	}

	db, err := spec.GetDatabase(dbPath)
	if printError(err, "Unable to open database\n\terror: ") {
		return 1
	}
	defer db.Close()

	im := &indexManager{db: db, indexes: map[string]types.Map{}}
	expr, err := parseQuery(query, im)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return 1
	}

	pgr := outputpager.Start()
	defer pgr.Stop()

	iter := expr.iterator(im)
	cnt := 0
	if iter != nil {
		for v := iter.Next(); v != nil; v = iter.Next() {
			types.WriteEncodedValue(pgr.Writer, v)
			fmt.Fprintf(pgr.Writer, "\n")
			cnt++
		}
	}
	fmt.Fprintf(pgr.Writer, "Found %d objects\n", cnt)

	return 0
}

func printObjects(w io.Writer, index types.Map, ranges queryRangeSlice) {
	cnt := 0
	first := true
	printObjectForRange := func(index types.Map, r queryRange) {
		index.IterFrom(r.lower.value, func(k, v types.Value) bool {
			if first && r.lower.value != nil && !r.lower.include && r.lower.value.Equals(k) {
				return false
			}
			if r.upper.value != nil {
				if !r.upper.include && r.upper.value.Equals(k) {
					return true
				}
				if r.upper.value.Less(k) {
					return true
				}
			}
			s := v.(types.Set)
			s.IterAll(func(v types.Value) {
				types.WriteEncodedValue(w, v)
				fmt.Fprintf(w, "\n")
				cnt++
			})
			return false
		})
	}
	for _, r := range ranges {
		printObjectForRange(index, r)
	}
	fmt.Fprintf(w, "Found %d objects\n", cnt)
}

func openIndex(idxName string, im *indexManager) error {
	if _, hasIndex := im.indexes[idxName]; hasIndex {
		return nil
	}

	ds := im.db.GetDataset(idxName)
	commit, ok := ds.MaybeHead()
	if !ok {
		return fmt.Errorf("index '%s' not found", idxName)
	}

	index, ok := commit.Get(datas.ValueField).(types.Map)
	if !ok {
		return fmt.Errorf("Value of commit at '%s' is not a valid index", idxName)
	}

	// Todo: make this type be Map<String | Number>, Set<Value>> once Issue #2326 gets resolved and
	// IsSubtype() returns the correct value.
	typ := types.MakeMapType(
		types.MakeUnionType(types.StringType, types.NumberType),
		types.ValueType)

	if !types.IsSubtype(typ, index.Type()) {
		return fmt.Errorf("%s does not point to a suitable index type:", idxName)
	}

	im.indexes[idxName] = index
	return nil
}
