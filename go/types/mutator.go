// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type MapMutator struct {
	oc *opCache
}

func newMutator(vrw ValueReadWriter) *MapMutator {
	return &MapMutator{newOpCache(vrw)}
}

func (mx *MapMutator) Set(key Value, val Value) *MapMutator {
	d.Chk.True(mx.oc != nil, "Can't call Set() again after Finish()")
	mx.oc.Set(key, val)
	return mx
}

func (mx *MapMutator) Finish() Map {
	d.Chk.True(mx.oc != nil, "Can only call Finish() once")
	defer func() {
		mx.oc.Destroy()
		mx.oc = nil
	}()

	seq := newEmptySequenceChunker(makeMapLeafChunkFn(nil), newOrderedMetaSequenceChunkFn(MapKind, nil), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	iter := mx.oc.NewIterator()
	defer iter.Release()
	for iter.Next() {
		seq.Append(iter.Op())
	}
	return newMap(seq.Done().(orderedSequence))
}
