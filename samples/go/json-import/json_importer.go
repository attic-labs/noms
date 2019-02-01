// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/jsontonoms"
	"github.com/attic-labs/noms/go/util/progressreader"
	"github.com/attic-labs/noms/go/util/status"
	"github.com/attic-labs/noms/go/util/verbose"
	humanize "github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')

	source := kingpin.Arg("source", "file or url to import").Required().String()
	destDB := kingpin.Arg("dest-db", "database to import to").Required().String()
	quiet := kingpin.Flag("quiet", "don't print out import progress").Short('q').Bool()

	verbose.RegisterVerboseFlags(flag.CommandLine)

	kingpin.Parse()

	cfg := config.NewResolver()
	db, err := cfg.GetDatabase(*destDB)
	d.CheckError(err)
	defer db.Close()

	var r io.Reader
	if strings.HasPrefix(*source, "http") {
		res, err := http.Get(*source)
		if err != nil {
			log.Fatalf("Error fetching %s: %+v\n", *source, err)
		} else if res.StatusCode != 200 {
			log.Fatalf("Error fetching %s: %s\n", *source, res.Status)
		}
		defer res.Body.Close()
		r = res.Body
	} else {
		// assume it's a file
		f, err := os.Open(*source)
		if err != nil {
			log.Fatalf("Invalid URL %s - does not start with 'http' and isn't local file either. fopen error: %s", *source, err)
		}

		r = f
	}

	var jsonObject interface{}
	start := time.Now()
	r = progressreader.New(r, func(seen uint64) {
		elapsed := time.Since(start).Seconds()
		rate := uint64(float64(seen) / elapsed)
		if !*quiet {
			status.Printf("%s decoded in %ds (%s/s)...", humanize.Bytes(seen), int(elapsed), humanize.Bytes(rate))
		}
	})
	err = json.NewDecoder(r).Decode(&jsonObject)
	if err != nil {
		log.Fatalln("Error decoding JSON: ", err)
	}
	if !*quiet {
		status.Done()
	}

	ref := db.WriteValue(jsontonoms.NomsValueFromDecodedJSON(db, jsonObject, true))
	db.Flush()
	fmt.Fprintf(os.Stdout, "#%s\n", ref.TargetHash().String())
}
