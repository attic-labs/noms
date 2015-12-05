package types

import (
	"crypto/sha1"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

const (
	// The window size to use for computing the rolling hash.
	listWindowSize = 64
	listPattern    = uint32(1<<6 - 1) // Average size of 64 elements
)

type compoundList struct {
	metaSequenceObject
	ref *ref.Ref
	cs  chunks.ChunkStore
}

func buildCompoundList(tuples metaSequenceData, t Type, cs chunks.ChunkStore) Value {
	cl := compoundList{metaSequenceObject{tuples, t}, &ref.Ref{}, cs}
	return valueFromType(cs, cl, t)
}

func listAsSequenceItems(ls listLeaf) []sequenceItem {
	items := make([]sequenceItem, len(ls.values))
	for i, v := range ls.values {
		items[i] = v
	}
	return items
}

func init() {
	registerMetaValue(ListKind, buildCompoundList)
}

func (cl compoundList) Equals(other Value) bool {
	return other != nil && cl.t.Equals(other.Type()) && cl.Ref() == other.Ref()
}

func (cl compoundList) Ref() ref.Ref {
	return EnsureRef(cl.ref, cl)
}

func (cl compoundList) Len() uint64 {
	return cl.tuples[len(cl.tuples)-1].uint64Value()
}

func (cl compoundList) Empty() bool {
	d.Chk.True(cl.Len() > 0) // A compound object should never be empty.
	return false
}

func (cl compoundList) cursorAt(idx uint64) (*sequenceCursor, listLeaf, uint64) {
	d.Chk.True(idx <= cl.Len())
	cursor, leaf := newMetaSequenceCursor(cl, cl.cs)

	chunkStart := cursor.seek(func(carry interface{}, mt sequenceItem) bool {
		return idx < uint64(carry.(Uint64))+uint64(mt.(metaTuple).value.(Uint64))
	}, func(carry interface{}, prev, current sequenceItem) interface{} {
		pv := uint64(0)
		if prev != nil {
			pv = uint64(prev.(metaTuple).value.(Uint64))
		}
		return Uint64(uint64(carry.(Uint64)) + pv)
	}, Uint64(0))

	current := cursor.current().(metaTuple)
	if current.ref != leaf.Ref() {
		leaf = readMetaTupleValue(cursor.current(), cl.cs)
	}

	return cursor, leaf.(listLeaf), uint64(chunkStart.(Uint64))
}

func (cl compoundList) Get(idx uint64) Value {
	_, l, start := cl.cursorAt(idx)
	return l.Get(idx - start)
}

func (cl compoundList) IterAllP(concurrency int, f listIterAllFunc) {
	panic("not implemented")
}

func (cl compoundList) Slice(start uint64, end uint64) List {
	panic("not implemented")
}

func (cl compoundList) Map(mf MapFunc) []interface{} {
	panic("not implemented")
}

func (cl compoundList) MapP(concurrency int, mf MapFunc) []interface{} {
	panic("not implemented")
}

func (cl compoundList) Set(idx uint64, v Value) List {
	panic("not implemented")
}

func (cl compoundList) Append(vs ...Value) List {
	seq := cl.sequenceChunkerAtIndex(cl.Len())
	for _, v := range vs {
		seq.Append(v)
	}
	return seq.Done().(List)
}

func (cl compoundList) sequenceChunkerAtIndex(idx uint64) *sequenceChunker {
	metaCur, leaf, start := cl.cursorAt(idx)
	cur := &sequenceCursor{metaCur, leaf, int(idx - start), len(leaf.values), func(list sequenceItem, idx int) sequenceItem {
		return list.(listLeaf).values[idx]
	}, func(mt sequenceItem) (sequenceItem, int) {
		list := readMetaTupleValue(mt, cl.cs).(listLeaf)
		return list, len(list.values)
	}}

	return newSequenceChunker(cur, makeListLeafChunkFn(cl.t, cl.cs), newMetaSequenceChunkFn(cl.t, cl.cs), normalizeChunkNoop, normalizeMetaSequenceChunk, newListLeafBoundaryChecker(), newMetaSequenceBoundaryChecker)
}

func (cl compoundList) Filter(cb listFilterCallback) List {
	panic("not implemented")
}

func (cl compoundList) Insert(idx uint64, v ...Value) List {
	panic("not implemented")
}

func (cl compoundList) Remove(start uint64, end uint64) List {
	panic("not implemented")
}

func (cl compoundList) RemoveAt(idx uint64) List {
	panic("not implemented")
}

func (cl compoundList) Iter(f listIterFunc) {
	start := uint64(0)

	iterateMetaSequenceLeaf(cl, cl.cs, func(l Value) bool {
		list := l.(listLeaf)
		for i, v := range list.values {
			if f(v, start+uint64(i)) {
				return true
			}
		}
		start += list.Len()
		return false
	})
}

func (cl compoundList) IterAll(f listIterAllFunc) {
	start := uint64(0)

	iterateMetaSequenceLeaf(cl, cl.cs, func(l Value) bool {
		list := l.(listLeaf)
		for i, v := range list.values {
			f(v, start+uint64(i))
		}
		start += list.Len()
		return false
	})
}

func newListLeafBoundaryChecker() boundaryChecker {
	return newBuzHashBoundaryChecker(listWindowSize, sha1.Size, listPattern, func(item sequenceItem) []byte {
		digest := item.(Value).Ref().Digest()
		return digest[:]
	})
}

func makeListLeafChunkFn(t Type, cs chunks.ChunkStore) makeChunkFn {
	return func(items []sequenceItem) (sequenceItem, Value) {
		values := make([]Value, len(items))

		for i, v := range items {
			values[i] = v.(Value)
		}

		list := valueFromType(cs, newListLeaf(cs, t, values...), t)
		ref := WriteValue(list, cs)
		return metaTuple{ref, Uint64(len(values))}, list
	}
}
