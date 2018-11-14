// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/progressreader"
	"github.com/attic-labs/noms/go/util/status"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/attic-labs/noms/samples/go/csv"
	humanize "github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

const (
	destList = iota
	destMap  = iota
)

func main() {
	// Actually the delimiter uses runes, which can be multiple characters long.
	// https://blog.golang.org/strings
	delimiter := flag.String("delimiter", ",", "field delimiter for csv file, must be exactly one character long.")
	header := flag.String("header", "", "header row. If empty, we'll use the first row of the file")
	lowercase := flag.Bool("lowercase", false, "convert column names to lowercase (otherwise preserve the case in the resulting struct fields)")
	name := flag.String("name", "Row", "struct name. The user-visible name to give to the struct type that will hold each row of data.")
	columnTypes := flag.String("column-types", "", "a comma-separated list of types representing the desired type of each column. if absent all types default to be String")
	pathDescription := "noms path to blob to import"
	path := flag.String("path", "", pathDescription)
	flag.StringVar(path, "p", "", pathDescription)
	noProgress := flag.Bool("no-progress", false, "prevents progress from being output if true")
	destType := flag.String("dest-type", "list", "the destination type to import to. can be 'list' or 'map:<pk>', where <pk> is a list of comma-delimited column headers or indexes (0-based) used to uniquely identify a row")
	skipRecords := flag.Uint("skip-records", 0, "number of records to skip at beginning of file")
	limit := flag.Uint64("limit-records", math.MaxUint64, "maximum number of records to process")
	performCommit := flag.Bool("commit", true, "commit the data to head of the dataset (otherwise only write the data to the dataset)")
	append := flag.Bool("append", false, "append new data to list at head of specified dataset.")
	invert := flag.Bool("invert", false, "import rows in column major format rather than row major")
	spec.RegisterCommitMetaFlags(flag.CommandLine)
	verbose.RegisterVerboseFlags(flag.CommandLine)
	profile.RegisterProfileFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: csv-import [options] <csvfile> <dataset>\n\n")
		flag.PrintDefaults()
	}

	flag.Parse(true)

	var err error
	switch {
	case flag.NArg() == 0:
		err = errors.New("Maybe you put options after the dataset?")
	case flag.NArg() == 1 && *path == "":
		err = errors.New("If <csvfile> isn't specified, you must specify a noms path with -p")
	case flag.NArg() == 2 && *path != "":
		err = errors.New("Cannot specify both <csvfile> and a noms path with -p")
	case flag.NArg() > 2:
		err = errors.New("Too many arguments")
	case strings.HasPrefix(*destType, "map") && *append:
		err = errors.New("--append is only compatible with list imports")
	case strings.HasPrefix(*destType, "map") && *invert:
		err = errors.New("--invert is only compatible with list imports")
	}
	d.CheckError(err)

	defer profile.MaybeStartProfile().Stop()

	var r io.Reader
	var size uint64
	var filePath string
	var dataSetArgN int

	cfg := config.NewResolver()
	if *path != "" {
		db, val, err := cfg.GetPath(*path)
		d.CheckError(err)
		if val == nil {
			d.CheckError(fmt.Errorf("Path %s not found\n", *path))
		}
		blob, ok := val.(types.Blob)
		if !ok {
			d.CheckError(fmt.Errorf("Path %s not a Blob: %s\n", *path, types.EncodedValue(types.TypeOf(val))))
		}
		defer db.Close()
		preader, pwriter := io.Pipe()
		go func() {
			blob.Copy(pwriter)
			pwriter.Close()
		}()
		r = preader
		size = blob.Len()
		dataSetArgN = 0
	} else {
		filePath = flag.Arg(0)
		res, err := os.Open(filePath)
		d.CheckError(err)
		defer res.Close()
		fi, err := res.Stat()
		d.CheckError(err)
		r = res
		size = uint64(fi.Size())
		dataSetArgN = 1
	}

	if !*noProgress {
		r = progressreader.New(r, getStatusPrinter(size))
	}

	delim, err := csv.StringToRune(*delimiter)
	d.CheckErrorNoUsage(err)

	var dest int
	var strPks []string
	if *destType == "list" {
		dest = destList
	} else if strings.HasPrefix(*destType, "map:") {
		dest = destMap
		strPks = strings.Split(strings.TrimPrefix(*destType, "map:"), ",")
		if len(strPks) == 0 {
			fmt.Println("Invalid dest-type map: ", *destType)
			return
		}
	} else {
		fmt.Println("Invalid dest-type: ", *destType)
		return
	}

	cr := csv.NewCSVReader(r, delim)
	err = csv.SkipRecords(cr, *skipRecords)

	if err == io.EOF {
		err = fmt.Errorf("skip-records skipped past EOF")
	}
	d.CheckErrorNoUsage(err)

	var headers []string
	if *header == "" {
		headers, err = cr.Read()
		d.PanicIfError(err)
	} else {
		headers = strings.Split(*header, ",")
	}
	if *lowercase {
		for i, _ := range headers {
			headers[i] = strings.ToLower(headers[i])
		}
	}

	uniqueHeaders := make(map[string]bool)
	for _, header := range headers {
		uniqueHeaders[header] = true
	}
	if len(uniqueHeaders) != len(headers) {
		d.CheckErrorNoUsage(fmt.Errorf("Invalid headers specified, headers must be unique"))
	}

	kinds := []types.NomsKind{}
	if *columnTypes != "" {
		kinds = csv.StringsToKinds(strings.Split(*columnTypes, ","))
		if len(kinds) != len(uniqueHeaders) {
			d.CheckErrorNoUsage(fmt.Errorf("Invalid column-types specified, column types do not correspond to number of headers"))
		}
	}

	db, ds, err := cfg.GetDataset(flag.Arg(dataSetArgN))
	d.CheckError(err)
	defer db.Close()

	var value types.Value
	if dest == destMap {
		value = csv.ReadToMap(cr, *name, headers, strPks, kinds, db, *limit)
	} else if *invert {
		value = csv.ReadToColumnar(cr, *name, headers, kinds, db, *limit)
	} else {
		value = csv.ReadToList(cr, *name, headers, kinds, db, *limit)
	}

	if *performCommit {
		meta, err := spec.CreateCommitMetaStruct(ds.Database(), "", "", additionalMetaInfo(filePath, *path), nil)
		d.CheckErrorNoUsage(err)
		if *append {
			if headVal, present := ds.MaybeHeadValue(); present {
				switch headVal.Kind() {
				case types.ListKind:
					l, isList := headVal.(types.List)
					d.PanicIfFalse(isList)
					ref := db.WriteValue(value)
					value = l.Concat(ref.TargetValue(db).(types.List))
				case types.StructKind:
					hstr, isStruct := headVal.(types.Struct)
					d.PanicIfFalse(isStruct)
					d.PanicIfFalse(hstr.Name() == "Columnar")
					str := value.(types.Struct)
					hstr.IterFields(func(fieldname string, v types.Value) bool {
						hl := v.(types.Ref).TargetValue(db).(types.List)
						nl := str.Get(fieldname).(types.Ref).TargetValue(db).(types.List)
						l := hl.Concat(nl)
						r := db.WriteValue(l)
						str = str.Set(fieldname, r)

						return false
					})
					value = str
				default:
					d.Panic("append can only be used with list or columnar")
				}
			}
		}
		_, err = db.Commit(ds, value, datas.CommitOptions{Meta: meta})
		if !*noProgress {
			status.Clear()
		}
		d.PanicIfError(err)
	} else {
		ref := db.WriteValue(value)
		if !*noProgress {
			status.Clear()
		}
		fmt.Fprintf(os.Stdout, "#%s\n", ref.TargetHash().String())
	}
}

func additionalMetaInfo(filePath, nomsPath string) map[string]string {
	fileOrNomsPath := "inputPath"
	path := nomsPath
	if path == "" {
		path = filePath
		fileOrNomsPath = "inputFile"
	}
	return map[string]string{fileOrNomsPath: path}
}

func getStatusPrinter(expected uint64) progressreader.Callback {
	startTime := time.Now()
	return func(seen uint64) {
		percent := float64(seen) / float64(expected) * 100
		elapsed := time.Since(startTime)
		rate := float64(seen) / elapsed.Seconds()

		status.Printf("%.2f%% of %s (%s/s)...",
			percent,
			humanize.Bytes(expected),
			humanize.Bytes(uint64(rate)))
	}
}
