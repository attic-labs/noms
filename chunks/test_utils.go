package chunks

import (
	"io"
	"io/ioutil"

	"github.com/attic-labs/noms/ref"
	"github.com/stretchr/testify/assert"
)

func assertInputInStore(input string, ref ref.Ref, s ChunkStore, assert *assert.Assertions) {
	reader := s.Get(ref)
	assert.NotNil(reader)
	data, err := ioutil.ReadAll(reader)
	assert.NoError(err)
	assert.Equal(input, string(data))
}

func assertInputNotInStore(input string, ref ref.Ref, s ChunkStore, assert *assert.Assertions) {
	reader := s.Get(ref)
	assert.Nil(reader)
}

type TestStore struct {
	MemoryStore
	Reads  int
	Writes int
}

func (s *TestStore) Get(ref ref.Ref) io.ReadCloser {
	s.Reads++
	return s.MemoryStore.Get(ref)
}

func (s *TestStore) Has(ref ref.Ref) bool {
	return s.MemoryStore.Has(ref)
}

func (s *TestStore) Put(ref ref.Ref, data []byte) {
	s.Writes++
	s.MemoryStore.Put(ref, data)
}
