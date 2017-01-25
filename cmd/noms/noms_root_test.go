// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/datas"
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

func (s *nomsRootTestSuite) TestBasic() {
	datasetName := "root-get"
	str := spec.CreateValueSpecString("ldb", s.LdbDir, datasetName)
	sp, err := spec.ForDataset(str)
	s.NoError(err)
	defer sp.Close()

	ds := sp.GetDataset()
	ds, _ = ds.Database().Commit(ds, types.String("hello!"), datas.CommitOptions{})
	c1, _ := s.MustRun(main, []string{"root", spec.CreateDatabaseSpecString("ldb", s.LdbDir)})
	s.Equal("gt8mq6r7hvccp98s2vpeu9v9ct4rhloc\n", c1)

	ds, _ = ds.Database().Commit(ds, types.String("goodbye"), datas.CommitOptions{})
	c2, _ := s.MustRun(main, []string{"root", spec.CreateDatabaseSpecString("ldb", s.LdbDir)})
	s.Equal("8tj5ctfhbka8fag417huneepg5ji283u\n", c2)

	// TODO: Would be good to test --update too, but requires changes to MustRun to allow input
	// because of prompt :(.
}
