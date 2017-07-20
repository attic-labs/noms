// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/hash"
)

const (
	objectWindowSize          = 8
	orderedSequenceWindowSize = 1
	objectPattern             = uint32(1<<6 - 1) // Average size of 64 elements
)

var emptyKey = orderedKey{}

func newMetaTuple(ref Ref, key orderedKey, numLeaves uint64, child Collection) metaTuple {
	d.PanicIfFalse(Ref{} != ref)
	return metaTuple{ref, key, numLeaves, child}
}

// metaTuple is a node in a Prolly Tree, consisting of data in the node (either tree leaves or other metaSequences), and a Value annotation for exploring the tree (e.g. the largest item if this an ordered sequence).
type metaTuple struct {
	ref       Ref
	key       orderedKey
	numLeaves uint64
	child     Collection // may be nil
}

func (mt metaTuple) getChildSequence(vr ValueReader) sequence {
	if mt.child != nil {
		return mt.child.sequence()
	}
	return mt.ref.TargetValue(vr).(Collection).sequence()
}

// orderedKey is a key in a Prolly Tree level, which is a metaTuple in a metaSequence, or a value in a leaf sequence.
// |v| may be nil or |h| may be empty, but not both.
type orderedKey struct {
	isOrderedByValue bool
	v                Value
	h                hash.Hash
}

func newOrderedKey(v Value) orderedKey {
	if isKindOrderedByValue(v.Kind()) {
		return orderedKey{true, v, hash.Hash{}}
	}
	return orderedKey{false, v, v.Hash()}
}

func orderedKeyFromHash(h hash.Hash) orderedKey {
	return orderedKey{false, nil, h}
}

func orderedKeyFromInt(n int) orderedKey {
	return newOrderedKey(Number(n))
}

func orderedKeyFromUint64(n uint64) orderedKey {
	return newOrderedKey(Number(n))
}

func (key orderedKey) Less(mk2 orderedKey) bool {
	switch {
	case key.isOrderedByValue && mk2.isOrderedByValue:
		return key.v.Less(mk2.v)
	case key.isOrderedByValue:
		return true
	case mk2.isOrderedByValue:
		return false
	default:
		d.PanicIfTrue(key.h.IsEmpty() || mk2.h.IsEmpty())
		return key.h.Less(mk2.h)
	}
}

type metaSequence struct {
	kind   NomsKind
	level  uint64
	tuples []metaTuple
	vr     ValueReader
}

func newMetaSequence(kind NomsKind, level uint64, tuples []metaTuple, vr ValueReader) metaSequence {
	d.PanicIfFalse(level > 0)
	return metaSequence{kind, level, tuples, vr}
}

func (ms metaSequence) data() []metaTuple {
	return ms.tuples
}

func (ms metaSequence) getKey(idx int) orderedKey {
	return ms.tuples[idx].key
}

func (ms metaSequence) cumulativeNumberOfLeaves(idx int) uint64 {
	cum := uint64(0)
	for i := 0; i <= idx; i++ {
		cum += ms.tuples[i].numLeaves
	}
	return cum
}

func (ms metaSequence) getCompareFn(other sequence) compareFn {
	oms := other.(metaSequence)
	return func(idx, otherIdx int) bool {
		return ms.tuples[idx].ref.TargetHash() == oms.tuples[otherIdx].ref.TargetHash()
	}
}

// sequence interface
func (ms metaSequence) getItem(idx int) sequenceItem {
	return ms.tuples[idx]
}

func (ms metaSequence) seqLen() int {
	return len(ms.tuples)
}

func (ms metaSequence) valueReader() ValueReader {
	return ms.vr
}

func (ms metaSequence) WalkRefs(cb RefCallback) {
	for _, tuple := range ms.tuples {
		cb(tuple.ref)
	}
}

func (ms metaSequence) typeOf() *Type {
	ts := make(typeSlice, len(ms.tuples))
	for i, mt := range ms.tuples {
		ts[i] = mt.ref.TargetType()
	}
	return makeCompoundType(UnionKind, ts...)
}

func (ms metaSequence) Kind() NomsKind {
	return ms.kind
}

func (ms metaSequence) numLeaves() uint64 {
	return ms.cumulativeNumberOfLeaves(len(ms.tuples) - 1)
}

func (ms metaSequence) treeLevel() uint64 {
	return ms.level
}

func (ms metaSequence) isLeaf() bool {
	d.PanicIfTrue(ms.level == 0)
	return false
}

// metaSequence interface
func (ms metaSequence) getChildSequence(idx int) sequence {
	mt := ms.tuples[idx]
	return mt.getChildSequence(ms.vr)
}

// Returns the sequences pointed to by all items[i], s.t. start <= i < end, and returns the
// concatentation as one long composite sequence
func (ms metaSequence) getCompositeChildSequence(start uint64, length uint64) sequence {
	d.PanicIfFalse(ms.level > 0)
	if length == 0 {
		return emptySequence{ms.level - 1}
	}

	metaItems := []metaTuple{}
	mapItems := []mapEntry{}
	valueItems := []Value{}

	childIsMeta := false
	isIndexedSequence := false
	if ListKind == ms.Kind() {
		isIndexedSequence = true
	}

	output := ms.getChildren(start, start+length)
	for _, seq := range output {

		switch t := seq.(type) {
		case metaSequence:
			childIsMeta = true
			metaItems = append(metaItems, t.tuples...)
		case mapLeafSequence:
			mapItems = append(mapItems, t.data...)
		case setLeafSequence:
			valueItems = append(valueItems, t.data...)
		case listLeafSequence:
			valueItems = append(valueItems, t.values...)
		default:
			panic("unreachable")
		}
	}

	if childIsMeta {
		return newMetaSequence(ms.kind, ms.level-1, metaItems, ms.vr)
	}

	if isIndexedSequence {
		return newListLeafSequence(ms.vr, valueItems...)
	}

	if MapKind == ms.Kind() {
		return newMapLeafSequence(ms.vr, mapItems...)
	}

	return newSetLeafSequence(ms.vr, valueItems...)
}

// fetches child sequences from start (inclusive) to end (exclusive) and respects uncommitted child
// sequences.
func (ms metaSequence) getChildren(start, end uint64) (seqs []sequence) {
	d.Chk.True(end <= uint64(len(ms.tuples)))
	d.Chk.True(start <= end)

	seqs = make([]sequence, end-start)
	hs := make(hash.HashSet, len(seqs))

	for i := start; i < end; i++ {
		mt := ms.tuples[i]
		if mt.child != nil {
			seqs[i-start] = mt.child.sequence()
		} else {
			hs[mt.ref.TargetHash()] = struct{}{}
		}
	}

	if len(hs) == 0 {
		return // can occur with ptree that is fully uncommitted
	}

	// Fetch committed child sequences in a single batch
	valueChan := make(chan Value, len(hs))
	go func() {
		ms.vr.ReadManyValues(hs, valueChan)
		close(valueChan)
	}()
	children := make(map[hash.Hash]sequence, len(hs))
	for value := range valueChan {
		children[value.Hash()] = value.(Collection).sequence()
	}

	for i := start; i < end; i++ {
		mt := ms.tuples[i]
		if mt.child != nil {
			continue
		}

		childSeq := children[mt.ref.TargetHash()]
		d.Chk.NotNil(childSeq)
		seqs[i-start] = childSeq
	}

	return
}

func readMetaTupleValue(item sequenceItem, vr ValueReader) Value {
	mt := item.(metaTuple)
	if mt.child != nil {
		return mt.child
	}

	r := mt.ref.TargetHash()
	d.PanicIfTrue(r.IsEmpty())
	return vr.ReadValue(r)
}

func metaHashValueBytes(item sequenceItem, rv *rollingValueHasher) {
	mt := item.(metaTuple)
	v := mt.key.v
	if !mt.key.isOrderedByValue {
		// See https://github.com/attic-labs/noms/issues/1688#issuecomment-227528987
		d.PanicIfTrue(mt.key.h.IsEmpty())
		v = constructRef(mt.key.h, BoolType, 0)
	}

	hashValueBytes(mt.ref, rv)
	hashValueBytes(v, rv)
}

type emptySequence struct {
	level uint64
}

func (es emptySequence) getItem(idx int) sequenceItem {
	panic("empty sequence")
}

func (es emptySequence) seqLen() int {
	return 0
}

func (es emptySequence) numLeaves() uint64 {
	return 0
}

func (es emptySequence) valueReader() ValueReader {
	return nil
}

func (es emptySequence) WalkRefs(cb RefCallback) {
}

func (es emptySequence) getCompareFn(other sequence) compareFn {
	return func(idx, otherIdx int) bool { panic("empty sequence") }
}

func (es emptySequence) getKey(idx int) orderedKey {
	panic("empty sequence")
}

func (es emptySequence) getValue(idx int) Value {
	panic("empty sequence")
}

func (es emptySequence) cumulativeNumberOfLeaves(idx int) uint64 {
	panic("empty sequence")
}

func (es emptySequence) getChildSequence(i int) sequence {
	return nil
}

func (es emptySequence) Kind() NomsKind {
	panic("empty sequence")
}

func (es emptySequence) typeOf() *Type {
	panic("empty sequence")
}

func (es emptySequence) getCompositeChildSequence(start uint64, length uint64) sequence {
	d.PanicIfFalse(es.level > 0)
	d.PanicIfFalse(start == 0)
	d.PanicIfFalse(length == 0)
	return emptySequence{es.level - 1}
}

func (es emptySequence) treeLevel() uint64 {
	return es.level
}

func (es emptySequence) isLeaf() bool {
	return es.level == 0
}

// Invokes |cb| on all "uncommitted" ptree nodes reachable from |v| in
// "bottom-up" order. It will only follow refs in the case of an in-memory
// reference to an uncommitted child. Note that |v| maybe not necessarily be
// a collection, but a collection with uncommitted children maybe be reachable
// from it (e.g. a Struct).
func iterateUncommittedChildren(v Value, cb func(sv Value)) {
	if _, ok := v.(*Type); ok {
		// Types can be in-memory cyclic, but they cannot reference uncommitted
		// nodes. Just avoid traversing into them.
		return
	}

	if c, ok := v.(Collection); ok {
		if ms, ok := c.sequence().(metaSequence); ok {
			// Only traverse uncommitted children of meta sequences
			for _, mt := range ms.tuples {
				if mt.child != nil {
					iterateUncommittedChildren(mt.child, cb)
					cb(mt.child)
				}
			}
			return
		}
	}

	// Traverse subvalues in search of nested collections with uncommitted
	// ptree nodes.
	v.WalkValues(func(v Value) {
		iterateUncommittedChildren(v, cb)
	})
}
