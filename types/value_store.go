package types

import (
	"sync"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

// ValueStore provides methods to read and write Noms Values to a BatchStore. It validates Values as they are written, but does not guarantee that these Values are persisted to the BatchStore until a subsequent Flush. or Close.
// Currently, WriteValue validates the following properties of a Value v:
// - v can be correctly serialized and its Ref taken
// - all Refs in v point to a Value that can be read from this ValueStore
// - all Refs in v point to a Value of the correct Type
type ValueStore struct {
	bs    BatchStore
	cache map[ref.Ref]chunkCacheEntry
	mu    *sync.Mutex
}

type chunkCacheEntry interface {
	Present() bool
	Hint() ref.Ref
	Type() Type
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
	return &ValueStore{bs, map[ref.Ref]chunkCacheEntry{}, &sync.Mutex{}}
}

// ReadValue reads and decodes a value from lvs. It is not considered an error for the requested chunk to be empty; in this case, the function simply returns nil.
func (lvs *ValueStore) ReadValue(r ref.Ref) Value {
	v := DecodeChunk(lvs.bs.Get(r), lvs)

	var entry chunkCacheEntry = absentChunk{}
	if v != nil {
		lvs.cacheChunks(v, r)
		// r is trivially a hint for v, so consider putting that in the cache. If we got to v by reading some higher-level chunk, this entry gets dropped on the floor because r already has a hint in the cache. If we later read some other chunk that references v, cacheChunks will overwrite this with a hint pointing to that chunk.
		// If we don't do this, top-level Values that get read but not written -- such as the existing Head of a DataStore upon a Commit -- can be erroneously left out during a pull.
		entry = hintedChunk{v.Type(), r}
	}
	if cur := lvs.check(r); cur == nil || cur.Hint().IsEmpty() {
		lvs.set(r, entry)
	}
	return v
}

// WriteValue takes a Value, schedules it to be written it to lvs, and returns an appropriately-typed types.RefBase. v is not guaranteed to be actually written until after Flush().
func (lvs *ValueStore) WriteValue(v Value) RefBase {
	if v == nil {
		return nil
	}

	targetRef := v.Ref()
	r := refFromType(targetRef, MakeRefType(v.Type()))
	if lvs.isPresent(targetRef) {
		return r
	}

	// Encoding v causes any child chunks, e.g. internal nodes if v is a meta sequence, to get written. That needs to happen before we try to validate v.
	c := EncodeValue(v, lvs)

	hints := lvs.checkChunksInCache(v)
	lvs.set(targetRef, presentChunk(v.Type()))
	lvs.bs.SchedulePut(c, hints)

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
func (lvs *ValueStore) cacheChunks(v Value, r ref.Ref) {
	for _, reachable := range v.Chunks() {
		hash := reachable.TargetRef()
		if cur := lvs.check(hash); cur == nil || cur.Hint().IsEmpty() || cur.Hint() == hash {
			lvs.set(hash, hintedChunk{getTargetType(reachable), r})
			// Code-genned Packages are side-loaded when reading Values for performance reasons. This means that they won't pass through the ReadValue() codepath above, which means that they won't have their Chunks added to the cache. So, if reachable is a RefOfPackage, go look the package up in the PackageRegistry and recursively add its Chunks to the cache.
			if _, ok := reachable.(Ref); ok {
				if p := LookupPackage(hash); p != nil {
					lvs.cacheChunks(p, hash)
				}
			}
		}
	}
}

func (lvs *ValueStore) isPresent(r ref.Ref) (present bool) {
	if entry := lvs.check(r); entry != nil && entry.Present() {
		present = true
	}
	return
}

func (lvs *ValueStore) check(r ref.Ref) chunkCacheEntry {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()
	return lvs.cache[r]
}

func (lvs *ValueStore) set(r ref.Ref, entry chunkCacheEntry) {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()
	lvs.cache[r] = entry
}

func (lvs *ValueStore) checkAndSet(r ref.Ref, entry chunkCacheEntry) {
	if cur := lvs.check(r); cur == nil || cur.Hint().IsEmpty() {
		lvs.set(r, entry)
	}
}

func (lvs *ValueStore) checkChunksInCache(v Value) map[ref.Ref]struct{} {
	hints := map[ref.Ref]struct{}{}
	for _, reachable := range v.Chunks() {
		entry := lvs.check(reachable.TargetRef())
		d.Exp.True(entry != nil && entry.Present(), "Value to write -- Type %s -- contains ref %s, which points to a non-existent Value.", v.Type().Describe(), reachable.TargetRef())
		if hint := entry.Hint(); !hint.IsEmpty() {
			hints[hint] = struct{}{}
		}

		// BUG 1121
		// It's possible that entry.Type() will be simply 'Value', but that 'reachable' is actually a
		// properly-typed object -- that is, a Ref to some specific Type. The Exp below would fail,
		// though it's possible that the Type is actually correct. We wouldn't be able to verify
		// without reading it, though, so we'll dig into this later.
		targetType := getTargetType(reachable)
		if targetType.Equals(MakePrimitiveType(ValueKind)) {
			continue
		}
		d.Exp.True(entry.Type().Equals(targetType), "Value to write contains ref %s, which points to a value of a different type: %+v != %+v", reachable.TargetRef(), entry.Type(), targetType)
	}
	return hints
}

func getTargetType(refBase RefBase) Type {
	refType := refBase.Type()
	d.Chk.Equal(RefKind, refType.Kind())
	return refType.Desc.(CompoundDesc).ElemTypes[0]
}

type presentChunk Type

func (p presentChunk) Present() bool {
	return true
}

func (p presentChunk) Hint() (r ref.Ref) {
	return
}

func (p presentChunk) Type() Type {
	return Type(p)
}

type hintedChunk struct {
	t    Type
	hint ref.Ref
}

func (h hintedChunk) Present() bool {
	return true
}

func (h hintedChunk) Hint() (r ref.Ref) {
	return h.hint
}

func (h hintedChunk) Type() Type {
	return h.t
}

type absentChunk struct{}

func (a absentChunk) Present() bool {
	return false
}

func (a absentChunk) Hint() (r ref.Ref) {
	return
}

func (a absentChunk) Type() Type {
	panic("Not reached. Should never call Type() on an absentChunk.")
}
