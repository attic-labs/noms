// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

const (
	mapWindowSize = 1
	mapPattern    = uint32(1<<6 - 1) // Average size of 64 elements
)

type Map struct {
	seq orderedSequence
	h   *hash.Hash
}

func newMap(seq orderedSequence) Map {
	return Map{seq, &hash.Hash{}}
}

func NewMap(kv ...Value) Map {
	entries := buildMapData(kv)
	seq := newEmptySequenceChunker(nil, makeMapLeafChunkFn(nil), newOrderedMetaSequenceChunkFn(MapKind, nil), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	for _, entry := range entries {
		seq.Append(entry)
	}

	return newMap(seq.Done(nil).(orderedSequence))
}

func NewStreamingMap(vrw ValueReadWriter, kvs <-chan Value) <-chan Map {
	var k Value

	outChan := make(chan Map)
	go func() {
		mx := newMutator(vrw)

		for v := range kvs {
			if k == nil {
				k = v
				continue
			}
			mx.Set(k, v)
			k = nil
		}

		d.Chk.True(k == nil)
		outChan <- mx.Finish()
	}()
	return outChan
}

func (m Map) Diff(last Map, changes chan<- ValueChanged, closeChan <-chan struct{}) {
	orderedSequenceDiffBest(last.sequence().(orderedSequence), m.sequence().(orderedSequence), changes, closeChan)
}

// Collection interface
func (m Map) Len() uint64 {
	return m.seq.numLeaves()
}

func (m Map) Empty() bool {
	return m.Len() == 0
}

func (m Map) sequence() sequence {
	return m.seq
}

func (m Map) hashPointer() *hash.Hash {
	return m.h
}

// Value interface
func (m Map) Equals(other Value) bool {
	return other != nil && m.Hash() == other.Hash()
}

func (m Map) Less(other Value) bool {
	return valueLess(m, other)
}

func (m Map) Hash() hash.Hash {
	if m.h.IsEmpty() {
		*m.h = getHash(m)
	}

	return *m.h
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

func (m Map) Type() *Type {
	return m.seq.Type()
}

func (m Map) First() (Value, Value) {
	cur := newCursorAt(m.seq, emptyKey, false, false)
	if !cur.valid() {
		return nil, nil
	}
	entry := cur.current().(mapEntry)
	return entry.key, entry.value
}

func (m Map) MaybeGet(key Value) (v Value, ok bool) {
	cur := newCursorAtValue(m.seq, key, false, false)
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

	k, v, tail := kv[0], kv[1], kv[2:]

	cur, found := m.getCursorAtValue(k)
	deleteCount := uint64(0)
	if found {
		deleteCount = 1
	}
	return m.splice(cur, deleteCount, mapEntry{k, v}).SetM(tail...)
}

func (m Map) Remove(k Value) Map {
	if cur, found := m.getCursorAtValue(k); found {
		return m.splice(cur, 1)
	}
	return m
}

func (m Map) splice(cur *sequenceCursor, deleteCount uint64, vs ...mapEntry) Map {
	ch := newSequenceChunker(cur, nil, makeMapLeafChunkFn(m.seq.valueReader()), newOrderedMetaSequenceChunkFn(MapKind, m.seq.valueReader()), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)
	for deleteCount > 0 {
		ch.Skip()
		deleteCount--
	}

	for _, v := range vs {
		ch.Append(v)
	}
	return newMap(ch.Done(nil).(orderedSequence))
}

func (m Map) getCursorAtValue(v Value) (cur *sequenceCursor, found bool) {
	cur = newCursorAtValue(m.seq, v, true, false)
	found = cur.idx < cur.seq.seqLen() && cur.current().(mapEntry).key.Equals(v)
	return
}

func (m Map) Has(key Value) bool {
	cur := newCursorAtValue(m.seq, key, false, false)
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
	cur := newCursorAt(m.seq, emptyKey, false, false)
	cur.iter(func(v interface{}) bool {
		entry := v.(mapEntry)
		return cb(entry.key, entry.value)
	})
}

type mapIterAllCallback func(key, value Value)

func (m Map) IterAll(cb mapIterAllCallback) {
	cur := newCursorAt(m.seq, emptyKey, false, false)
	cur.iter(func(v interface{}) bool {
		entry := v.(mapEntry)
		cb(entry.key, entry.value)
		return false
	})
}

func (m Map) elemTypes() []*Type {
	return m.Type().Desc.(CompoundDesc).ElemTypes
}

func buildMapData(values []Value) mapEntrySlice {
	if len(values) == 0 {
		return mapEntrySlice{}
	}

	// Sadly, d.Chk.Equals() costs too much. BUG #83
	d.Chk.True(0 == len(values)%2, "Must specify even number of key/value pairs")
	kvs := make(mapEntrySlice, len(values)/2)

	for i := 0; i < len(values); i += 2 {
		entry := mapEntry{values[i], values[i+1]}
		kvs[i/2] = entry
	}

	uniqueSorted := make(mapEntrySlice, 0, len(kvs))
	sort.Stable(kvs)
	last := kvs[0]
	for i := 1; i < len(kvs); i++ {
		kv := kvs[i]
		if !kv.key.Equals(last.key) {
			uniqueSorted = append(uniqueSorted, last)
		}

		last = kv
	}

	return append(uniqueSorted, last)
}

func newMapLeafBoundaryChecker() boundaryChecker {
	return newBuzHashBoundaryChecker(mapWindowSize, hash.ByteLen, mapPattern, func(item sequenceItem) []byte {
		digest := item.(mapEntry).key.Hash().Digest()
		return digest[:]
	})
}

// If |vw| is not nil, chunks will be eagerly written as they're created. Otherwise they are
// written when the root is written.
func makeMapLeafChunkFn(vr ValueReader) makeChunkFn {
	return func(items []sequenceItem) (Collection, orderedKey, uint64) {
		mapData := make([]mapEntry, len(items), len(items))

		for i, v := range items {
			mapData[i] = v.(mapEntry)
		}

		m := newMap(newMapLeafSequence(vr, mapData...))
		var key orderedKey
		if len(mapData) > 0 {
			key = newOrderedKey(mapData[len(mapData)-1].key)
		}
		return m, key, uint64(len(items))
	}
}
