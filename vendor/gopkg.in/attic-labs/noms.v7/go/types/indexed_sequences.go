// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/hash"
)

func newListMetaSequence(level uint64, tuples []metaTuple, vr ValueReader) metaSequence {
	return newMetaSequence(ListKind, level, tuples, vr)
}

func newBlobMetaSequence(level uint64, tuples []metaTuple, vr ValueReader) metaSequence {
	return newMetaSequence(BlobKind, level, tuples, vr)
}

// advanceCursorToOffset advances the cursor as close as possible to idx
//
// If the cursor references a leaf sequence,
// 	advance to idx,
// 	and return the number of values preceding the idx
// If it references a meta-sequence,
// 	advance to the tuple containing idx,
// 	and return the number of leaf values preceding this tuple
func advanceCursorToOffset(cur *sequenceCursor, idx uint64) uint64 {
	seq := cur.seq

	if ms, ok := seq.(metaSequence); ok {
		// For a meta sequence, advance the cursor to the smallest position where idx < seq.cumulativeNumLeaves()
		cur.idx = 0
		cum := uint64(0)

		// Advance the cursor to the meta-sequence tuple containing idx
		for cur.idx < ms.seqLen()-1 && uint64(idx) >= cum+ms.tuples[cur.idx].numLeaves {
			cum += ms.tuples[cur.idx].numLeaves
			cur.idx++
		}

		return cum // number of leaves sequences BEFORE cur.idx in meta sequence
	}

	cur.idx = int(idx)
	if cur.idx > seq.seqLen() {
		cur.idx = seq.seqLen()
	}
	return uint64(cur.idx)
}

// If |sink| is not nil, chunks will be eagerly written as they're created. Otherwise they are
// written when the root is written.
func newIndexedMetaSequenceChunkFn(kind NomsKind, source ValueReader) makeChunkFn {
	return func(level uint64, items []sequenceItem) (Collection, orderedKey, uint64) {
		tuples := make([]metaTuple, len(items))
		numLeaves := uint64(0)

		for i, v := range items {
			mt := v.(metaTuple)
			tuples[i] = mt
			numLeaves += mt.numLeaves
		}

		var col Collection
		if kind == ListKind {
			col = newList(newListMetaSequence(level, tuples, source))
		} else {
			d.PanicIfFalse(BlobKind == kind)
			col = newBlob(newBlobMetaSequence(level, tuples, source))
		}
		return col, orderedKeyFromSum(tuples), numLeaves
	}
}

func orderedKeyFromSum(msd []metaTuple) orderedKey {
	sum := uint64(0)
	for _, mt := range msd {
		sum += mt.numLeaves
	}
	return orderedKeyFromUint64(sum)
}

// loads the set of leaf sequences which contain the items [startIdx -> endIdx).
// Returns the set of sequences and the offset within the first sequence which corresponds to |startIdx|.
func loadLeafSequences(vr ValueReader, seqs []sequence, startIdx, endIdx uint64) ([]sequence, uint64) {
	if seqs[0].isLeaf() {
		for _, s := range seqs {
			d.PanicIfFalse(s.isLeaf())
		}

		return seqs, startIdx
	}

	level := seqs[0].treeLevel()
	childTuples := []metaTuple{}

	cum := uint64(0)
	for _, s := range seqs {
		d.PanicIfFalse(s.treeLevel() == level)
		ms := s.(metaSequence)

		for _, mt := range ms.tuples {
			if cum == 0 && mt.numLeaves <= startIdx {
				// skip tuples whose items are < startIdx
				startIdx -= mt.numLeaves
				endIdx -= mt.numLeaves
				continue
			}

			childTuples = append(childTuples, mt)
			cum += mt.numLeaves
			if cum >= endIdx {
				break
			}
		}
	}

	hs := hash.HashSet{}
	for _, mt := range childTuples {
		if mt.child != nil {
			continue
		}
		hs.Insert(mt.ref.TargetHash())
	}

	// Fetch committed child sequences in a single batch
	fetched := make(map[hash.Hash]sequence, len(hs))
	if len(hs) > 0 {
		valueChan := make(chan Value, len(hs))
		go func() {
			d.PanicIfTrue(vr == nil)
			vr.ReadManyValues(hs, valueChan)
			close(valueChan)
		}()
		for value := range valueChan {
			fetched[value.Hash()] = value.(Collection).sequence()
		}
	}

	childSeqs := make([]sequence, len(childTuples))
	for i, mt := range childTuples {
		if mt.child != nil {
			childSeqs[i] = mt.child.sequence()
			continue
		}

		childSeqs[i] = fetched[mt.ref.TargetHash()]
	}

	return loadLeafSequences(vr, childSeqs, startIdx, endIdx)
}
