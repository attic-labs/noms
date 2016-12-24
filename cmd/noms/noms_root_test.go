// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestNomsRoot(t *testing.T) {
	suite.Run(t, &nomsRootTestSuite{})
}

type nomsRootTestSuite struct {
	clienttest.ClientTestSuite
}

func (s *nomsShowTestSuite) TestNomsRoot() {
	datasetName := "root-get"
	str := spec.CreateValueSpecString("ldb", s.LdbDir, datasetName)
	sp, err := spec.ForDataset(str)
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()
	r1 := db.WriteValue(types.String("test"))
	res, _ := s.MustRun(main, []string{"show", "--raw", spec.CreateValueSpecString("ldb", s.LdbDir, "#"+r1.TargetHash().String())})

	ch := chunks.NewChunk([]byte(res))
	v := types.DecodeValue(ch, db)
	s.True(v.Equals(types.String("test")))
}
