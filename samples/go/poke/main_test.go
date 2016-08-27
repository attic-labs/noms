// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"testing"

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

func (s *testSuite) TestRoundTrip() {
	sp := fmt.Sprintf("ldb:%s::test", s.LdbDir)
	ds, _ := spec.GetDataset(sp)
	ds.CommitValue(types.NewStruct("", map[string]types.Value{
		"num": types.Number(42),
		"str": types.String("foobar"),
		"lst": types.NewList(types.Number(1), types.String("foo")),
		"map": types.NewMap(types.Number(1), types.String("foo"),
			types.String("foo"), types.Number(1)),
	}))
	ds.Database().Close()

	changes := map[string]string{
		".num":          "43",
		".str":          "\"foobaz\"",
		".lst[0]":       "2",
		".map[1]":       "\"bar\"",
		".map[\"foo\"]": "2",
	}

	for k, v := range changes {
		stdout, stderr := s.Run(main, []string{sp, k, v})
		s.Equal("", stdout)
		s.Equal("", stderr)
	}

	ds, _ = spec.GetDataset(sp)
	r := ds.HeadValue()
	for k, vs := range changes {
		v, _, _, _ := types.ParsePathIndex(vs)
		p, err := types.ParsePath(k)
		s.NoError(err)
		actual := p.Resolve(r)
		s.True(actual.Equals(v), "value at path %s incorrect (expected: %#v, got: %#v)", p.String(), v, actual)
	}
}
