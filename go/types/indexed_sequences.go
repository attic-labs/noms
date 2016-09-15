// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type indexedSequence interface {
	sequence
}

type indexedMetaSequence struct {
	metaSequenceObject
}

func newListMetaSequence(tuples metaSequenceData, vr ValueReader) indexedMetaSequence {
	ts := make([]*Type, len(tuples))
	for i, mt := range tuples {
		// Ref<List<T>>
		ts[i] = mt.ref.Type().Desc.(CompoundDesc).ElemTypes[0].Desc.(CompoundDesc).ElemTypes[0]
	}
	t := MakeListType(MakeUnionType(ts...))
	return newIndexedMetaSequence(tuples, t, vr)
}

func newBlobMetaSequence(tuples metaSequenceData, vr ValueReader) indexedMetaSequence {
	return newIndexedMetaSequence(tuples, BlobType, vr)
}

func newIndexedMetaSequence(tuples metaSequenceData, t *Type, vr ValueReader) indexedMetaSequence {
	return indexedMetaSequence{newMetaSequenceObject(tuples, t, vr)}
}

func (ims indexedMetaSequence) getCompareFn(other sequence) compareFn {
	oms := other.(indexedMetaSequence)
	return func(idx, otherIdx int) bool {
		return ims.tuples[idx].ref.TargetHash() == oms.tuples[otherIdx].ref.TargetHash()
	}
}

// If |sink| is not nil, chunks will be eagerly written as they're created. Otherwise they are
// written when the root is written.
func newIndexedMetaSequenceChunkFn(kind NomsKind, source ValueReader) makeChunkFn {
	return func(items []sequenceItem) (Collection, orderedKey, uint64) {
		tuples := make(metaSequenceData, len(items))
		numLeaves := uint64(0)

		for i, v := range items {
			mt := v.(metaTuple)
			tuples[i] = mt
			numLeaves += mt.numLeaves
		}

		var col Collection
		if kind == ListKind {
			col = newList(newListMetaSequence(tuples, source))
		} else {
			d.PanicIfFalse(BlobKind == kind)
			col = newBlob(newBlobMetaSequence(tuples, source))
		}
		return col, orderedKeyFromSum(tuples), numLeaves
	}
}

func orderedKeyFromSum(msd metaSequenceData) orderedKey {
	sum := uint64(0)
	for _, mt := range msd {
		sum += mt.key.uint64Value()
	}
	return orderedKeyFromUint64(sum)
}
