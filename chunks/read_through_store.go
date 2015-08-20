package chunks

import (
	"bytes"
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
	cachingStore ChunkStore
	backingStore ChunkStore
	putCount     int
}

func NewReadThroughStore(cachingStore ChunkStore, backingStore ChunkStore) ReadThroughStore {
	return ReadThroughStore{cachingStore, backingStore, 0}
}

// forwardCloser closes multiple io.Closer objects.
type forwardCloser struct {
	io.Reader
	cs []io.Closer
}

func (fc forwardCloser) Close() error {
	for _, c := range fc.cs {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (rts ReadThroughStore) Get(ref ref.Ref) io.ReadCloser {
	r := rts.cachingStore.Get(ref)
	if r != nil {
		return r
	}
	r = rts.backingStore.Get(ref)
	if r == nil {
		return r
	}

	buff := &bytes.Buffer{}
	io.Copy(buff, r)
	rts.cachingStore.Put(ref, buff.Bytes())
	return ioutil.NopCloser(buff)
}

func (rts ReadThroughStore) Has(ref ref.Ref) bool {
	if rts.cachingStore.Has(ref) {
		return true
	}

	return rts.backingStore.Has(ref)
}

func (rts ReadThroughStore) Put(ref ref.Ref, data []byte) {
	rts.backingStore.Put(ref, data)
	rts.cachingStore.Put(ref, data)
}

func (rts ReadThroughStore) Root() ref.Ref {
	return rts.backingStore.Root()
}

func (rts ReadThroughStore) UpdateRoot(current, last ref.Ref) bool {
	return rts.backingStore.UpdateRoot(current, last)
}
