// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/test_util"
	"github.com/attic-labs/testify/suite"
)

func TestCSVImporter(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	test_util.ClientTestSuite
}

func (s *testSuite) TestCSVImporter() {
	input, err := ioutil.TempFile(s.TempDir, "")
	d.Chk.NoError(err)
	defer input.Close()
	defer os.Remove(input.Name())

	_, err = input.WriteString("a,b\n")
	d.Chk.NoError(err)
	for i := 0; i < 100; i++ {
		_, err = input.WriteString(fmt.Sprintf("a%d,%d\n", i, i))
		d.Chk.NoError(err)
	}
	_, err = input.Seek(0, 0)
	d.Chk.NoError(err)

	setName := "csv"
	dataspec := test_util.CreateValueSpecString("ldb", s.LdbDir, setName)
	out := s.Run(main, []string{"-no-progress", "-column-types", "String,Number", dataspec, input.Name()})
	s.Equal("", out)

	cs := chunks.NewLevelDBStore(s.LdbDir, "", 1, false)
	ds := dataset.NewDataset(datas.NewDatabase(cs), setName)
	defer ds.Database().Close()
	defer os.RemoveAll(s.LdbDir)

	l := ds.Head().Get(datas.ValueField).(types.List)
	s.Equal(uint64(100), l.Len())

	i := uint64(0)
	l.IterAll(func(v types.Value, j uint64) {
		s.Equal(i, j)
		st := v.(types.Struct)
		s.Equal(types.NewString(fmt.Sprintf("a%d", i)), st.Get("a"))
		s.Equal(types.Number(i), st.Get("b"))
		i++
	})
}

func (s *testSuite) TestCSVImporterWithPipe() {
	oldDelimiter := delimiter
	newDelimiter := "|"
	delimiter = &newDelimiter
	defer func() { delimiter = oldDelimiter }()

	input, err := ioutil.TempFile(s.TempDir, "")
	d.Chk.NoError(err)
	defer input.Close()
	defer os.Remove(input.Name())

	_, err = input.WriteString("a|b\n1|2\n")
	d.Chk.NoError(err)

	setName := "csv"
	dataspec := test_util.CreateValueSpecString("ldb", s.LdbDir, setName)
	out := s.Run(main, []string{"-no-progress", "-column-types", "String,Number", dataspec, input.Name()})
	s.Equal("", out)

	cs := chunks.NewLevelDBStore(s.LdbDir, "", 1, false)
	ds := dataset.NewDataset(datas.NewDatabase(cs), setName)
	defer ds.Database().Close()
	defer os.RemoveAll(s.LdbDir)

	l := ds.Head().Get(datas.ValueField).(types.List)
	s.Equal(uint64(1), l.Len())
	v := l.Get(0)
	st := v.(types.Struct)
	s.Equal(types.NewString("1"), st.Get("a"))
	s.Equal(types.Number(2), st.Get("b"))
}

func (s *testSuite) TestCSVImporterWithExternalHeader() {
	oldHeader := header
	newHeader := "x,y"
	header = &newHeader
	defer func() { header = oldHeader }()

	input, err := ioutil.TempFile(s.TempDir, "")
	d.Chk.NoError(err)
	defer input.Close()
	defer os.Remove(input.Name())

	_, err = input.WriteString("7,8\n")
	d.Chk.NoError(err)

	setName := "csv"
	dataspec := test_util.CreateValueSpecString("ldb", s.LdbDir, setName)
	out := s.Run(main, []string{"-no-progress", "-column-types", "String,Number", dataspec, input.Name()})
	s.Equal("", out)

	cs := chunks.NewLevelDBStore(s.LdbDir, "", 1, false)
	ds := dataset.NewDataset(datas.NewDatabase(cs), setName)
	defer ds.Database().Close()
	defer os.RemoveAll(s.LdbDir)

	l := ds.Head().Get(datas.ValueField).(types.List)
	s.Equal(uint64(1), l.Len())
	v := l.Get(0)
	st := v.(types.Struct)
	s.Equal(types.NewString("7"), st.Get("x"))
	s.Equal(types.Number(8), st.Get("y"))
}
