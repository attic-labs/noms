// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type listLeafSequence struct {
	leafSequence
	values ValueSlice
}

func newListLeafSequence(vrw ValueReadWriter, v ...Value) sequence {
	d.PanicIfTrue(vrw == nil)
	return listLeafSequence{leafSequence{vrw, len(v), ListKind}, v}
}

// sequence interface

func (ll listLeafSequence) getCompareFn(other sequence) compareFn {
	oll := other.(listLeafSequence)
	return func(idx, otherIdx int) bool {
		return ll.values[idx].Equals(oll.values[otherIdx])
	}
}

func (ll listLeafSequence) getItem(idx int) sequenceItem {
	return ll.values[idx]
}

func (ll listLeafSequence) WalkRefs(cb RefCallback) {
	for _, v := range ll.values {
		v.WalkRefs(cb)
	}
}

func (ll listLeafSequence) typeOf() *Type {
	ts := make([]*Type, len(ll.values))
	for i, v := range ll.values {
		ts[i] = v.typeOf()
	}
	return makeCompoundType(ListKind, makeCompoundType(UnionKind, ts...))
}
