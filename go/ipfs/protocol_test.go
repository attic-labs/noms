package ipfs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const repoTempPrefix = "noms-ipfs-test-"

func mustRegister() func() {
	if err := RegisterProtocols(); err != nil {
		panic(err)
	}

	return func() {
		UnregisterProtocols()
	}
}

func mustTempDir(prefix string) (name string) {
	name, err := ioutil.TempDir("", prefix)
	if err != nil {
		panic("couldn't create temporary directory: " + err.Error())
	}
	return name
}

func TestProtocol(t *testing.T) {
	defer mustRegister()()
	for _, proto := range []string{"ipfs", "ipfs-local"} {
		t.Run(proto, func(t *testing.T) {
			tempDir := mustTempDir(repoTempPrefix)
			defer os.RemoveAll(tempDir)
			specStr := spec.CreateDatabaseSpecString(proto, tempDir)
			_, err := spec.ForDatabase(specStr)
			require.NoError(t, err, "creating spec from database string", specStr, "should not fail")
		})
	}
}

func TestDbFromProtocol(t *testing.T) {
	defer mustRegister()()
	for _, proto := range []string{"ipfs", "ipfs-local"} {
		t.Run(proto, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			tempDir := mustTempDir(repoTempPrefix)

			defer os.RemoveAll(tempDir)
			specStr := spec.CreateDatabaseSpecString(proto, tempDir)
			sp, err := spec.ForDatabase(specStr)
			require.NoError(err, "creating spec from database string", specStr, "should not fail")
			var db datas.Database
			require.NotPanics(func() {
				db = sp.GetDatabase()
			}, "GetDatabase should not panic")

			assert.Implements((*HasIPFSNode)(nil), db, "the returned database should implement HasIPFSNode")
			var stats IPFSStats
			assert.NotPanics(func() {
				stats = db.Stats().(IPFSStats)
			}, "Stats should return an IPFSStats")

			local := proto == LocalProtoName

			assert.Equal(local, stats.Local, "stats.Local should match proto %s", proto)
			assert.NoError(db.Close(), "db.Close() should not fail")
		})
	}
}

func TestChunkStoreFromProtocol(t *testing.T) {
	defer mustRegister()()
	for _, proto := range []string{"ipfs", "ipfs-local"} {
		t.Run(proto, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			tempDir := mustTempDir(repoTempPrefix)

			defer os.RemoveAll(tempDir)
			specStr := spec.CreateDatabaseSpecString(proto, tempDir)
			sp, err := spec.ForDatabase(specStr)
			require.NoError(err, "creating spec from database string", specStr, "should not fail")
			var cs chunks.ChunkStore
			require.NotPanics(func() {
				cs = sp.NewChunkStore()
			}, "NewChunkStore should not panic")

			assert.Implements((*HasIPFSNode)(nil), cs, "the returned ChunkStore should implement HasIPFSNode")
			var stats IPFSStats
			assert.NotPanics(func() {
				stats = cs.Stats().(IPFSStats)
			}, "Stats should return an IPFSStats")

			local := proto == LocalProtoName

			assert.Equal(local, stats.Local, "stats.Local should match proto %s", proto)
			assert.NoError(cs.Close(), "cs.Close() should not fail")
		})
	}
}
