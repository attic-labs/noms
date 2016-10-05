// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

type sequenceItem interface{}

type compareFn func(x int, y int) bool

type sequence interface {
	getItem(idx int) sequenceItem
	seqLen() int
	numLeaves() uint64
	valueReader() ValueReader
	WalkRefs(cb RefCallback)
	Type() *Type
	getCompareFn(other sequence) compareFn
	getChildSequence(idx int) sequence
}
