// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package chunks_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/chunks/chunkstest"
)

func TestMemoryStoreTestSuite(t *testing.T) {
	suite.Run(t, &MemoryStoreTestSuite{})
}

type MemoryStoreTestSuite struct {
	chunkstest.ChunkStoreTestSuite
}

func (suite *MemoryStoreTestSuite) SetupTest() {
	suite.Factory = chunks.NewMemoryStoreFactory()
}

func (suite *MemoryStoreTestSuite) TearDownTest() {
	suite.Factory.Shutter()
}
