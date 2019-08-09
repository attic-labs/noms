// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/samples/go/csv"
)

func main() {
	app := kingpin.New("csv-analyze", "")
	// Actually the delimiter uses runes, which can be multiple characters long.
	// https://blog.golang.org/strings
	delimiter := app.Flag("delimiter", "field delimiter for csv file, must be exactly one character long.").String()
	header := app.Flag("header", "header row. If empty, we'll use the first row of the file").String()
	skipRecords := app.Flag("skip-records", "number of records to skip at beginning of file").Uint()
	detectColumnTypes := app.Flag("detect-column-types", "detect column types by analyzing a portion of csv file").Bool()
	detectPrimaryKeys := app.Flag("detect-pk", "detect primary key candidates by analyzing a portion of csv file").Bool()
	numSamples := app.Flag("num-samples", "number of records to use for samples").Default("1000000").Int()
	numFieldsInPK := app.Flag("num-fields-pk", "maximum number of columns to consider when detecting PKs").Default("3").Int()
	r := app.Arg("filepath", "csv file to analyze").Required().File()

	profile.RegisterProfileFlags(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	defer profile.MaybeStartProfile().Stop()
	defer (*r).Close()

	comma, err := csv.StringToRune(*delimiter)
	d.CheckError(err)

	cr := csv.NewCSVReader(*r, comma)
	csv.SkipRecords(cr, *skipRecords)

	var headers []string
	if *header == "" {
		headers, err = cr.Read()
		d.PanicIfError(err)
	} else {
		headers = strings.Split(*header, string(comma))
	}

	kinds := []types.NomsKind{}
	if *detectColumnTypes {
		kinds = csv.GetSchema(cr, *numSamples, len(headers))
		fmt.Fprintf(os.Stdout, "%s\n", strings.Join(csv.KindsToStrings(kinds), ","))
	}

	if *detectPrimaryKeys {
		pks := csv.FindPrimaryKeys(cr, *numSamples, *numFieldsInPK, len(headers))
		for _, pk := range pks {
			fmt.Fprintf(os.Stdout, "%s\n", strings.Join(csv.GetFieldNamesFromIndices(headers, pk), ","))
		}
	}
}
