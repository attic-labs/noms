package chunks

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestLevelDBStoreTestSuite(t *testing.T) {
	suite.Run(t, &LevelDBStoreTestSuite{})
}

type LevelDBStoreTestSuite struct {
	suite.Suite
	dir   string
	store *LevelDBStore
}

func (suite *LevelDBStoreTestSuite) SetupTest() {
	var err error
	suite.dir, err = ioutil.TempDir(os.TempDir(), "")
	suite.NoError(err)
	suite.store = NewLevelDBStore(suite.dir)
}

func (suite *LevelDBStoreTestSuite) TearDownTest() {
	os.Remove(suite.dir)
}

func (suite *LevelDBStoreTestSuite) Store() ChunkStore {
	return suite.store
}

func (suite *LevelDBStoreTestSuite) PutCountFn() func() int {
	return func() int {
		return suite.store.putCount
	}
}

func (suite *LevelDBStoreTestSuite) TestLevelDBStoreCommon() {
	ChunkStoreTestCommon(suite)
}
