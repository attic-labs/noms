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
	Chunks() []Ref
	Type() *Type
	getCompareFn(other sequence) compareFn
	cumulativeNumberOfLeaves(idx int) uint64 // returns the total number of leaf values reachable from this sequence for all sub-trees from 0 to |idx|
}
