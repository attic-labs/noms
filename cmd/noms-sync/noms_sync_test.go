// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"path"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/test_util"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

func TestSync(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	test_util.ClientTestSuite
}

func (s *testSuite) TestSync() {
	source1 := dataset.NewDataset(datas.NewDatabase(chunks.NewLevelDBStore(s.LdbDir, "", 1, false)), "foo")
	source1, err := source1.Commit(types.Number(42))
	s.NoError(err)
	source, err := source1.Commit(types.Number(45))
	s.NoError(err)
	for i := 0; i < 4; i++ {
		source, err = source.Commit(types.Number(43))
		s.NoError(err)
	}
	source1HeadRef := source1.Head().Hash()
	source.Database().Close() // Close Database backing both Datasets

	sourceSpec := test_util.CreateValueSpecString("ldb", s.LdbDir, source1HeadRef.String())
	ldb2dir := path.Join(s.TempDir, "ldb2")
	sinkDatasetSpec := test_util.CreateValueSpecString("ldb", ldb2dir, "bar")
	out := s.Run(main, []string{sourceSpec, sinkDatasetSpec})
	s.Equal("\x1b[2K\r0/0\n", out)

	dest := dataset.NewDataset(datas.NewDatabase(chunks.NewLevelDBStore(ldb2dir, "", 1, false)), "bar")
	s.True(types.Number(42).Equals(dest.Head().Get(datas.ValueField)))
	dest.Database().Close()

	sourceDataset := test_util.CreateValueSpecString("ldb", s.LdbDir, "foo")
	out = s.Run(main, []string{sourceDataset, sinkDatasetSpec})
	s.Equal("\x1b[2K\r0/0\x1b[2K\r0/1\x1b[2K\r1/1\x1b[2K\r1/2\x1b[2K\r2/2\x1b[2K\r2/3\x1b[2K\r3/3\x1b[2K\r3/4\x1b[2K\r4/4\x1b[2K\r4/5\x1b[2K\r5/5\n", out)

	dest = dataset.NewDataset(datas.NewDatabase(chunks.NewLevelDBStore(ldb2dir, "", 1, false)), "bar")
	s.True(types.Number(43).Equals(dest.Head().Get(datas.ValueField)))
	dest.Database().Close()
}

func createTestDataset(name string) dataset.Dataset {
	return dataset.NewDataset(datas.NewDatabase(chunks.NewTestStore()), name)
}

func pullDeepRef(t *testing.T) {
	assert := assert.New(t)

	sink := createTestDataset("sink")
	source := createTestDataset("source")

	sourceInitialValue := types.NewStruct("", map[string]types.Value{
		"Foo":  types.NewString("Foo"),
		"Bar":  source.Database().WriteValue(types.NewString("Bar")),
		"Baz":  source.Database().WriteValue(types.NewString("Baz")),
		"Barz": source.Database().WriteValue(types.NewString("Barz"))})

	source, err := source.Commit(sourceInitialValue)
	assert.NoError(err)

	var out string
	var count, total uint64

	progressCallback := func(sofar, expect uint64) {
		if total > 0 {
			count += sofar
		}
		total += expect
		out += fmt.Sprintf("%s%d/%d", clearLine, count, total)
	}

	sink, err = sink.Pull(source.Database(), types.NewRef(source.Head()), 1, progressCallback)
	assert.NoError(err)
	assert.True(source.Head().Equals(sink.Head()))
	assert.Equal("\x1b[2K\r0/0\x1b[2K\r0/1\x1b[2K\r0/2\x1b[2K\r0/3\x1b[2K\r1/3\x1b[2K\r2/3\x1b[2K\r3/3", out)
}

func TestPullDeepRefTopDown(t *testing.T) {
	pullDeepRef(t)
}
