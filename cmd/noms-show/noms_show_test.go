// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/test_util"
	"github.com/attic-labs/testify/suite"
)

func TestNomsShow(t *testing.T) {
	suite.Run(t, &nomsShowTestSuite{})
}

type nomsShowTestSuite struct {
	test_util.ClientTestSuite
}

const (
	res1 = "struct Commit {\n  parents: Set<Ref<Cycle<0>>>,\n  value: Value,\n}({\n  parents: {},\n  value: 12p4gc9k31cj9bwu8g8nrh03dadtg87pjqhd9zyi6hx80e25uf,\n})\n"
	res2 = "\"test string\"\n"
	res3 = "struct Commit {\n  parents: Set<Ref<Cycle<0>>>,\n  value: Value,\n}({\n  parents: {\n    65peo6a27ctstqj2du8vyfep2zffo9pdvpb2blrs7hpfurvqdk,\n  },\n  value: 22sx5udbba8xjlwipwrqif9su0nc7kokqo3xuffpzt5kuc0w4g,\n})\n"
	res4 = "List<Number | String>([\n  \"elem1\",\n  2,\n  \"elem3\",\n])\n"
	res5 = "struct Commit {\n  parents: Set<Ref<Cycle<0>>>,\n  value: Value,\n}({\n  parents: {\n    2f4gg3u1euh0z3joocekcanah6bgfm19n9n5bco6p8ijt4tcpg,\n  },\n  value: 12p4gc9k31cj9bwu8g8nrh03dadtg87pjqhd9zyi6hx80e25uf,\n})\n"
)

func writeTestData(ds dataset.Dataset, value types.Value) types.Ref {
	r1 := ds.Database().WriteValue(value)
	ds, err := ds.Commit(r1)
	d.Chk.NoError(err)
	err = ds.Database().Close()
	d.Chk.NoError(err)

	return r1
}

func (s *nomsShowTestSuite) TestNomsShow() {
	datasetName := "dsTest"
	str := test_util.CreateValueSpecString("ldb", s.LdbDir, datasetName)
	sp, err := spec.ParseDatasetSpec(str)
	d.Chk.NoError(err)
	ds, err := sp.Dataset()
	d.Chk.NoError(err)

	s1 := types.String("test string")
	r := writeTestData(ds, s1)
	s.Equal(res1, s.Run(main, []string{str}))

	spec1 := test_util.CreateValueSpecString("ldb", s.LdbDir, r.TargetHash().String())
	s.Equal(res2, s.Run(main, []string{spec1}))

	ds, err = sp.Dataset()
	list := types.NewList(types.String("elem1"), types.Number(2), types.String("elem3"))
	r = writeTestData(ds, list)
	s.Equal(res3, s.Run(main, []string{str}))

	spec1 = test_util.CreateValueSpecString("ldb", s.LdbDir, r.TargetHash().String())
	s.Equal(res4, s.Run(main, []string{spec1}))

	ds, err = sp.Dataset()
	_ = writeTestData(ds, s1)
	s.Equal(res5, s.Run(main, []string{str}))
}
