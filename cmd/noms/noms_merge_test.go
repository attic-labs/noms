// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

type nomsMergeTestSuite struct {
	clienttest.ClientTestSuite
}

func TestNomsMerge(t *testing.T) {
	suite.Run(t, &nomsMergeTestSuite{})
}

func (s *nomsMergeTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll(s.LdbDir))
}

func (s *nomsMergeTestSuite) setupMergeDatasets(left, right string) (l, r types.Ref) {
	p := s.setupMergeDataset(
		"parent",
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobar"),
			"lst": types.NewList(types.Number(1), types.String("foo")),
			"map": types.NewMap(types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1)),
		},
		types.NewSet())

	l = s.setupMergeDataset(
		left,
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobaz"),
			"lst": types.NewList(types.Number(1), types.String("foo")),
			"map": types.NewMap(types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1)),
		},
		types.NewSet(p))

	r = s.setupMergeDataset(
		right,
		types.StructData{
			"num": types.Number(42),
			"str": types.String("foobar"),
			"lst": types.NewList(types.Number(1), types.String("foo")),
			"map": types.NewMap(types.Number(1), types.String("foo"),
				types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
		},
		types.NewSet(p))
	return
}

func (s *nomsMergeTestSuite) setupMergeDataset(name string, data types.StructData, p types.Set) types.Ref {
	db, ds, _ := spec.GetDataset(spec.CreateValueSpecString("ldb", s.LdbDir, name))
	defer db.Close()
	ds, err := db.Commit(ds, types.NewStruct("", data), datas.CommitOptions{Parents: p})
	s.NoError(err)
	return ds.HeadRef()
}

func (s *nomsMergeTestSuite) validateDataset(name string, expected types.Struct, parents ...types.Value) {

	db, ds, err := spec.GetDataset(spec.CreateValueSpecString("ldb", s.LdbDir, name))
	if s.NoError(err) {
		commit := ds.Head()
		s.True(commit.Get(datas.ParentsField).Equals(types.NewSet(parents...)))
		merged := ds.HeadValue()
		s.True(expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
	defer db.Close()
}

func (s *nomsMergeTestSuite) TestNomsMerge_Success() {
	left, right := "left", "right"
	l, r := s.setupMergeDatasets(left, right)
	expected := types.NewStruct("", types.StructData{
		"num": types.Number(42),
		"str": types.String("foobaz"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
	})

	stdout, stderr, err := s.Run(main, []string{"merge", s.LdbDir, left, right})
	if err == nil {
		s.Equal("", stderr)
		s.validateDataset(right, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_SuccessNesDataset() {
	left, right := "left", "right"
	l, r := s.setupMergeDatasets(left, right)
	expected := types.NewStruct("", types.StructData{
		"num": types.Number(42),
		"str": types.String("foobaz"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1), types.Number(2), types.String("bar")),
	})

	output := "output"
	stdout, stderr, err := s.Run(main, []string{"merge", s.LdbDir, left, right, output})
	if err == nil {
		s.Equal("", stderr)
		s.validateDataset(output, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Left() {
	left, right := "left", "right"
	p := s.setupMergeDataset("parent", types.StructData{"num": types.Number(42)}, types.NewSet())
	l := s.setupMergeDataset(left, types.StructData{"num": types.Number(43)}, types.NewSet(p))
	r := s.setupMergeDataset(right, types.StructData{"num": types.Number(44)}, types.NewSet(p))

	expected := types.NewStruct("", types.StructData{"num": types.Number(43)})

	stdout, stderr, err := s.Run(main, []string{"merge", "--policy=l", s.LdbDir, left, right})
	if err == nil {
		s.Equal("", stderr)
		s.validateDataset(right, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Right() {
	left, right := "left", "right"
	p := s.setupMergeDataset("parent", types.StructData{"num": types.Number(42)}, types.NewSet())
	l := s.setupMergeDataset(left, types.StructData{"num": types.Number(43)}, types.NewSet(p))
	r := s.setupMergeDataset(right, types.StructData{"num": types.Number(44)}, types.NewSet(p))

	expected := types.NewStruct("", types.StructData{"num": types.Number(44)})

	output := "output"
	stdout, stderr, err := s.Run(main, []string{"merge", "--policy=r", s.LdbDir, left, right, output})
	if err == nil {
		s.Equal("", stderr)
		s.validateDataset(output, expected, l, r)
	} else {
		s.Fail("Run failed", "err: %v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
	}
}

func (s *nomsMergeTestSuite) TestNomsMerge_Conflict() {
	left, right := "left", "right"
	p := s.setupMergeDataset("parent", types.StructData{"num": types.Number(42)}, types.NewSet())
	s.setupMergeDataset(left, types.StructData{"num": types.Number(43)}, types.NewSet(p))
	s.setupMergeDataset(right, types.StructData{"num": types.Number(44)}, types.NewSet(p))

	s.Panics(func() { s.MustRun(main, []string{"merge", s.LdbDir, left, right}) })
}

func (s *nomsMergeTestSuite) TestBadInput() {
	sp := spec.CreateDatabaseSpecString("ldb", s.LdbDir)
	p, l, r := "parent", "left", "right"
	type c struct {
		args []string
		err  string
	}
	cases := []c{
		{[]string{"foo"}, "error: Incorrect number of arguments\n"},
		{[]string{"foo", "bar"}, "error: Incorrect number of arguments\n"},
		{[]string{"foo", "bar", "baz", "quux", "five"}, "error: Incorrect number of arguments\n"},
		{[]string{sp, l + "!!", r}, "error: Invalid dataset " + l + "!!, must match [a-zA-Z0-9\\-_/]+\n"},
		{[]string{sp, l + "2", r}, "error: Dataset " + l + "2 has no data\n"},
		{[]string{sp, l, r + "2"}, "error: Dataset " + r + "2 has no data\n"},
		{[]string{sp, l, r, "!invalid"}, "error: Invalid dataset !invalid, must match [a-zA-Z0-9\\-_/]+\n"},
	}

	db, _ := spec.GetDatabase(sp)
	prep := func(dsName string) {
		ds := db.GetDataset(dsName)
		db.CommitValue(ds, types.NewMap(types.String("foo"), types.String("bar")))
	}
	prep(p)
	prep(l)
	prep(r)
	db.Close()

	for _, c := range cases {
		stdout, stderr, err := s.Run(main, append([]string{"merge"}, c.args...))
		s.Empty(stdout, "Expected empty stdout for case: %#v", c.args)
		if !s.NotNil(err, "Unexpected success for case: %#v\n", c.args) {
			continue
		}
		if mainErr, ok := err.(clienttest.ExitError); ok {
			s.Equal(1, mainErr.Code)
			s.Equal(c.err, stderr, "Unexpected output for case: %#v\n", c.args)
		} else {
			s.Fail("Run() recovered non-error panic", "err: %#v\nstdout: %s\nstderr: %s\n", err, stdout, stderr)
		}
	}
}

func TestNomsMergeCliResolve(t *testing.T) {
	type c struct {
		input            string
		aChange, bChange types.DiffChangeType
		aVal, bVal       types.Value
		expectedChange   types.DiffChangeType
		expected         types.Value
		success          bool
	}

	cases := []c{
		{"l\n", types.DiffChangeAdded, types.DiffChangeAdded, types.String("foo"), types.String("bar"), types.DiffChangeAdded, types.String("foo"), true},
		{"r\n", types.DiffChangeAdded, types.DiffChangeAdded, types.String("foo"), types.String("bar"), types.DiffChangeAdded, types.String("bar"), true},
		{"l\n", types.DiffChangeAdded, types.DiffChangeAdded, types.Number(7), types.String("bar"), types.DiffChangeAdded, types.Number(7), true},
		{"r\n", types.DiffChangeModified, types.DiffChangeModified, types.Number(7), types.String("bar"), types.DiffChangeModified, types.String("bar"), true},
	}

	for _, c := range cases {
		input := bytes.NewBufferString(c.input)

		changeType, newVal, ok := cliResolve(input, ioutil.Discard, c.aChange, c.bChange, c.aVal, c.bVal, types.Path{})
		if !c.success {
			assert.False(t, ok)
		} else if assert.True(t, ok) {
			assert.Equal(t, c.expectedChange, changeType)
			assert.True(t, c.expected.Equals(newVal))
		}
	}
}
