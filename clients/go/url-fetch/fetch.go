package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/noms/clients/go/flags"
	"github.com/attic-labs/noms/clients/go/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/types"
	"github.com/attic-labs/noms/util/progressreader"
	"github.com/attic-labs/noms/util/status"
	human "github.com/dustin/go-humanize"
)

var (
	start time.Time
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Fetches a URL into a noms blob\n\nUsage: %s <dataset> <url>:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flags.RegisterDatabaseFlags()
	flag.Parse()

	if flag.NArg() != 2 {
		util.CheckError(errors.New("expected dataset and url arguments"))
	}

	spec, err := flags.ParseDatasetSpec(flag.Arg(0))
	if err != nil {
		util.CheckError(err)
	}
	ds, err := spec.Dataset()
	util.CheckError(err)

	url := flag.Arg(1)
	start = time.Now()

	var pr io.Reader

	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not fetch url %s, error: %s\n", url, err)
			return
		}

		switch resp.StatusCode / 100 {
		case 4, 5:
			fmt.Fprintf(os.Stderr, "Could not fetch url %s, error: %d (%s)\n", url, resp.StatusCode, resp.Status)
			return
		}

		pr = progressreader.New(resp.Body, getStatusPrinter(resp.ContentLength))
	} else {
		// assume it's a file
		f, err := os.Open(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid URL %s - does not start with 'http' and isn't local file either. fopen error: %s", url, err)
			return
		}

		s, err := f.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not stat file %s: %s", url, err)
			return
		}

		pr = progressreader.New(f, getStatusPrinter(s.Size()))
	}

	b := types.NewBlob(pr)
	ds, err = ds.Commit(b)
	if err != nil {
		d.Chk.Equal(datas.ErrMergeNeeded, err)
		fmt.Fprintf(os.Stderr, "Could not commit, optimistic concurrency failed.")
		return
	}

	status.Done()
	fmt.Println("Done")
}

func getStatusPrinter(expectedLen int64) progressreader.Callback {
	return func(seenLen uint64) {
		var expected string
		if expectedLen < 0 {
			expected = "(unknown)"
		} else {
			expected = human.Bytes(uint64(expectedLen))
		}

		elapsed := time.Now().Sub(start)
		rate := uint64(float64(seenLen) / elapsed.Seconds())

		status.Printf("%s of %s written in %ds (%s/s)...",
			human.Bytes(seenLen),
			expected,
			uint64(elapsed.Seconds()),
			human.Bytes(rate))
	}
}
