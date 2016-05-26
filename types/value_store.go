package types

import (
	"sync"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/hash"
)

// ValueStore provides methods to read and write Noms Values to a BatchStore. It validates Values as they are written, but does not guarantee that these Values are persisted to the BatchStore until a subsequent Flush. or Close.
// Currently, WriteValue validates the following properties of a Value v:
// - v can be correctly serialized and its Ref taken
// - all Refs in v point to a Value that can be read from this ValueStore
// - all Refs in v point to a Value of the correct Type
type ValueStore struct {
	bs    BatchStore
	cache map[hash.Hash]chunkCacheEntry
	mu    *sync.Mutex
}

type chunkCacheEntry interface {
	Present() bool
	Hint() hash.Hash
	Type() *Type
}

// NewTestValueStore creates a simple struct that satisfies ValueReadWriter and is backed by a chunks.TestStore.
func NewTestValueStore() *ValueStore {
	return newLocalValueStore(chunks.NewTestStore())
}

func newLocalValueStore(cs chunks.ChunkStore) *ValueStore {
	return NewValueStore(NewBatchStoreAdaptor(cs))
}

// NewValueStore returns a ValueStore instance that owns the provided BatchStore and manages its lifetime. Calling Close on the returned ValueStore will Close bs.
func NewValueStore(bs BatchStore) *ValueStore {
	return &ValueStore{bs, map[hash.Hash]chunkCacheEntry{}, &sync.Mutex{}}
}

func (lvs *ValueStore) BatchStore() BatchStore {
	return lvs.bs
}

// ReadValue reads and decodes a value from lvs. It is not considered an error for the requested chunk to be empty; in this case, the function simply returns nil.
func (lvs *ValueStore) ReadValue(r hash.Hash) Value {
	v := DecodeChunk(lvs.bs.Get(r), lvs)

	var entry chunkCacheEntry = absentChunk{}
	if v != nil {
		lvs.cacheChunks(v, r)
		// r is trivially a hint for v, so consider putting that in the cache. If we got to v by reading some higher-level chunk, this entry gets dropped on the floor because r already has a hint in the cache. If we later read some other chunk that references v, cacheChunks will overwrite this with a hint pointing to that chunk.
		// If we don't do this, top-level Values that get read but not written -- such as the existing Head of a Database upon a Commit -- can be erroneously left out during a pull.
		entry = hintedChunk{v.Type(), r}
	}
	if cur := lvs.check(r); cur == nil || cur.Hint().IsEmpty() {
		lvs.set(r, entry)
	}
	return v
}

// WriteValue takes a Value, schedules it to be written it to lvs, and returns an appropriately-typed types.Ref. v is not guaranteed to be actually written until after Flush().
func (lvs *ValueStore) WriteValue(v Value) Ref {
	d.Chk.NotNil(v)
	// Encoding v causes any child chunks, e.g. internal nodes if v is a meta sequence, to get written. That needs to happen before we try to validate v.
	c := EncodeValue(v, lvs)
	d.Chk.False(c.IsEmpty())
	hash := c.Hash()
	height := maxChunkHeight(v) + 1
	r := constructRef(MakeRefType(v.Type()), hash, height)
	if lvs.isPresent(hash) {
		return r
	}
	hints := lvs.chunkHintsFromCache(v)
	lvs.bs.SchedulePut(c, height, hints)
	lvs.set(hash, hintedChunk{v.Type(), hash})
	return r
}

func (lvs *ValueStore) Flush() {
	lvs.bs.Flush()
}

// Close closes the underlying BatchStore
func (lvs *ValueStore) Close() error {
	lvs.Flush()
	return lvs.bs.Close()
}

// cacheChunks looks at the Chunks reachable from v and, for each one checks if there's a hint in the cache. If there isn't, or if the hint is a self-reference, the chunk gets r set as its new hint.
func (lvs *ValueStore) cacheChunks(v Value, r hash.Hash) {
	for _, reachable := range v.Chunks() {
		hash := reachable.TargetHash()
		if cur := lvs.check(hash); cur == nil || cur.Hint().IsEmpty() || cur.Hint() == hash {
			lvs.set(hash, hintedChunk{getTargetType(reachable), r})
		}
	}
}

func (lvs *ValueStore) isPresent(r hash.Hash) (present bool) {
	if entry := lvs.check(r); entry != nil && entry.Present() {
		present = true
	}
	return
}

func (lvs *ValueStore) check(r hash.Hash) chunkCacheEntry {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()
	return lvs.cache[r]
}

func (lvs *ValueStore) set(r hash.Hash, entry chunkCacheEntry) {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()
	lvs.cache[r] = entry
}

func (lvs *ValueStore) checkAndSet(r hash.Hash, entry chunkCacheEntry) {
	if cur := lvs.check(r); cur == nil || cur.Hint().IsEmpty() {
		lvs.set(r, entry)
	}
}

func (lvs *ValueStore) chunkHintsFromCache(v Value) Hints {
	return lvs.checkChunksInCache(v, false)
}

func (lvs *ValueStore) ensureChunksInCache(v Value) {
	lvs.checkChunksInCache(v, true)
}

func (lvs *ValueStore) checkChunksInCache(v Value, readValues bool) Hints {
	hints := map[hash.Hash]struct{}{}
	for _, reachable := range v.Chunks() {
		// First, check the type cache to see if reachable is already known to be valid.
		targetHash := reachable.TargetHash()
		entry := lvs.check(targetHash)

		// If it's not already in the cache, attempt to read the value directly, which will put it and its chunks into the cache.
		if entry == nil || !entry.Present() {
			var reachableV Value
			if readValues {
				reachableV = lvs.ReadValue(targetHash)
				entry = lvs.check(targetHash)
			}
			if reachableV == nil {
				d.Exp.Fail("Attempted to write Value containing Ref to non-existent object.", "%s\n, contains ref %s, which points to a non-existent Value.", EncodedValueWithTags(v), reachable.TargetHash())
			}
		}
		if hint := entry.Hint(); !hint.IsEmpty() {
			hints[hint] = struct{}{}
		}

		targetType := getTargetType(reachable)
		d.Exp.True(entry.Type().Equals(targetType), "Value to write contains ref %s, which points to a value of a different type: %+v != %+v", reachable.TargetHash(), entry.Type(), targetType)
	}
	return hints
}

func getTargetType(refBase Ref) *Type {
	refType := refBase.Type()
	d.Chk.Equal(RefKind, refType.Kind())
	return refType.Desc.(CompoundDesc).ElemTypes[0]
}

type hintedChunk struct {
	t    *Type
	hint hash.Hash
}

func (h hintedChunk) Present() bool {
	return true
}

func (h hintedChunk) Hint() (r hash.Hash) {
	return h.hint
}

func (h hintedChunk) Type() *Type {
	return h.t
}

type absentChunk struct{}

func (a absentChunk) Present() bool {
	return false
}

func (a absentChunk) Hint() (r hash.Hash) {
	return
}

func (a absentChunk) Type() *Type {
	panic("Not reached. Should never call Type() on an absentChunk.")
}
