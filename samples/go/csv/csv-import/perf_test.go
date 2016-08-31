// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
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

	raw := s.readGlob(s.Testdata, "sf-crime", "2016-07-28.*")
	blob := types.NewBlob(raw)
	fmt.Fprintf(s.W, "\tsf-crime is %s\n", humanize.Bytes(blob.Len()))

	ds := dataset.NewDataset(s.Database, "sf-crime/raw")
	_, err := ds.CommitValue(blob)
	assert.NoError(err)
}

func (s *perfSuite) Test02ImportSfCrimeCSVFromBlob() {
	s.execCsvImportExe("sf-crime")
}

func (s *perfSuite) Test03ImportSfRegisteredBusinessesFromBlobAsMap() {
	assert := s.NewAssert()

	raw := s.readGlob(s.Testdata, "sf-registered-businesses", "2016-07-25.csv")
	blob := types.NewBlob(raw)
	fmt.Fprintf(s.W, "\tsf-reg-bus is %s\n", humanize.Bytes(blob.Len()))

	ds := dataset.NewDataset(s.Database, "sf-reg-bus/raw")
	_, err := ds.CommitValue(blob)
	assert.NoError(err)

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

	raw := s.readGlob(path.Join(s.Testdata, "sf-crime", "2016-07-28.*"))
	reader := csv.NewCSVReader(raw, ',')

	for {
		_, err := reader.Read()
		if err != nil {
			assert.Equal(io.EOF, err)
			break
		}
	}
}

// readGlob returns a bytes.Buffer containing the concatenation of all files
// that match `pattern`. Large CSV files in testdata are broken up into foo.a,
// foo.b, etc to get around GitHub file size restrictions.
func (s *perfSuite) readGlob(pattern ...string) *bytes.Buffer {
	assert := s.NewAssert()
	res := &bytes.Buffer{}

	s.Pause(func() {
		glob, err := filepath.Glob(path.Join(pattern...))
		assert.NoError(err)

		for _, m := range glob {
			f, err := os.Open(m)
			defer f.Close()
			assert.NoError(err)
			io.Copy(res, f)
		}
	})

	return res
}

func TestPerf(t *testing.T) {
	suite.Run("csv-import", t, &perfSuite{})
}
