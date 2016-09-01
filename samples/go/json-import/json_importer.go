// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/util/jsontonoms"
	"github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

func newProgressReader(r io.Reader) io.Reader {
	duration := time.Second / 10
	timer := time.NewTimer(duration)
	progress := make(chan int)
	done := make(chan bool)
	go func() {
		start := time.Now()
		completed := int64(0)
		reportProgress := func() {
			elapsed := time.Since(start)
			rate := float64(completed) * (float64(time.Second) / float64(elapsed))
			clearLine := "\x1b[2K\r"
			fmt.Fprintf(os.Stderr, "%s %s decoded at %s/S", clearLine, humanize.BigBytes(big.NewInt(int64(completed))), humanize.BigBytes(big.NewInt(int64(rate))))
		}
		for {
			select {
			case p := <-progress:
				completed += int64(p)
			case <-timer.C:
				timer.Reset(duration)
				reportProgress()
			case <-done:
				timer.Stop()
				reportProgress()
				fmt.Fprintf(os.Stderr, "\n")
				return
			}
		}
	}()
	return &progressReader{r, progress, done}
}

type progressReader struct {
	R        io.Reader // underlying reader
	progress chan<- int
	done     chan<- bool
}

func (l *progressReader) Read(p []byte) (n int, err error) {
	n, err = l.R.Read(p)
	l.progress <- n
	if err == io.EOF {
		l.done <- true
		close(l.progress)
	}
	return
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <url> <dataset>\n", os.Args[0])
		flag.PrintDefaults()
	}

	spec.RegisterDatabaseFlags(flag.CommandLine)
	flag.Parse(true)

	if len(flag.Args()) != 2 {
		d.CheckError(errors.New("expected url and dataset flags"))
	}

	ds, err := spec.GetDataset(flag.Arg(1))
	d.CheckError(err)

	url := flag.Arg(0)
	if url == "" {
		flag.Usage()
	}

	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching %s: %+v\n", url, err)
	} else if res.StatusCode != 200 {
		log.Fatalf("Error fetching %s: %s\n", url, res.Status)
	}
	defer res.Body.Close()

	var jsonObject interface{}
	err = json.NewDecoder(newProgressReader(res.Body)).Decode(&jsonObject)
	if err != nil {
		log.Fatalln("Error decoding JSON: ", err)
	}

	_, err = ds.CommitValue(jsontonoms.NomsValueFromDecodedJSON(jsonObject, true))
	d.PanicIfError(err)
	ds.Database().Close()
}
