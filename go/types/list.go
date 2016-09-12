// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type List struct {
	seq indexedSequence
	h   *hash.Hash
}

func newList(seq indexedSequence) List {
	return List{seq, &hash.Hash{}}
}

// NewList creates a new List where the type is computed from the elements in the list, populated with values, chunking if and when needed.
func NewList(values ...Value) List {
	seq := newEmptySequenceChunker(nil, nil, makeListLeafChunkFn(nil), newIndexedMetaSequenceChunkFn(ListKind, nil), hashValueBytes)
	for _, v := range values {
		seq.Append(v)
	}
	return newList(seq.Done().(indexedSequence))
}

// NewStreamingList creates a new List, populated with values, chunking if and when needed. As chunks are created, they're written to vrw -- including the root chunk of the list. Caller should close the values channel to read completed list.
func NewStreamingList(vrw ValueReadWriter, values <-chan Value) <-chan List {
	out := make(chan List)
	go func() {
		seq := newEmptySequenceChunker(vrw, vrw, makeListLeafChunkFn(vrw), newIndexedMetaSequenceChunkFn(ListKind, vrw), hashValueBytes)
		for v := range values {
			seq.Append(v)
		}
		out <- newList(seq.Done().(indexedSequence))
		close(out)
	}()
	return out
}

// Collection interface
func (l List) Len() uint64 {
	return l.seq.numLeaves()
}

func (l List) Empty() bool {
	return l.Len() == 0
}

func (l List) sequence() sequence {
	return l.seq
}

func (l List) hashPointer() *hash.Hash {
	return l.h
}

// Value interface
func (l List) Equals(other Value) bool {
	return l.Hash() == other.Hash()
}

func (l List) Less(other Value) bool {
	return valueLess(l, other)
}

func (l List) Hash() hash.Hash {
	if l.h.IsEmpty() {
		*l.h = getHash(l)
	}

	return *l.h
}

func (l List) ChildValues() []Value {
	values := make([]Value, l.Len())
	l.IterAll(func(v Value, idx uint64) {
		values[idx] = v
	})
	return values
}

func (l List) Chunks() []Ref {
	return l.seq.Chunks()
}

func (l List) Type() *Type {
	return l.seq.Type()
}

func (l List) Get(idx uint64) Value {
	d.Chk.True(idx < l.Len())
	cur := newCursorAtIndex(l.seq, idx)
	return cur.current().(Value)
}

type MapFunc func(v Value, index uint64) interface{}

func (l List) Map(mf MapFunc) []interface{} {
	idx := uint64(0)
	cur := newCursorAtIndex(l.seq, idx)

	results := make([]interface{}, 0, l.Len())
	cur.iter(func(v interface{}) bool {
		res := mf(v.(Value), uint64(idx))
		results = append(results, res)
		idx++
		return false
	})
	return results
}

func (l List) elemType() *Type {
	return l.seq.Type().Desc.(CompoundDesc).ElemTypes[0]
}

func (l List) Set(idx uint64, v Value) List {
	d.Chk.True(idx < l.Len())
	return l.Splice(idx, 1, v)
}

func (l List) Append(vs ...Value) List {
	return l.Splice(l.Len(), 0, vs...)
}

func (l List) Splice(idx uint64, deleteCount uint64, vs ...Value) List {
	if deleteCount == 0 && len(vs) == 0 {
		return l
	}

	d.Chk.True(idx <= l.Len())
	d.Chk.True(idx+deleteCount <= l.Len())

	cur := newCursorAtIndex(l.seq, idx)
	ch := l.newChunker(cur)
	for deleteCount > 0 {
		ch.Skip()
		deleteCount--
	}

	for _, v := range vs {
		ch.Append(v)
	}
	return newList(ch.Done().(indexedSequence))
}

func (l List) Insert(idx uint64, vs ...Value) List {
	return l.Splice(idx, 0, vs...)
}

// Concat returns new list comprised of this joined with other. It only needs to
// visit the rightmost prolly tree chunks of this list, and the leftmost prolly
// tree chunks of other.
func (l List) Concat(other List) List {
	if l.Empty() {
		return other
	}
	if other.Empty() {
		return l
	}
	d.Chk.Equal(l.seq.valueReader(), other.seq.valueReader())

	seq := concat(l.seq, other.seq, func(cur *sequenceCursor) *sequenceChunker {
		return l.newChunker(cur)
	})
	return newList(seq.(indexedSequence))
}

func (l List) Remove(start uint64, end uint64) List {
	d.Chk.True(start <= end)
	return l.Splice(start, end-start)
}

func (l List) RemoveAt(idx uint64) List {
	return l.Splice(idx, 1)
}

type listIterFunc func(v Value, index uint64) (stop bool)

func (l List) Iter(f listIterFunc) {
	idx := uint64(0)
	cur := newCursorAtIndex(l.seq, idx)
	cur.iter(func(v interface{}) bool {
		if f(v.(Value), uint64(idx)) {
			return true
		}
		idx++
		return false
	})
}

type listIterAllFunc func(v Value, index uint64)

func (l List) IterAll(f listIterAllFunc) {
	idx := uint64(0)
	cur := newCursorAtIndex(l.seq, idx)
	cur.iter(func(v interface{}) bool {
		f(v.(Value), uint64(idx))
		idx++
		return false
	})
}

func (l List) Iterator() ListIterator {
	return l.IteratorAt(0)
}

func (l List) IteratorAt(index uint64) ListIterator {
	return ListIterator{newCursorAtIndex(l.seq, index)}
}

func (l List) Diff(last List, changes chan<- Splice, closeChan <-chan struct{}) {
	l.DiffWithLimit(last, changes, closeChan, DEFAULT_MAX_SPLICE_MATRIX_SIZE)
}

func (l List) DiffWithLimit(last List, changes chan<- Splice, closeChan <-chan struct{}, maxSpliceMatrixSize uint64) {
	if l.Equals(last) {
		return
	}
	lLen, lastLen := l.Len(), last.Len()
	if lLen == 0 {
		changes <- Splice{0, lastLen, 0, 0} // everything removed
		return
	}
	if lastLen == 0 {
		changes <- Splice{0, 0, lLen, 0} // everything added
		return
	}

	lastCur := newCursorAtIndex(last.seq, 0)
	lCur := newCursorAtIndex(l.seq, 0)
	indexedSequenceDiff(last.seq, lastCur.depth(), 0, l.seq, lCur.depth(), 0, changes, closeChan, maxSpliceMatrixSize)
}

func (l List) newChunker(cur *sequenceCursor) *sequenceChunker {
	return newSequenceChunker(cur, l.seq.valueReader(), nil, makeListLeafChunkFn(l.seq.valueReader()), newIndexedMetaSequenceChunkFn(ListKind, l.seq.valueReader()), hashValueBytes)
}

// If |sink| is not nil, chunks will be eagerly written as they're created. Otherwise they are
// written when the root is written.
func makeListLeafChunkFn(vr ValueReader) makeChunkFn {
	return func(items []sequenceItem) (Collection, orderedKey, uint64) {
		values := make([]Value, len(items))

		for i, v := range items {
			values[i] = v.(Value)
		}

		list := newList(newListLeafSequence(vr, values...))
		return list, orderedKeyFromInt(len(values)), uint64(len(values))
	}
}
