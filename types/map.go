package types

import (
	"crypto/sha1"
	"sort"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

const (
	mapWindowSize = 1
	mapPattern    = uint32(1<<6 - 1) // Average size of 64 elements
)

type Map struct {
	seq orderedSequence
	ref *ref.Ref
}

var mapType = MakeMapType(ValueType, ValueType)

func newMap(seq orderedSequence) Map {
	return Map{seq, &ref.Ref{}}
}

func NewMap(kv ...Value) Map {
	return NewTypedMap(mapType, kv...)
}

func NewTypedMap(t *Type, kv ...Value) Map {
	d.Chk.Equal(MapKind, t.Kind(), "Invalid type. Expected: MapKind, found: %s", t.Describe())
	return newTypedMap(t, buildMapData([]mapEntry{}, kv, t)...)
}

func newTypedMap(t *Type, entries ...mapEntry) Map {
	seq := newEmptySequenceChunker(makeMapLeafChunkFn(t, nil), newOrderedMetaSequenceChunkFn(t, nil), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	for _, entry := range entries {
		seq.Append(entry)
	}

	return seq.Done().(Map)
}

func (m Map) Type() *Type {
	return m.seq.Type()
}

func (m Map) Equals(other Value) bool {
	return other != nil && m.Ref() == other.Ref()
}

func (m Map) Ref() ref.Ref {
	return EnsureRef(m.ref, m)
}

func (m Map) Len() uint64 {
	return m.seq.numLeaves()
}

func (m Map) Empty() bool {
	return m.Len() == 0
}

func (m Map) ChildValues() (values []Value) {
	m.IterAll(func(k, v Value) {
		values = append(values, k, v)
	})
	return
}

func (m Map) Chunks() []Ref {
	return m.seq.Chunks()
}

func (m Map) sequence() sequence {
	return m.seq
}

func (m Map) First() (Value, Value) {
	cur := newCursorAtKey(m.seq, nil, false, false)
	if !cur.valid() {
		return nil, nil
	}
	entry := cur.current().(mapEntry)
	return entry.key, entry.value
}

func (m Map) MaybeGet(key Value) (v Value, ok bool) {
	cur := newCursorAtKey(m.seq, key, false, false)
	if !cur.valid() {
		return nil, false
	}
	entry := cur.current().(mapEntry)
	if !entry.key.Equals(key) {
		return nil, false
	}

	return entry.value, true
}

func (m Map) Set(key Value, val Value) Map {
	return m.SetM(key, val)
}

func (m Map) SetM(kv ...Value) Map {
	if len(kv) == 0 {
		return m
	}
	d.Chk.True(len(kv)%2 == 0)

	assertMapElemTypes(m, kv...)

	k, v, tail := kv[0], kv[1], kv[2:]

	ch, found := mapSequenceChunkerAtKey(m.seq, k)
	if found {
		ch.Skip()
	}
	ch.Append(mapEntry{k, v})
	return ch.Done().(Map).SetM(tail...)
}

func (m Map) Remove(k Value) Map {
	if ch, found := mapSequenceChunkerAtKey(m.seq, k); found {
		ch.Skip()
		return ch.Done().(Map)
	}
	return m
}

func mapSequenceChunkerAtKey(seq orderedSequence, k Value) (*sequenceChunker, bool) {
	cur := newCursorAtKey(seq, k, true, false)
	found := cur.idx < cur.seq.seqLen() && cur.current().(mapEntry).key.Equals(k)
	ch := newSequenceChunker(cur, makeMapLeafChunkFn(seq.Type(), seq.valueReader()), newOrderedMetaSequenceChunkFn(seq.Type(), seq.valueReader()), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)
	return ch, found
}

type mapFilterCallback func(key, value Value) (keep bool)

func (m Map) Filter(cb mapFilterCallback) Map {
	seq := m.seq
	ch := newEmptySequenceChunker(makeMapLeafChunkFn(seq.Type(), seq.valueReader()), newOrderedMetaSequenceChunkFn(seq.Type(), seq.valueReader()), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	m.IterAll(func(k, v Value) {
		if cb(k, v) {
			ch.Append(mapEntry{k, v})
		}
	})

	return ch.Done().(Map)
}

func (m Map) Has(key Value) bool {
	cur := newCursorAtKey(m.seq, key, false, false)
	if !cur.valid() {
		return false
	}
	entry := cur.current().(mapEntry)
	return entry.key.Equals(key)
}

func (m Map) Get(key Value) Value {
	v, _ := m.MaybeGet(key)
	return v
}

type mapIterCallback func(key, value Value) (stop bool)

func (m Map) Iter(cb mapIterCallback) {
	cur := newCursorAtKey(m.seq, nil, false, false)
	cur.iter(func(v interface{}) bool {
		entry := v.(mapEntry)
		return cb(entry.key, entry.value)
	})
}

type mapIterAllCallback func(key, value Value)

func (m Map) IterAll(cb mapIterAllCallback) {
	cur := newCursorAtKey(m.seq, nil, false, false)
	cur.iter(func(v interface{}) bool {
		entry := v.(mapEntry)
		cb(entry.key, entry.value)
		return false
	})
}

func (m Map) elemTypes() []*Type {
	return m.Type().Desc.(CompoundDesc).ElemTypes
}

func buildMapData(oldData []mapEntry, values []Value, t *Type) []mapEntry {
	idxFn := getIndexFnForMapType(t)
	elemTypes := t.Desc.(CompoundDesc).ElemTypes

	// Sadly, d.Chk.Equals() costs too much. BUG #83
	d.Chk.True(0 == len(values)%2, "Must specify even number of key/value pairs")

	data := make([]mapEntry, len(oldData), len(oldData)+len(values))
	copy(data, oldData)
	for i := 0; i < len(values); i += 2 {
		k := values[i]
		v := values[i+1]
		assertType(elemTypes[0], k)
		assertType(elemTypes[1], v)
		idx := idxFn(data, k)
		if idx < len(data) && data[idx].key.Equals(k) {
			if !data[idx].value.Equals(v) {
				data[idx] = mapEntry{k, v}
			}
			continue
		}

		// TODO: These repeated copies suck. We're not allocating more memory (because we made the slice with the correct capacity to begin with above - yay!), but still, this is more work than necessary. Perhaps we should use an actual BST for the in-memory state, rather than a flat list.
		data = append(data, mapEntry{})
		copy(data[idx+1:], data[idx:])
		data[idx] = mapEntry{k, v}
	}
	return data
}

type indexOfMapFn func(m []mapEntry, v Value) int

func getIndexFnForMapType(t *Type) indexOfMapFn {
	orderByValue := t.Desc.(CompoundDesc).ElemTypes[0].IsOrdered()
	if orderByValue {
		return indexOfOrderedMapValue
	}

	return indexOfMapValue
}

func indexOfMapValue(m []mapEntry, v Value) int {
	return sort.Search(len(m), func(i int) bool {
		return !m[i].key.Ref().Less(v.Ref())
	})
}

func indexOfOrderedMapValue(m []mapEntry, v Value) int {
	ov := v.(OrderedValue)

	return sort.Search(len(m), func(i int) bool {
		return !m[i].key.(OrderedValue).Less(ov)
	})
}

func newMapLeafBoundaryChecker() boundaryChecker {
	return newBuzHashBoundaryChecker(mapWindowSize, sha1.Size, mapPattern, func(item sequenceItem) []byte {
		digest := item.(mapEntry).key.Ref().Digest()
		return digest[:]
	})
}

func makeMapLeafChunkFn(t *Type, vr ValueReader) makeChunkFn {
	return func(items []sequenceItem) (sequenceItem, Value) {
		mapData := make([]mapEntry, len(items), len(items))

		for i, v := range items {
			mapData[i] = v.(mapEntry)
		}

		m := newMap(newMapLeafSequence(t, vr, mapData...))

		var indexValue Value
		if len(mapData) > 0 {
			indexValue = mapData[len(mapData)-1].key
			if !isSequenceOrderedByIndexedType(t) {
				indexValue = NewTypedRefFromValue(indexValue)
			}
		}

		return newMetaTuple(indexValue, m, NewTypedRefFromValue(m), uint64(len(items))), m
	}
}
