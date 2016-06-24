package sizecache

import (
	"container/list"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type lruList struct {
	list.List
}

type SizeCache interface {
	Get(h hash.Hash) (interface{}, bool)
	Add(h hash.Hash, size uint64, value interface{})
}

type sizeCacheEntry struct {
	size     uint64
	lruEntry *list.Element
	value    interface{}
}

type noopSizeCache struct{}

type realSizeCache struct {
	totalSize uint64
	maxSize   uint64
	mu        sync.Mutex
	lru       lruList
	cache     map[string]sizeCacheEntry
}

func New(maxSize uint64) SizeCache {
	if maxSize == 0 {
		return &noopSizeCache{}
	}
	return &realSizeCache{maxSize: maxSize, cache: map[string]sizeCacheEntry{}}
}

func (c *realSizeCache) entry(h hash.Hash) (sizeCacheEntry, bool) {
	key := h.String()
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.cache[key]
	if !ok {
		return sizeCacheEntry{}, false
	}
	c.lru.MoveToBack(entry.lruEntry)
	return entry, true
}

func (c *realSizeCache) Get(h hash.Hash) (interface{}, bool) {
	if entry, ok := c.entry(h); ok {
		return entry.value, true
	}
	return nil, false
}

func (c *realSizeCache) Add(h hash.Hash, size uint64, value interface{}) {
	if _, ok := c.entry(h); ok {
		return
	}

	key := h.String()
	if size < c.maxSize {
		c.mu.Lock()
		defer c.mu.Unlock()
		newEl := c.lru.PushBack(key)
		ce := sizeCacheEntry{size: size, lruEntry: newEl, value: value}
		c.cache[key] = ce
		c.totalSize += ce.size
		for el := c.lru.Front(); el != nil && c.totalSize > c.maxSize; {
			key := el.Value.(string)
			ce, ok := c.cache[key]
			d.Chk.True(ok, "SizeCache is missing expected value")
			next := el.Next()
			delete(c.cache, key)
			c.totalSize -= ce.size
			c.lru.Remove(el)
			el = next
		}
	}
}

func (c *noopSizeCache) Get(h hash.Hash) (interface{}, bool) {
	return nil, false
}

func (c *noopSizeCache) Add(h hash.Hash, size uint64, value interface{}) {}
