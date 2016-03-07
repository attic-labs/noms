package chunks

import (
	"io"
	"io/ioutil"

	"github.com/attic-labs/noms/ref"
)

// ReadThroughStore is a store that consists of two other stores. A caching and
// a backing store. All reads check the caching store first and if the ref is
// present there the caching store is used. If not present the backing store is
// used and the value gets cached in the caching store. All writes go directly
// to the backing store.
type ReadThroughStore struct {
	io.Closer
	cachingStore ChunkStore
	backingStore ChunkStore
	putCount     int
}

func NewReadThroughStore(cachingStore ChunkStore, backingStore ChunkStore) ReadThroughStore {
	return ReadThroughStore{ioutil.NopCloser(nil), cachingStore, backingStore, 0}
}

func (rts ReadThroughStore) Get(ref ref.Ref) Chunk {
	c := rts.cachingStore.Get(ref)
	if !c.IsEmpty() {
		return c
	}
	c = rts.backingStore.Get(ref)
	if c.IsEmpty() {
		return c
	}

	rts.cachingStore.Put(c)
	return c
}

func (rts ReadThroughStore) Has(ref ref.Ref) bool {
	return rts.cachingStore.Has(ref) || rts.backingStore.Has(ref)
}

func (rts ReadThroughStore) Put(c Chunk) {
	rts.backingStore.Put(c)
	rts.cachingStore.Put(c)
}

func (rts ReadThroughStore) PutMany(chunks ...Chunk) BackpressureError {
	bpe1 := rts.backingStore.PutMany(chunks...)
	bpe2 := rts.cachingStore.PutMany(chunks...)
	if bpe1 == nil {
		return bpe2
	} else if bpe2 == nil {
		return bpe1
	}
	// Neither is nil
	lookup := make(map[ref.Ref]bool, len(bpe1))
	for _, c := range bpe1 {
		lookup[c.Ref()] = true
	}
	for _, c := range bpe2 {
		if !lookup[c.Ref()] {
			bpe1 = append(bpe1, c)
		}
	}
	return bpe1
}

func (rts ReadThroughStore) Root() ref.Ref {
	return rts.backingStore.Root()
}

func (rts ReadThroughStore) UpdateRoot(current, last ref.Ref) bool {
	return rts.backingStore.UpdateRoot(current, last)
}
