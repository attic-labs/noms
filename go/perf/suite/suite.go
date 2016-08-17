// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package suite implements a performance test suite for Noms, intended for measuring and reporting long running tests.
//
// Usage is similar to testify's suite:
//
// 1. Define a test suite struct which inherits from `suite.PerfSuite`.
// 2. Define methods on that struct that start with the word "Test", optionally followed by digits, then followed a non-empty capitalized string.
// 3. Call `suite.Run` with an instance of that struct.
// 4. Run `go test` with the `-perf <path to noms db>` flag. Use `-perf.repeat` to set how many times tests are repeated.
//
// Test results are written to Noms, along with a soup of the environment they were recorded in.
//
// Test names are derived from that "non-empty capitalized string": `"Test"` is ommitted because it's redundant, and leading digits are omitted to allow for manual test ordering. For example:
//
// ```
// > cat ./samples/go/csv/csv-import/perf_test.go
// type perfSuite {
//   suite.PerfSuite
// }
//
// func (s *perfSuite) TestFoo() { ... }
// func (s *perfSuite) TestZoo() { ... }
// func (s *perfSuite) Test01Qux() { ... }
// func (s *perfSuite) Test02Bar() { ... }
//
// func TestPerf(t *testing.T) {
//   suite.Run("csv-import", t, &perfSuite{})
// }
//
// > go test -v ./samples/go/csv/... -perf http://localhost:8000 -perf.repeat 3
// (perf) RUN(1/3) Test01Qux (recorded as "Qux")
// (perf) PASS:    Test01Qux (5s, paused 15s, total 20s)
// (perf) RUN(1/3) Test02Bar (recorded as "Bar")
// (perf) PASS:    Test02Bar (15s, paused 2s, total 17s)
// (perf) RUN(1/3) TestFoo (recorded as "Foo")
// (perf) PASS:    TestFoo (10s, paused 1s, total 11s)
// (perf) RUN(1/3) TestZoo (recorded as "Zoo")
// (perf) PASS:    TestZoo (1s, paused 42s, total 43s)
// ...
//
// > noms show http://localhost:8000::csv-import
// {
//   environment: ...
//   tests: [{
//     "Bar": {elapsed: 15s, paused: 2s,  total: 17s},
//     "Foo": {elapsed: 10s, paused: 1s,  total: 11s},
//     "Qux": {elapsed: 5s,  paused: 15s, total: 20s},
//     "Zoo": {elapsed: 1s,  paused: 42s, total: 43s},
//   }, ...]
//   ...
// }
// ```
package suite

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

var (
	perfFlag        = flag.String("perf", "", "The database to write perf tests to. If this isn't specified, perf tests are skipped. If you want a dry run, use \"mem\" as a database")
	perfRepeatFlag  = flag.Int("perf.repeat", 1, "The number of times to repeat each perf test")
	testNamePattern = regexp.MustCompile("^Test[0-9]*([A-Z].*$)")
)

// PerfSuite is the core of the perf testing suite. See package documentation for details.
type PerfSuite struct {
	// T is the `testing.T` instance set when the suite is passed into `Run`.
	T *testing.T

	// W is the `io.Writer` to write test output, which only outputs if the verbose flag is set.
	W io.Writer

	// AtticLabs is the path to the attic-labs directory (e.g. /path/to/go/src/github.com/attic-labs).
	AtticLabs string

	// Database is a Noms database that tests can use for reading and writing. State is persisted across a single run of a suite.
	Database datas.Database

	// DatabaseSpec is the Noms spec of `Database` (typically a localhost URL).
	DatabaseSpec string

	tempFiles []*os.File
	paused    time.Duration
}

type perfSuiteT interface {
	Suite() *PerfSuite
}

type timeInfo struct {
	elapsed, paused, total time.Duration
}

type testRun map[string]timeInfo

type nopWriter struct{}

func (r nopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

// Run runs `suiteT` and writes results to dataset `datasetID` in the database given by the `-perf` command line flag.
func Run(datasetID string, t *testing.T, suiteT perfSuiteT) {
	if *perfFlag == "" {
		return
	}

	assert := assert.New(t)

	// Piggy-back off the go test -v flag.
	verboseFlag := flag.Lookup("test.v")
	assert.NotNil(verboseFlag)
	verbose := verboseFlag.Value.(flag.Getter).Get().(bool)

	suite := suiteT.Suite()
	suite.T = t
	if verbose {
		suite.W = os.Stdout
	} else {
		suite.W = nopWriter{}
	}

	gopath := os.Getenv("GOPATH")
	assert.True(gopath != "")
	suite.AtticLabs = path.Join(gopath, "src", "github.com", "attic-labs")

	// This is the database the perf test results are written to.
	db, err := spec.GetDatabase(*perfFlag)
	assert.NoError(err)

	// This is the temporary database for tests to use.
	server := datas.NewRemoteDatabaseServer(chunks.NewMemoryStore(), 0)
	portChan := make(chan int)
	server.Ready = func() { portChan <- server.Port() }
	go server.Run()

	port := <-portChan
	suite.DatabaseSpec = fmt.Sprintf("http://localhost:%d", port)
	suite.Database = datas.NewRemoteDatabase(suite.DatabaseSpec, "")

	// List of test runs, each a map of test name => timing info.
	testRuns := make([]testRun, *perfRepeatFlag)

	defer func() {
		for _, f := range suite.tempFiles {
			os.Remove(f.Name())
		}

		runs := make([]types.Value, *perfRepeatFlag)
		for i, run := range testRuns {
			timesSlice := []types.Value{}
			for name, info := range run {
				timesSlice = append(timesSlice, types.String(name), types.NewStruct("", types.StructData{
					"elapsed": types.Number(info.elapsed.Nanoseconds()),
					"paused":  types.Number(info.paused.Nanoseconds()),
					"total":   types.Number(info.total.Nanoseconds()),
				}))
			}
			runs[i] = types.NewMap(timesSlice...)
		}

		record := types.NewStruct("", map[string]types.Value{
			"environment":     suite.getEnvironment(),
			"nomsVersion":     types.String(suite.getGitHead(path.Join(suite.AtticLabs, "noms"))),
			"testdataVersion": types.String(suite.getGitHead(path.Join(suite.AtticLabs, "testdata"))),
			"runs":            types.NewList(runs...),
		})

		ds := dataset.NewDataset(db, datasetID)
		var err error
		ds, err = ds.CommitValue(record)
		assert.NoError(err)
		assert.NoError(db.Close())
		server.Stop()
	}()

	for runIdx := 0; runIdx < *perfRepeatFlag; runIdx++ {
		run := testRun{}
		testRuns[runIdx] = run

		for t, mIdx := reflect.TypeOf(suiteT), 0; mIdx < t.NumMethod(); mIdx++ {
			m := t.Method(mIdx)

			parts := testNamePattern.FindStringSubmatch(m.Name)
			if parts == nil {
				continue
			}

			recordName := parts[1]
			if verbose {
				fmt.Printf("(perf) RUN(%d/%d) %s (as \"%s\")\n", runIdx+1, *perfRepeatFlag, m.Name, recordName)
			}

			start := time.Now()
			suite.paused = 0

			err := callSafe(m.Name, m.Func, suiteT)

			total := time.Since(start)
			elapsed := total - suite.paused

			if verbose && err == nil {
				fmt.Printf("(perf) PASS:    %s (%s, paused for %s, total %s)\n", m.Name, elapsed, suite.paused, total)
			} else if err != nil {
				fmt.Printf("(perf) FAIL:    %s (%s, paused for %s, total %s)\n", m.Name, elapsed, suite.paused, total)
				fmt.Println(err)
			}

			run[recordName] = timeInfo{elapsed, suite.paused, total}
		}
	}
}

func (suite *PerfSuite) Suite() *PerfSuite {
	return suite
}

// NewAssert returns the `assert.Assertions` instance for this test.
func (suite *PerfSuite) NewAssert() *assert.Assertions {
	return assert.New(suite.T)
}

// TempFile creates a temporary file, which will be automatically cleaned up by the perf test suite.
func (suite *PerfSuite) TempFile() *os.File {
	f, err := ioutil.TempFile("", "perf")
	assert.NoError(suite.T, err)
	suite.tempFiles = append(suite.tempFiles, f)
	return f
}

// Pause pauses the test timer while `fn` is executing. Useful for omitting long setup code (e.g. copying files) from the test elapsed time.
func (suite *PerfSuite) Pause(fn func()) {
	start := time.Now()
	fn()
	suite.paused += time.Since(start)
}

func callSafe(name string, fun reflect.Value, args ...interface{}) (err interface{}) {
	defer func() {
		if r := recover(); r != nil {
			err = r
		}
	}()
	funArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		funArgs[i] = reflect.ValueOf(arg)
	}
	fun.Call(funArgs)
	return
}

func (suite *PerfSuite) getEnvironment() types.Struct {
	assert := suite.NewAssert()

	cpuInfos, err := cpu.Info()
	assert.NoError(err)
	cpus := make([]types.Value, 0, 2*len(cpuInfos))
	for i, c := range cpuInfos {
		cm, err := marshal.Marshal(c)
		assert.NoError(err)
		cpus = append(cpus, types.Number(i), cm)
	}

	vmStat, err := mem.VirtualMemory()
	assert.NoError(err)
	mem, err := marshal.Marshal(*vmStat)
	assert.NoError(err)

	hostInfo, err := host.Info()
	assert.NoError(err)
	host, err := marshal.Marshal(*hostInfo)
	assert.NoError(err)

	partitionStats, err := disk.Partitions(false)
	assert.NoError(err)
	partitions := make([]types.Value, 0, 2*len(partitionStats))
	diskUsages := make([]types.Value, 0, 2*len(partitionStats))
	for _, p := range partitionStats {
		pm, err := marshal.Marshal(p)
		assert.NoError(err)
		partitions = append(partitions, types.String(p.Device), pm)
		du, err := disk.Usage(p.Mountpoint)
		assert.NoError(err)
		dum, err := marshal.Marshal(*du)
		assert.NoError(err)
		diskUsages = append(diskUsages, types.String(p.Mountpoint), dum)
	}

	return types.NewStruct("", types.StructData{
		"diskUsages": types.NewMap(diskUsages...),
		"cpus":       types.NewMap(cpus...),
		"mem":        mem,
		"host":       host,
		"partitions": types.NewMap(partitions...),
	})
}

func (suite *PerfSuite) getGitHead(dir string) string {
	stdout := &bytes.Buffer{}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Stdout = stdout
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}
