package chunks

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestFileStoreTestSuite(t *testing.T) {
	suite.Run(t, &FileStoreTestSuite{})
}

type FileStoreTestSuite struct {
	suite.Suite
	dir      string
	store    FileStore
	putCount int
}

func (suite *FileStoreTestSuite) SetupTest() {
	var err error
	suite.dir, err = ioutil.TempDir(os.TempDir(), "")
	suite.NoError(err)
	suite.store = NewFileStore(suite.dir, "root")

	suite.putCount = 0
	suite.store.mkdirAll = func(path string, perm os.FileMode) error {
		suite.putCount++
		return os.MkdirAll(path, perm)
	}
}

func (suite *FileStoreTestSuite) TearDownTest() {
	os.Remove(suite.dir)
}

func (suite *FileStoreTestSuite) Store() ChunkStore {
	return suite.store
}

func (suite *FileStoreTestSuite) PutCountFn() func() int {
	return func() int {
		return suite.putCount
	}
}

func (suite *FileStoreTestSuite) TestFileStoreCommon() {
	ChunkStoreTestCommon(suite)
}
