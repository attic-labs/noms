// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

type orderedSequence interface {
	sequence
	getKey(idx int) orderedKey
	getValue(idx int) Value
}

func newSetMetaSequence(tuples []metaTuple, vr ValueReader) metaSequence {
	return newMetaSequence(tuples, SetKind, vr)
}

func newMapMetaSequence(tuples []metaTuple, vr ValueReader) metaSequence {
	return newMetaSequence(tuples, MapKind, vr)
}

func newCursorAtValue(seq orderedSequence, val Value, forInsertion bool, last bool, readAhead bool) *sequenceCursor {
	var key orderedKey
	if val != nil {
		key = newOrderedKey(val)
	}
	return newCursorAt(seq, key, forInsertion, last, readAhead)
}

func newCursorAt(seq orderedSequence, key orderedKey, forInsertion bool, last bool, readAhead bool) *sequenceCursor {
	var cur *sequenceCursor
	for {
		idx := 0
		if last {
			idx = -1
		}
		cur = newSequenceCursor(cur, seq, idx, readAhead)
		if key != emptyKey {
			if !seekTo(cur, key, forInsertion && !seq.isLeaf()) {
				return cur
			}
		}

		cs := cur.getChildSequence()
		if cs == nil {
			break
		}
		seq = cs.(orderedSequence)
	}
	d.PanicIfFalse(cur != nil)
	return cur
}

func seekTo(cur *sequenceCursor, key orderedKey, lastPositionIfNotFound bool) bool {
	seq := cur.seq.(orderedSequence)

	// Find smallest idx in seq where key(idx) >= key
	cur.idx = sort.Search(seq.seqLen(), func(i int) bool {
		return !seq.getKey(i).Less(key)
	})

	if cur.idx == seq.seqLen() && lastPositionIfNotFound {
		d.PanicIfFalse(cur.idx > 0)
		cur.idx--
	}

	return cur.idx < seq.seqLen()
}

// Gets the key used for ordering the sequence at current index.
func getCurrentKey(cur *sequenceCursor) orderedKey {
	seq, ok := cur.seq.(orderedSequence)
	if !ok {
		d.Panic("need an ordered sequence here")
	}
	return seq.getKey(cur.idx)
}

func getCurrentValue(cur *sequenceCursor) Value {
	seq, ok := cur.seq.(orderedSequence)
	if !ok {
		d.Panic("need an ordered sequence here")
	}
	return seq.getValue(cur.idx)
}

// If |vw| is not nil, chunks will be eagerly written as they're created. Otherwise they are
// written when the root is written.
func newOrderedMetaSequenceChunkFn(kind NomsKind, vr ValueReader) makeChunkFn {
	return func(items []sequenceItem) (Collection, orderedKey, uint64) {
		tuples := make([]metaTuple, len(items))
		numLeaves := uint64(0)

		for i, v := range items {
			mt := v.(metaTuple)
			tuples[i] = mt // chunk is written when the root sequence is written
			numLeaves += mt.numLeaves
		}

		var col Collection
		if kind == SetKind {
			col = newSet(newSetMetaSequence(tuples, vr))
		} else {
			d.PanicIfFalse(MapKind == kind)
			col = newMap(newMapMetaSequence(tuples, vr))
		}

		return col, tuples[len(tuples)-1].key, numLeaves
	}
}
