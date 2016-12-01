// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/testify/assert"
	"github.com/dustin/go-humanize"
	flag "github.com/juju/gnuflag"
)

var (
	count    = flag.Int("c", 10, "Number of iterations to run")
	dataSize = flag.Uint64("data", 4096, "MiB of data to test with")
	mtMiB    = flag.Uint64("mem", 64, "Size in MiB of memTable")
	useNBS   = flag.String("useNBS", "", "Existing Database to use for not-WriteNovel benchmarks")
	toNBS    = flag.String("toNBS", "", "Write to an NBS store in the given directory")
	toFile   = flag.String("toFile", "", "Write to a file in the given directory")
)

type panickingBencher struct {
	n int
}

func (pb panickingBencher) Errorf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (pb panickingBencher) N() int {
	return pb.n
}

func (pb panickingBencher) ResetTimer() {}
func (pb panickingBencher) StartTimer() {}
func (pb panickingBencher) StopTimer()  {}

func main() {
	profile.RegisterProfileFlags(flag.CommandLine)
	flag.Parse(true)

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	pb := panickingBencher{*count}

	src, err := getInput((*dataSize) * humanize.MiByte)
	d.PanicIfError(err)
	defer src.Close()

	bufSize := (*mtMiB) * humanize.MiByte

	open := newNullBlockStore
	wrote := false
	var writeDB func()
	var refresh func() blockStore
	if *toNBS != "" || *toFile != "" {
		var dir string
		if *toNBS != "" {
			dir = makeTempDir(*toNBS, pb)
			open = func() blockStore {
				return nbs.NewBlockStore(dir, bufSize)
			}
		} else if *toFile != "" {
			dir = makeTempDir(*toFile, pb)
			open = func() blockStore {
				f, err := ioutil.TempFile(dir, "")
				d.Chk.NoError(err)
				return newFileBlockStore(f)
			}
		}
		defer os.RemoveAll(dir)
		writeDB = func() { wrote = ensureNovelWrite(wrote, open, src, pb) }
		refresh = func() blockStore {
			os.RemoveAll(dir)
			os.MkdirAll(dir, 0777)
			return open()
		}
	} else {
		if *useNBS != "" {
			open = func() blockStore {
				return nbs.NewBlockStore(*useNBS, bufSize)
			}
		}
		writeDB = func() {}
		refresh = func() blockStore { panic("WriteNovel unsupported with --useLDB and --useNBS") }
	}

	benchmarks := []struct {
		name  string
		setup func()
		run   func()
	}{
		{"WriteNovel", func() {}, func() { wrote = benchmarkNovelWrite(refresh, src, pb) }},
		{"WriteDuplicate", writeDB, func() { benchmarkNoRefreshWrite(open, src, pb) }},
		{"ReadSequential", writeDB, func() { benchmarkRead(open, src.GetHashes(), src, pb) }},
		{"ReadManySequential", writeDB, func() { benchmarkReadMany(open, src.GetHashes(), src, 1<<8, 6, pb) }},
		{"ReadHashOrder", writeDB, func() {
			ordered := src.GetHashes()
			sort.Sort(ordered)
			benchmarkRead(open, ordered, src, pb)
		}},
	}
	w := 0
	for _, bm := range benchmarks {
		if len(bm.name) > w {
			w = len(bm.name)
		}
	}
	defer profile.MaybeStartProfile().Stop()
	for _, bm := range benchmarks {
		if matched, _ := regexp.MatchString(flag.Arg(0), bm.name); matched {
			trialName := fmt.Sprintf("%dMiB/%sbuffer/%-[3]*s", *dataSize, humanize.IBytes(bufSize), w, bm.name)
			bm.setup()
			dur := time.Duration(0)
			var trials []time.Duration
			for i := 0; i < *count; i++ {
				d.Chk.NoError(dropCache())
				src.PrimeFilesystemCache()

				t := time.Now()
				bm.run()
				trialDur := time.Since(t)
				trials = append(trials, trialDur)
				dur += trialDur
			}
			fmt.Printf("%s\t%d\t%ss/iter %v\n", trialName, *count, humanize.FormatFloat("", (dur/time.Duration(*count)).Seconds()), formatTrials(trials))
		}
	}
}

func makeTempDir(tmpdir string, t assert.TestingT) (dir string) {
	dir, err := ioutil.TempDir(tmpdir, "")
	assert.NoError(t, err)
	return
}

func formatTrials(trials []time.Duration) (formatted []string) {
	for _, trial := range trials {
		formatted = append(formatted, humanize.FormatFloat("", trial.Seconds()))
	}
	return
}
