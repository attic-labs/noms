// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type boundaryChecker interface {
	// Write takes an item and returns true if the sequence should chunk after this item, false if not.
	Write(sequenceItem) bool
	// WindowSize returns the minimum number of items in a stream that must be written before resuming a chunking sequence.
	WindowSize() int
}

type newBoundaryCheckerFn func() boundaryChecker

type sequenceChunker struct {
	cur                        *sequenceCursor
	vw                         ValueWriter
	parent                     *sequenceChunker
	current                    []sequenceItem
	makeChunk, parentMakeChunk makeChunkFn
	boundaryChk                boundaryChecker
	newBoundaryChecker         newBoundaryCheckerFn
	isLeaf                     bool
	done                       bool
}

// makeChunkFn takes a sequence of items to chunk, and returns the result of chunking those items, a tuple of a reference to that chunk which can itself be chunked + its underlying value.
type makeChunkFn func(values []sequenceItem) (Collection, orderedKey, uint64)

func newEmptySequenceChunker(vw ValueWriter, makeChunk, parentMakeChunk makeChunkFn, boundaryChk boundaryChecker, newBoundaryChecker newBoundaryCheckerFn) *sequenceChunker {
	return newSequenceChunker(nil, vw, makeChunk, parentMakeChunk, boundaryChk, newBoundaryChecker)
}

func newSequenceChunker(cur *sequenceCursor, vw ValueWriter, makeChunk, parentMakeChunk makeChunkFn, boundaryChk boundaryChecker, newBoundaryChecker newBoundaryCheckerFn) *sequenceChunker {
	// |cur| will be nil if this is a new sequence, implying this is a new tree, or the tree has grown in height relative to its original chunked form.
	d.Chk.True(makeChunk != nil)
	d.Chk.True(parentMakeChunk != nil)
	d.Chk.True(boundaryChk != nil)
	d.Chk.True(newBoundaryChecker != nil)

	sc := &sequenceChunker{
		cur,
		vw,
		nil,
		[]sequenceItem{},
		makeChunk, parentMakeChunk,
		boundaryChk,
		newBoundaryChecker,
		true,
		false,
	}

	if cur != nil {
		sc.resume()
	}

	return sc
}

func (sc *sequenceChunker) resume() {
	if sc.cur.parent != nil {
		sc.createParent()
	}

	// Number of previous items which must be hashed into the boundary checker.
	primeHashWindow := sc.boundaryChk.WindowSize() - 1

	retreater := sc.cur.clone()
	appendCount := 0
	primeHashCount := 0

	// If the cursor is beyond the final position in the sequence, the preceeding item may have been a chunk boundary. In that case, we must test at least the preceeding item.
	appendPenultimate := sc.cur.idx == sc.cur.length()
	if appendPenultimate && retreater.retreatMaybeAllowBeforeStart(false) {
		// In that case, we prime enough items *prior* to the penultimate item to be correct.
		appendCount++
		primeHashCount++
	}

	// Walk backwards to the start of the existing chunk
	for retreater.indexInChunk() > 0 && retreater.retreatMaybeAllowBeforeStart(false) {
		appendCount++
		if primeHashWindow > 0 {
			primeHashCount++
			primeHashWindow--
		}
	}

	// If the hash window won't be filled by the preceeding items in the current chunk, walk further back until they will.
	for primeHashWindow > 0 && retreater.retreatMaybeAllowBeforeStart(false) {
		primeHashCount++
		primeHashWindow--
	}

	for primeHashCount > 0 || appendCount > 0 {
		item := retreater.current()
		if primeHashCount > appendCount {
			// Before the start of the current chunk: just hash value bytes into window
			sc.boundaryChk.Write(item)
			primeHashCount--
		} else if appendCount > primeHashCount {
			// In current chunk, but before window: just append item
			sc.current = append(sc.current, item)
			appendCount--
		} else {
			// Within current chunk and hash window: append item & hash value bytes into window.
			if appendPenultimate && appendCount == 1 {
				// It's ONLY correct Append immediately preceeding the cursor position because only after its insertion into the hash will the window be filled.
				sc.Append(item)
			} else {
				sc.boundaryChk.Write(item)
				sc.current = append(sc.current, item)
			}
			appendCount--
			primeHashCount--
		}

		retreater.advance()
	}
}

func (sc *sequenceChunker) Append(item sequenceItem) {
	d.Chk.True(item != nil)
	sc.current = append(sc.current, item)
	if sc.boundaryChk.Write(item) {
		sc.handleChunkBoundary()
	}
}

func (sc *sequenceChunker) Skip() {
	if sc.cur.advance() && sc.cur.indexInChunk() == 0 {
		// Advancing moved our cursor into the next chunk. We need to advance our parent's cursor, so that when our parent writes out the remaining chunks it doesn't include the chunk that we skipped.
		sc.skipParentIfExists()
	}
}

func (sc *sequenceChunker) skipParentIfExists() {
	if sc.parent != nil && sc.parent.cur != nil {
		sc.parent.Skip()
	}
}

func (sc *sequenceChunker) createParent() {
	d.Chk.True(sc.parent == nil)
	var parent *sequenceCursor
	if sc.cur != nil && sc.cur.parent != nil {
		// Clone the parent cursor because otherwise calling cur.advance() will affect our parent - and vice versa - in surprising ways. Instead, Skip moves forward our parent's cursor if we advance across a boundary.
		parent = sc.cur.parent.clone()
	}
	sc.parent = newSequenceChunker(parent, sc.vw, sc.parentMakeChunk, sc.parentMakeChunk, sc.newBoundaryChecker(), sc.newBoundaryChecker)
	sc.parent.isLeaf = false
}

func (sc *sequenceChunker) createSequence() (sequence, metaTuple) {
	// If the sequence chunker has a ValueWriter, eagerly write sequences
	col, key, numLeaves := sc.makeChunk(sc.current)
	seq := col.sequence()
	var ref Ref
	if sc.vw != nil {
		ref = sc.vw.WriteValue(col)
		col = nil
	} else {
		ref = NewRef(col)
	}
	mt := newMetaTuple(ref, key, numLeaves, col)

	sc.current = []sequenceItem{}
	return seq, mt
}

func (sc *sequenceChunker) handleChunkBoundary() {
	d.Chk.NotEmpty(sc.current)

	_, mt := sc.createSequence()
	if sc.parent == nil {
		sc.createParent()
	}
	sc.parent.Append(mt)
}

// Returns true if this chunker of any of its parents have any pending items in their |current| slice.
func (sc *sequenceChunker) anyPending() bool {
	if len(sc.current) > 0 {
		return true
	}

	if sc.parent != nil {
		return sc.parent.anyPending()
	}

	return false
}

// Returns the root sequence of the resulting tree.
func (sc *sequenceChunker) Done(vr ValueReader) sequence {
	d.Chk.True((vr == nil) == (sc.vw == nil))
	d.Chk.False(sc.done)
	sc.done = true

	if sc.cur != nil {
		sc.finalizeCursor()
	}

	if sc.parent == nil || !sc.parent.anyPending() {
		if sc.isLeaf {
			// Return the (possibly empty) sequence which never chunked
			seq, _ := sc.createSequence()
			return seq
		}

		if len(sc.current) == 1 {
			// Walk down until we find either a leaf sequence or meta sequence with more than one metaTuple.
			seq := sc.current[0].(metaTuple).getChildSequence(vr)

			for {
				if ms, ok := seq.(metaSequence); ok && seq.seqLen() == 1 {
					seq = ms.getChildSequence(0)
					continue
				}

				return seq
			}
			panic("not reached")
		}
	}

	if len(sc.current) > 0 {
		sc.handleChunkBoundary()
	}

	return sc.parent.Done(vr)
}

func (sc *sequenceChunker) finalizeCursor() {
	if !sc.cur.valid() {
		// The cursor is past the end, and due to the way cursors work, the parent cursor will actually point to its last chunk. We need to force it to point past the end so that our parent's Done() method doesn't add the last chunk twice.
		sc.skipParentIfExists()
		return
	}

	// Append the rest of the values in the sequence, up to the window size, plus the rest of that chunk. It needs to be the full window size because anything that was appended/skipped between chunker construction and finalization will have changed the hash state.
	hashWindow := sc.boundaryChk.WindowSize()
	fzr := sc.cur.clone()

	for i := 0; hashWindow > 0 || fzr.indexInChunk() > 0; i++ {
		if i == 0 || fzr.indexInChunk() == 0 {
			// Every time we step into a chunk from the original sequence, that chunk will no longer exist in the new sequence. The parent must be instructed to skip it.
			sc.skipParentIfExists()
		}

		item := fzr.current()
		didAdvance := fzr.advance()

		if hashWindow > 0 {
			// While we are within the hash window, append items (which explicit checks the hash value for chunk boundaries)
			sc.Append(item)
			hashWindow--
		} else {
			// Once we are beyond the hash window, we know that boundaries can only occur in the same place they did within the existing sequence
			sc.current = append(sc.current, item)
			if didAdvance && fzr.indexInChunk() == 0 {
				sc.handleChunkBoundary()
			}
		}
		if !didAdvance {
			break
		}
	}
}
