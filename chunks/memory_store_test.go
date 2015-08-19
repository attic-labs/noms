package chunks

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMemoryStoreTestSuite(t *testing.T) {
	suite.Run(t, &MemoryStoreTestSuite{})
}

type MemoryStoreTestSuite struct {
	suite.Suite
	store *MemoryStore
}

func (suite *MemoryStoreTestSuite) SetupTest() {
	suite.store = &MemoryStore{}
}

func (suite *MemoryStoreTestSuite) TearDownTest() {
}

func (suite *MemoryStoreTestSuite) Store() ChunkStore {
	return suite.store
}

func (suite *MemoryStoreTestSuite) PutCountFn() func() int {
	return nil
}

func (suite *MemoryStoreTestSuite) TestMemoryStoreCommon() {
	ChunkStoreTestCommon(suite)
}
