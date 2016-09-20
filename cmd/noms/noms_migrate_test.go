// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/spec"
	v7spec "github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	v7types "github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestNomsMigrate(t *testing.T) {
	suite.Run(t, &nomsMigrateTestSuite{})
}

type nomsMigrateTestSuite struct {
	clienttest.ClientTestSuite
}

func (s *nomsMigrateTestSuite) writeTestData(str string, value v7types.Value) {
	ds, err := v7spec.GetDataset(str)
	s.NoError(err)

	ds, err = ds.CommitValue(value)
	s.NoError(err)

	err = ds.Database().Close()
	s.NoError(err)
}

func (s *nomsMigrateTestSuite) TestNomsMigrate() {
	sourceDsName := "migrateSourceTest"
	sourceStr := v7spec.CreateValueSpecString("ldb", s.LdbDir, sourceDsName)

	destDsName := "migrateDestTest"
	destStr := spec.CreateValueSpecString("ldb", s.LdbDir, destDsName)

	str := "Hello world"
	v7val := v7types.String(str)

	s.writeTestData(sourceStr, v7val)

	outStr, errStr := s.MustRun(main, []string{"migrate", sourceStr, destStr})
	s.Equal("", outStr)
	s.Equal("", errStr)

	destDs, err := spec.GetDataset(destStr)
	s.NoError(err)

	s.True(destDs.HeadValue().Equals(types.String(str)))
}

func (s *nomsMigrateTestSuite) TestNomsMigrateNonCommit() {
	sourceDsName := "migrateSourceTest2"
	sourceStr := v7spec.CreateValueSpecString("ldb", s.LdbDir, sourceDsName)

	destDsName := "migrateDestTest2"
	destStr := spec.CreateValueSpecString("ldb", s.LdbDir, destDsName)

	str := "Hello world"
	v7val := v7types.NewStruct("", v7types.StructData{
		"str": v7types.String(str),
	})

	s.writeTestData(sourceStr, v7val)

	outStr, errStr := s.MustRun(main, []string{"migrate", sourceStr + ".value.str", destStr})
	s.Equal("", outStr)
	s.Equal("", errStr)

	destDs, err := spec.GetDataset(destStr)
	s.NoError(err)

	s.True(destDs.HeadValue().Equals(types.String(str)))
}

func (s *nomsMigrateTestSuite) TestNomsMigrateNil() {
	sourceDsName := "migrateSourceTest3"
	sourceStr := v7spec.CreateValueSpecString("ldb", s.LdbDir, sourceDsName)

	destDsName := "migrateDestTest3"
	destStr := spec.CreateValueSpecString("ldb", s.LdbDir, destDsName)

	defer func() {
		err := recover()
		s.Equal(clienttest.ExitError{Code: -1}, err)
	}()

	s.MustRun(main, []string{"migrate", sourceStr, destStr})
}
