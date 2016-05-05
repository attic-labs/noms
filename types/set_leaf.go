package types

import (
	"crypto/sha1"
	"sort"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

type setLeaf struct {
	data    setData // sorted by Ref()
	indexOf indexOfSetFn
	t       *Type
	ref     *ref.Ref
}

type setData []Value

func newSetLeaf(t *Type, m ...Value) setLeaf {
	return setLeaf{m, getIndexFnForSetType(t), t, &ref.Ref{}}
}

func (s setLeaf) Empty() bool {
	return s.Len() == uint64(0)
}

func (s setLeaf) Len() uint64 {
	return uint64(len(s.data))
}

func (s setLeaf) Has(v Value) bool {
	idx := s.indexOf(s.data, v)
	return idx < len(s.data) && s.data[idx].Equals(v)
}

func (s setLeaf) Insert(values ...Value) Set {
	assertType(s.elemType(), values...)
	return newTypedSet(s.t, buildSetData(s.data, values, s.t)...)
}

func (s setLeaf) Remove(values ...Value) Set {
	data := copySetData(s.data)
	for _, v := range values {
		d.Chk.NotNil(v)
		idx := s.indexOf(s.data, v)
		if idx < len(s.data) && s.data[idx].Equals(v) {
			data = append(data[:idx], data[idx+1:]...)
		}
	}

	return newTypedSet(s.t, data...)
}

func (s setLeaf) Union(others ...Set) Set {
	return setUnion(s, others)
}

func (s setLeaf) Iter(cb setIterCallback) {
	for _, v := range s.data {
		if cb(v) {
			break
		}
	}
}

func (s setLeaf) IterAll(cb setIterAllCallback) {
	for _, v := range s.data {
		cb(v)
	}
}

func (s setLeaf) Filter(cb setFilterCallback) Set {
	data := setData{}
	for _, v := range s.data {
		if cb(v) {
			data = append(data, v)
		}
	}

	return newTypedSet(s.t, data...)
}

func (s setLeaf) First() Value {
	for _, v := range s.data {
		return v
	}
	return nil
}

func (s setLeaf) Ref() ref.Ref {
	return EnsureRef(s.ref, s)
}

func (s setLeaf) Equals(other Value) bool {
	return other != nil && s.t.Equals(other.Type()) && s.Ref() == other.Ref()
}

func (s setLeaf) Chunks() (chunks []Ref) {
	for _, v := range s.data {
		chunks = append(chunks, v.Chunks()...)
	}
	return
}

func (s setLeaf) ChildValues() []Value {
	return append([]Value{}, s.data...)
}

func (s setLeaf) Type() *Type {
	return s.t
}

func (s setLeaf) elemType() *Type {
	return s.t.Desc.(CompoundDesc).ElemTypes[0]
}

func copySetData(m setData) setData {
	r := make(setData, len(m))
	copy(r, m)
	return r
}

func buildSetData(old setData, values []Value, t *Type) setData {
	idxFn := getIndexFnForSetType(t)
	elemType := t.Desc.(CompoundDesc).ElemTypes[0]

	data := make(setData, len(old), len(old)+len(values))
	copy(data, old)
	for _, v := range values {
		assertType(elemType, v)
		idx := idxFn(data, v)
		if idx < len(data) && data[idx].Equals(v) {
			// We already have this fellow.
			continue
		}
		// TODO: These repeated copies suck. We're not allocating more memory (because we made the slice with the correct capacity to begin with above - yay!), but still, this is more work than necessary. Perhaps we should use an actual BST for the in-memory state, rather than a flat list.
		data = append(data, nil)
		copy(data[idx+1:], data[idx:])
		data[idx] = v
	}
	return data
}

func getIndexFnForSetType(t *Type) indexOfSetFn {
	orderByValue := t.Desc.(CompoundDesc).ElemTypes[0].IsOrdered()
	if orderByValue {
		return indexOfOrderedSetValue
	}

	return indexOfSetValue
}

func indexOfSetValue(m setData, v Value) int {
	return sort.Search(len(m), func(i int) bool {
		return !m[i].Ref().Less(v.Ref())
	})
}

func indexOfOrderedSetValue(m setData, v Value) int {
	ov := v.(OrderedValue)

	return sort.Search(len(m), func(i int) bool {
		return !m[i].(OrderedValue).Less(ov)
	})
}

func newSetLeafBoundaryChecker() boundaryChecker {
	return newBuzHashBoundaryChecker(setWindowSize, sha1.Size, setPattern, func(item sequenceItem) []byte {
		digest := item.(Value).Ref().Digest()
		return digest[:]
	})
}

func makeSetLeafChunkFn(t *Type, vr ValueReader) makeChunkFn {
	return func(items []sequenceItem) (sequenceItem, Value) {
		setData := make([]Value, len(items), len(items))

		for i, v := range items {
			setData[i] = v.(Value)
		}

		setLeaf := newSetLeaf(t, setData...)

		var indexValue Value
		if len(setData) > 0 {
			lastValue := setData[len(setData)-1]
			if isSequenceOrderedByIndexedType(t) {
				indexValue = lastValue
			} else {
				indexValue = NewTypedRefFromValue(lastValue)
			}
		}

		return newMetaTuple(indexValue, setLeaf, NewTypedRefFromValue(setLeaf), uint64(len(items))), setLeaf
	}
}

func (s setLeaf) sequenceCursorAtFirst() *sequenceCursor {
	return &sequenceCursor{
		nil,
		s.data,
		0,
		len(s.data),
		func(parent sequenceItem, idx int) sequenceItem {
			return s.data[idx]
		},
		func(reference sequenceItem) (sequence sequenceItem, length int) {
			panic("unreachable")
		},
	}
}

func (s setLeaf) valueReader() ValueReader {
	return nil
}
