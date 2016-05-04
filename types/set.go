package types

import "github.com/attic-labs/noms/d"

type Set interface {
	Collection
	First() Value
	Has(key Value) bool
	Insert(values ...Value) Set
	Remove(values ...Value) Set
	Union(others ...Set) Set
	Iter(cb setIterCallback)
	IterAll(cb setIterAllCallback)
	IterAllP(concurrency int, f setIterAllCallback)
	Filter(cb setFilterCallback) Set
	elemType() *Type
	sequenceCursorAtFirst() *sequenceCursor
	valueReader() ValueReader
}

type indexOfSetFn func(m setData, v Value) int
type setIterCallback func(v Value) bool
type setIterAllCallback func(v Value)
type setFilterCallback func(v Value) (keep bool)

var setType = MakeSetType(ValueType)

func NewSet(v ...Value) Set {
	return NewTypedSet(setType, v...)
}

func NewTypedSet(t *Type, v ...Value) Set {
	d.Chk.Equal(SetKind, t.Kind(), "Invalid type. Expected:SetKind, found: %s", t.Describe())
	return newTypedSet(t, buildSetData(setData{}, v, t)...)
}

func newTypedSet(t *Type, data ...Value) Set {
	seq := newEmptySequenceChunker(makeSetLeafChunkFn(t, nil), newOrderedMetaSequenceChunkFn(t, nil), newSetLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	for _, v := range data {
		seq.Append(v)
	}

	return seq.Done().(Set)
}

func setUnion(set Set, others []Set) Set {
	// TODO: This can be done more efficiently by realizing that if two sets have the same meta tuple we only have to traverse one of the subtrees. Bug 794
	if len(others) == 0 {
		return set
	}
	assertSetsSameType(set, others...)

	vr := set.valueReader()
	for _, s := range others {
		// This both tries to find a ValueReader to use for the new Set, and asserts that if it does,
		// it's the same (or nil) for all Sets. I.e. it should be possible to take the union of a Set
		// without a ValueReader (constructed in memory but not written yet) and one with a ValueReader
		// (from that source) but not with a *different* ValueReader because this would get complicated.
		if vr == nil {
			vr = s.valueReader()
		}
		d.Chk.True(vr == nil || vr == s.valueReader())
	}

	tr := set.Type()
	seq := newEmptySequenceChunker(makeSetLeafChunkFn(tr, vr), newOrderedMetaSequenceChunkFn(tr, vr), newSetLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	var lessFunction func(a, b sequenceItem) bool
	if isSequenceOrderedByIndexedType(tr) {
		lessFunction = func(a, b sequenceItem) bool {
			return a.(OrderedValue).Less(b.(OrderedValue))
		}
	} else {
		lessFunction = func(a, b sequenceItem) bool {
			return a.(Value).Ref().Less(b.(Value).Ref())
		}
	}

	smallest := func(cursors map[*sequenceCursor]bool) (smallestCursor *sequenceCursor, smallestItem sequenceItem) {
		for cursor := range cursors {
			currentItem := cursor.current()
			if smallestCursor == nil || lessFunction(currentItem, smallestItem) {
				smallestCursor = cursor
				smallestItem = currentItem
			}
		}
		return
	}

	cursors := make(map[*sequenceCursor]bool, len(others)+1)
	if !set.Empty() {
		cursor := set.sequenceCursorAtFirst()
		cursors[cursor] = true
	}
	for _, s := range others {
		if !s.Empty() {
			cursor := s.sequenceCursorAtFirst()
			cursors[cursor] = true
		}
	}

	var last Value
	for len(cursors) > 0 {
		smallestCursor, smallestItem := smallest(cursors)
		d.Chk.NotNil(smallestCursor)

		// Don't add same value twice
		if last == nil || !last.Equals(smallestItem.(Value)) {
			seq.Append(smallestItem)
			last = smallestItem.(Value)
		}

		if !smallestCursor.advance() {
			delete(cursors, smallestCursor)
		}
	}

	return seq.Done().(Set)
}
