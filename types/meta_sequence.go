// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/d"

const (
	objectWindowSize          = 8
	orderedSequenceWindowSize = 1
	objectPattern             = uint32(1<<6 - 1) // Average size of 64 elements
)

// metaSequence is a logical abstraction, but has no concrete "base" implementation. A Meta Sequence is a non-leaf (internal) node of a "probably" tree, which results from the chunking of an ordered or unordered sequence of values.
type metaSequence interface {
	sequence
	getChildSequence(idx int) sequence
}

func newMetaTuple(ref Ref, value Value, numLeaves uint64, child Collection) metaTuple {
	d.Chk.NotEqual(Ref{}, ref)
	return metaTuple{ref, value, numLeaves, child}
}

// metaTuple is a node in a Prolly Tree, consisting of data in the node (either tree leaves or other metaSequences), and a Value annotation for exploring the tree (e.g. the largest item if this an ordered sequence).
type metaTuple struct {
	ref       Ref
	value     Value
	numLeaves uint64
	child     Collection // may be nil
}

func (mt metaTuple) uint64Value() uint64 {
	return uint64(mt.value.(Number))
}

type metaSequenceData []metaTuple

func (msd metaSequenceData) uint64ValuesSum() (sum uint64) {
	for _, mt := range msd {
		sum += mt.uint64Value()
	}
	return
}

func (msd metaSequenceData) numLeavesSum() (sum uint64) {
	for _, mt := range msd {
		sum += mt.numLeaves
	}
	return
}

func (msd metaSequenceData) last() metaTuple {
	return msd[len(msd)-1]
}

type metaSequenceObject struct {
	tuples metaSequenceData
	t      *Type
	vr     ValueReader
}

func (ms metaSequenceObject) data() metaSequenceData {
	return ms.tuples
}

// sequence interface
func (ms metaSequenceObject) getItem(idx int) sequenceItem {
	return ms.tuples[idx]
}

func (ms metaSequenceObject) seqLen() int {
	return len(ms.tuples)
}

func (ms metaSequenceObject) valueReader() ValueReader {
	return ms.vr
}

func (ms metaSequenceObject) Chunks() []Ref {
	chunks := make([]Ref, len(ms.tuples))
	for i, tuple := range ms.tuples {
		chunks[i] = tuple.ref
	}
	return chunks
}

func (ms metaSequenceObject) Type() *Type {
	return ms.t
}

// metaSequence interface
func (ms metaSequenceObject) getChildSequence(idx int) sequence {
	mt := ms.tuples[idx]
	if mt.child != nil {
		return mt.child.sequence()
	}

	return mt.ref.TargetValue(ms.vr).(Collection).sequence()
}

// Returns the sequences pointed to by all items[i], s.t. start <= i < end, and returns the
// concatentation as one long composite sequence
func (ms metaSequenceObject) getCompositeChildSequence(start uint64, length uint64) indexedSequence {
	childIsMeta := false
	metaItems := []metaTuple{}
	valueItems := []Value{}
	for i := start; i < start+length; i++ {
		seq := ms.getChildSequence(int(i))
		if i == start {
			if idxSeq, ok := seq.(indexedSequence); ok {
				childIsMeta = isMetaSequence(idxSeq)
			}
		}
		if childIsMeta {
			childMs, _ := seq.(indexedMetaSequence)
			metaItems = append(metaItems, childMs.metaSequenceObject.tuples...)
		} else {
			if ll, ok := seq.(listLeafSequence); ok {
				valueItems = append(valueItems, ll.values...)
			}
		}
	}

	if childIsMeta {
		return newIndexedMetaSequence(metaItems, ms.Type(), ms.vr)
	} else {
		return newListLeafSequence(ms.vr, valueItems...)
	}
}

func isMetaSequence(seq sequence) bool {
	_, seqIsMeta := seq.(metaSequence)
	return seqIsMeta
}

// Creates a sequenceCursor pointing to the first metaTuple in a metaSequence, and returns that cursor plus the leaf Value referenced from that metaTuple.
func newMetaSequenceCursor(root metaSequence, vr ValueReader) (*sequenceCursor, Value) {
	d.Chk.NotNil(root)

	cursors := []*sequenceCursor{newSequenceCursor(nil, root, 0)}
	for {
		cursor := cursors[len(cursors)-1]
		val := readMetaTupleValue(cursor.current(), vr)
		if ms, ok := val.(metaSequence); ok {
			cursors = append(cursors, newSequenceCursor(cursor, ms, 0))
		} else {
			return cursor, val
		}
	}
}

func readMetaTupleValue(item sequenceItem, vr ValueReader) Value {
	mt := item.(metaTuple)
	if mt.child != nil {
		return mt.child
	}

	r := mt.ref.TargetHash()
	d.Chk.False(r.IsEmpty())
	return vr.ReadValue(r)
}

func iterateMetaSequenceLeaf(ms metaSequence, vr ValueReader, cb func(Value) bool) {
	cursor, v := newMetaSequenceCursor(ms, vr)
	for {
		if cb(v) || !cursor.advance() {
			return
		}

		v = readMetaTupleValue(cursor.current(), vr)
	}
}
