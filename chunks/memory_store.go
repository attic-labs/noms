package chunks

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"

	"github.com/attic-labs/noms/ref"
)

// An in-memory implementation of store.ChunkStore. Useful mainly for tests.
type MemoryStore struct {
	data map[ref.Ref][]byte
	memoryRootTracker
}

func (ms *MemoryStore) Get(ref ref.Ref) io.ReadCloser {
	if b, ok := ms.data[ref]; ok {
		return ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

func (ms *MemoryStore) Has(r ref.Ref) bool {
	if ms.data == nil {
		return false
	}
	_, ok := ms.data[r]
	return ok
}

func (ms *MemoryStore) Put(r ref.Ref, data []byte) {
	if ms.data == nil {
		ms.data = map[ref.Ref][]byte{}
	}
	ms.data[r] = data
}

func (ms *MemoryStore) Len() int {
	return len(ms.data)
}

type memoryStoreFlags struct {
	use *bool
}

func memoryFlags(prefix string) memoryStoreFlags {
	return memoryStoreFlags{
		flag.Bool(prefix+"mem", false, "use a memory-based (ephemeral, and private to this application) chunkstore"),
	}
}

func (f memoryStoreFlags) createStore() ChunkStore {
	if *f.use {
		return &MemoryStore{}
	} else {
		return nil
	}
}
