// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type newSequenceChunkerFn func(cur *sequenceCursor) *sequenceChunker

func concat(fst, snd sequence, newSequenceChunker newSequenceChunkerFn) sequence {
	// To understand how concat (along with appendCursorToChunker) works, imagine
	// *incorrectly* implementing concat using the simplest possible method:
	// simply concatenate the sequences at each level.
	//
	//    [FF]
	//  [AA] [BB]
	//
	//  +
	//
	//    [SS]
	//  [CC] [DD]
	//
	//  =
	//
	//    [FF]       [SS]
	//  [AA] [BB] [CC] [DD]
	//
	// which requires adding another level to the prolly tree to reference the
	// roots of each tree:
	//
	//         [RR]
	//    [FF]       [SS]
	//  [AA] [BB] [CC] [DD]
	//
	// This is probably incorrect for 2 reasons:
	//
	//  1. It assumes that [BB] and [FF] each end on a chunk boundary, which has a
	//     small chance of being true, specificaly (1 / target chunk size).
	//  2. It assumes that the rolling hash window doesn't change the chunk
	//     boundaries. In snd, the chunk boundary after [CC] is calculated with a
	//     rolling hash primed with no values. In the new tree, it would be primed
	//     with AABB.
	//
	// concat/appendCursorToChunker fixes this by starting the chunking at the
	// beginning of the rightmost chunk of fst, then continuing through snd (a)
	// beyond the chunk window, to solve Incorrect Reason 2, then (b) through to
	// the next definite chunk boundary of snd, to solve Reason 1.
	//
	// Implementation-wise wise we can simply reuse sequenceChunker re-chunking
	// logic, which automatically steps back to the start of the previous chunk
	// (and can correctly prime the chunk window, etc), plus how to construct that
	// resulting concatenated sequence.
	chunker := newSequenceChunker(newCursorAtIndex(fst, fst.numLeaves()))
	appendCursorToChunker(chunker, newCursorAtIndex(snd, 0))
	return chunker.Done()
}

func appendCursorToChunker(chunker *sequenceChunker, cur *sequenceCursor) {
	// Append beyond the chunk window, then through to the next definite chunk
	// boundary.
	hashWindow := chunkWindow
	for cur.valid() && (hashWindow > 0 || cur.indexInChunk() != 0) {
		hashWindow -= chunker.Append(cur.current())
		cur.advance()
	}

	// There may not be a chunk boundary beyond the chunk window - terminate.
	if !cur.valid() {
		return
	}

	d.PanicIfTrue(len(chunker.current) > 0, "there were unchunked items")

	// sequenceChunker's implementation will create parent chunkers while the
	// cursor has a parent, and only trim then when Done is called.
	d.PanicIfTrue(cur.parent == nil, "parent is nil")
	d.PanicIfTrue(chunker.parent == nil, "chunker.parent is nil")

	appendCursorToChunker(chunker.parent, cur.parent)
}
