package chunks

import (
	"sync"

	"github.com/attic-labs/noms/ref"
)

type HasCacheStore struct {
	cs  ChunkStore
	has map[ref.Ref]bool
	mu  sync.Mutex
}

func NewHasCacheStore(cs ChunkStore) ChunkStore {
	return &HasCacheStore{cs, map[ref.Ref]bool{}, sync.Mutex{}}
}

func (self *HasCacheStore) checkHas(ref ref.Ref) (has bool, ok bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	has, ok = self.has[ref]
	return
}

func (self *HasCacheStore) setHas(ref ref.Ref, has bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.has[ref] = has
}

func (self *HasCacheStore) Get(ref ref.Ref) Chunk {
	c := self.cs.Get(ref)
	self.setHas(ref, !c.IsEmpty())
	return c
}

func (self *HasCacheStore) Has(ref ref.Ref) bool {
	has, ok := self.checkHas(ref)
	return ok && has
}

func (self *HasCacheStore) Put(c Chunk) {
	if self.Has(c.Ref()) {
		return
	}

	self.cs.Put(c)
	self.setHas(c.Ref(), true)
}

func (self *HasCacheStore) Close() error {
	return self.cs.Close()
}

func (self *HasCacheStore) Root() ref.Ref {
	return self.cs.Root()
}

func (self *HasCacheStore) UpdateRoot(current, last ref.Ref) bool {
	return self.cs.UpdateRoot(current, last)
}
