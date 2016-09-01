// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestBasics(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	clienttest.ClientTestSuite
}

func (s *testSuite) TestWin() {
	prep := func(name string, data types.StructData) {
		ds, _ := spec.GetDataset(spec.CreateValueSpecString("ldb", s.LdbDir, name))
		defer ds.Database().Close()
		ds.CommitValue(types.NewStruct("", data))
	}

	p := "parent"
	prep(p, types.StructData{
		"num": types.Number(42),
		"str": types.String("foobar"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1)),
	})

	l := "left"
	prep(l, types.StructData{
		"num": types.Number(42),
		"str": types.String("foobaz"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1)),
	})

	r := "right"
	prep(r, types.StructData{
		"num": types.Number(42),
		"str": types.String("foobar"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
	})

	expected := types.NewStruct("", types.StructData{
		"num": types.Number(42),
		"str": types.String("foobaz"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
	})

	var mainErr error
	stdout, stderr := s.Run(func() { mainErr = nomsMerge() }, []string{"--quiet=true", "--parent=" + p, s.LdbDir, l, r})
	if s.NoError(mainErr, "%s", mainErr) {
		s.Equal("", stdout)
		s.Equal("", stderr)

		ds, err := spec.GetDataset(spec.CreateValueSpecString("ldb", s.LdbDir, r))
		if s.NoError(err) {
			merged := ds.HeadValue()
			s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
		}
		defer ds.Database().Close()
	}
}

func (s *testSuite) TestLose() {
	sp := spec.CreateDatabaseSpecString("ldb", s.LdbDir)
	p, l, r := "parent", "left", "right"
	type c struct {
		args []string
		err  string
	}
	cases := []c{
		{[]string{"foo"}, "Incorrect number of arguments\n"},
		{[]string{"foo", "bar"}, "Incorrect number of arguments\n"},
		{[]string{"foo", "bar", "baz", "quux"}, "Incorrect number of arguments\n"},
		{[]string{"foo", "bar", "baz"}, "--parent is required\n"},
		{[]string{"--parent=" + p, sp, l + "!!", r}, "Invalid dataset " + l + "!!, must match [a-zA-Z0-9\\-_/]+\n"},
		{[]string{"--parent=" + p, sp, l + "2", r}, "Dataset " + l + "2 has no data\n"},
		{[]string{"--parent=" + p + "2", sp, l, r}, "Dataset " + p + "2 has no data\n"},
		{[]string{"--parent=" + p, sp, l, r + "2"}, "Dataset " + r + "2 has no data\n"},
		{[]string{"--parent=" + p, "--out-ds-name", "!invalid", sp, l, r}, "Invalid dataset !invalid, must match [a-zA-Z0-9\\-_/]+\n"},
	}

	db, _ := spec.GetDatabase(sp)
	prep := func(dsName string) {
		ds := dataset.NewDataset(db, dsName)
		ds.CommitValue(types.NewMap(types.String("foo"), types.String("bar")))
	}
	prep(p)
	prep(l)
	prep(r)
	db.Close()

	for _, c := range cases {
		var mainErr error
		stdout, _ := s.Run(func() { mainErr = nomsMerge() }, c.args)
		s.Empty(stdout, "Expected empty stdout for case: %#v", c.args)
		if s.Error(mainErr) {
			s.Equal(c.err, mainErr.Error(), "Unexpected output for case: %#v\n", c.args)
		}
	}
}
