// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type MapMutator struct {
	oc  *opCache
	vrw ValueReadWriter
}

func newMutator(vrw ValueReadWriter) *MapMutator {
	return &MapMutator{newOpCache(vrw), vrw}
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

	seq := newEmptySequenceChunker(makeMapLeafChunkFn(mx.vrw, mx.vrw), newOrderedMetaSequenceChunkFn(MapKind, mx.vrw, mx.vrw), newMapLeafBoundaryChecker(), newOrderedMetaSequenceBoundaryChecker)

	// I tried splitting this up so that the iteration ran in a separate goroutine from the Append'ing, but it actually made things a bit slower when I ran a test.
	iter := mx.oc.NewIterator()
	defer iter.Release()
	for iter.Next() {
		seq.Append(iter.Op())
	}
	return newMap(seq.Done().(orderedSequence))
}
