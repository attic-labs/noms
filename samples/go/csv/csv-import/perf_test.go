// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/perf/suite"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/csv"
	"github.com/attic-labs/testify/assert"
	humanize "github.com/dustin/go-humanize"
)

// CSV perf suites require the testdata directory to be checked out at $GOPATH/src/github.com/attic-labs/testdata (i.e. ../testdata relative to the noms directory).

type perfSuite struct {
	suite.PerfSuite
	csvImportExe string
}

func (s *perfSuite) SetupSuite() {
	// Trick the temp file logic into creating a unique path for the csv-import binary.
	f := s.TempFile("csv-import.perf_test")
	f.Close()
	os.Remove(f.Name())

	s.csvImportExe = f.Name()
	err := exec.Command("go", "build", "-o", s.csvImportExe, "github.com/attic-labs/noms/samples/go/csv/csv-import").Run()
	assert.NoError(s.T, err)
}

func (s *perfSuite) Test01ImportSfCrimeBlobFromTestdata() {
	assert := s.NewAssert()

	raw := s.openGlob(path.Join(s.Testdata, "sf-crime", "2016-07-28.*"))
	defer s.closeGlob(raw)

	blob := types.NewBlob(io.MultiReader(raw...))
	fmt.Fprintf(s.W, "\tsf-crime is %s\n", humanize.Bytes(blob.Len()))

	// CommitValue is much faster the first time, so pause the test while it's happening.
	s.Pause(func() {
		ds := dataset.NewDataset(s.Database, "sf-crime/raw")
		_, err := ds.CommitValue(blob)
		assert.NoError(err)
	})
}

func (s *perfSuite) Test02ImportSfCrimeCSVFromBlob() {
	s.execCsvImportExe("sf-crime")
}

func (s *perfSuite) Test03ImportSfRegisteredBusinessesFromBlobAsMap() {
	assert := s.NewAssert()

	// Pause because blob import speed is already tested in ImportSfCrimeBlobFromTestdata, and besides, this is a much smaller dataset.
	s.Pause(func() {
		f, err := os.Open(path.Join(s.Testdata, "sf-registered-businesses", "2016-07-25.csv"))
		assert.NoError(err)
		defer f.Close()

		blob := types.NewBlob(f)
		fmt.Fprintf(s.W, "\tsf-reg-bus is %s\n", humanize.Bytes(blob.Len()))

		ds := dataset.NewDataset(s.Database, "sf-reg-bus/raw")
		_, err = ds.CommitValue(blob)
		assert.NoError(err)
	})

	s.execCsvImportExe("sf-reg-bus", "--dest-type", "map:0")
}

func (s *perfSuite) execCsvImportExe(dsName string, args ...string) {
	assert := s.NewAssert()

	blobSpec := fmt.Sprintf("%s::%s/raw.value", s.DatabaseSpec, dsName)
	destSpec := fmt.Sprintf("%s::%s", s.DatabaseSpec, dsName)
	args = append(args, "-p", blobSpec, destSpec)
	importCmd := exec.Command(s.csvImportExe, args...)
	importCmd.Stdout = s.W
	importCmd.Stderr = os.Stderr

	assert.NoError(importCmd.Run())
}

func (s *perfSuite) TestParseSfCrime() {
	assert := s.NewAssert()

	raw := s.openGlob(path.Join(s.Testdata, "sf-crime", "2016-07-28.*"))
	defer s.closeGlob(raw)

	reader := csv.NewCSVReader(io.MultiReader(raw...), ',')
	for {
		_, err := reader.Read()
		if err != nil {
			assert.Equal(io.EOF, err)
			break
		}
	}
}

// openGlob opens all files that match `pattern`. Large CSV files in testdata are broken up into foo.a, foo.b, etc to get around GitHub file size restrictions.
func (s *perfSuite) openGlob(pattern string) (readers []io.Reader) {
	assert := s.NewAssert()

	s.Pause(func() {
		glob, err := filepath.Glob(pattern)
		assert.NoError(err)
		readers = make([]io.Reader, len(glob))
		for i, m := range glob {
			r, err := os.Open(m)
			assert.NoError(err)
			readers[i] = r
		}
	})
	return
}

// closeGlob closes `readers`. Intended to be used after `openGlob`.
func (s *perfSuite) closeGlob(readers []io.Reader) {
	assert := s.NewAssert()

	s.Pause(func() {
		for _, r := range readers {
			assert.NoError(r.(io.ReadCloser).Close())
		}
	})
}

func TestPerf(t *testing.T) {
	suite.Run("csv-import", t, &perfSuite{})
}
