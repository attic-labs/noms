// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"math"
	"os"
	"runtime"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/orderedparallel"
)

// sequenceCursor explores a tree of sequence items.
type sequenceCursor struct {
	parent      *sequenceCursor
	seq         sequence
	idx         int
	readAhead   bool
	readAheadCh chan interface{}
}

// newSequenceCursor creates a cursor on seq positioned at idx.
// If idx < 0, count backward from the end of seq.
func newSequenceCursor(parent *sequenceCursor, seq sequence, idx int) *sequenceCursor {
	d.PanicIfTrue(seq == nil)
	if idx < 0 {
		idx += seq.seqLen()
		d.PanicIfFalse(idx >= 0)
	}

	cur := &sequenceCursor{parent, seq, idx, false, nil}
	return cur
}

// enableReadAhead turns on read-ahead.
//
// When enabled, the cursor will assumes it'll be moving forward and can
// benefit from optimistically reading leaves before they are requested.
//
// This is a hack to minimize API changes. A better approach would be to
// declare a cursor as "forward" at construction time, and only enable
// read-ahead for "forward" cursors.
func (cur *sequenceCursor) enableReadAhead() {
	// Environment variable is temporary to allow for before & after perf tests.
	cur.readAhead = os.Getenv("NOMS_NO_READAHEAD") == ""
}

func (cur *sequenceCursor) length() int {
	return cur.seq.seqLen()
}

func (cur *sequenceCursor) getItem(idx int) sequenceItem {
	return cur.seq.getItem(idx)
}

// sync advances the cursor to the next chunk
func (cur *sequenceCursor) sync() {
	d.PanicIfFalse(cur.parent != nil)
	cur.seq = cur.parent.getChildSequence()
	cur.readAheadCh = nil
}

// doReadAhead returns true if read-head should be used when reading leaves
// of this cursor.
//
// Limit read-ahead to meta-sequences that contain leaf sequences. The benefit
// of read-ahead diminishes quickly as distance from the leaves increases since
// chunking leads to >100 branching factor at each level.
func (cur *sequenceCursor) doReadAhead() bool {
	if cur.readAhead && cur.readAheadCh == nil {
		if ms, ok := cur.seq.(metaSequence); ok {
			return ms.tuples[0].ref.height == 1
		}
	}
	return false
}

// Records in read-ahead channel
type readAheadRec struct {
	idx     int
	leafSeq sequence
}

var readAheadParallelism = runtime.NumCPU() * 8

func minInt(x, y int) int {
	return int(math.Min(float64(x), float64(y)))
}

// readAheadLeaves initiates read-ahead of child sequences.
//
// Fills a buffered read-ahead channel with child sequences in ascending index order.
// A forward moving cursor reads this channel to find it's next item (see getChildSequence).
//
// The channel is limited to c * NumCPU. This allows read-ahead to proceed in parallel
// while maintaining a bound on memory.
func (cur *sequenceCursor) readAheadLeaves() {
	ms := cur.seq.(metaSequence)
	startIdx := cur.idx
	tuplesToRead := len(ms.tuples) - startIdx
	if tuplesToRead < 2 {
		// Don't read-ahead if we're starting at the end of a sequence. If the cursor's moving
		// backward, read-ahead is useless. If the cursor's moving forward, there's no
		// benefit.
		cur.readAheadCh = nil
	} else {
		parallelism := minInt(readAheadParallelism, tuplesToRead)
		input := make(chan interface{}, parallelism)

		cur.readAheadCh = orderedparallel.New(input, func(item interface{}) interface{} {
			i := item.(int)
			return &readAheadRec{i, ms.getChildSequence(i)}
		}, parallelism)

		go func() {
			for i := startIdx; i < len(ms.tuples); i++ {
				input <- i
			}
			close(input)
		}()
	}
}

const channelReadTimeout = 30 * time.Second

// getChildSequence retrieves the child at the current cursor position.
//
// If the the read-ahead channel is enabled, read the child from the channel.
func (cur *sequenceCursor) getChildSequence() sequence {
	if cur.doReadAhead() {
		cur.readAheadLeaves()
	}
	if cur.readAheadCh != nil {
	Loop:
		for {
			select {
			case item, more := <-cur.readAheadCh:
				if !more {
					// Channel closed; read directly. This can occur if the cursor
					// is retreating.
					break Loop
				}
				ra := item.(*readAheadRec)
				if cur.idx > ra.idx {
					// Keep looking
				} else if cur.idx == ra.idx {
					// Match
					return ra.leafSeq
				} else {
					// Leaf missing; read directly. This can occur if the cursor is
					// retreating.
					break Loop
				}
			case <-time.After(channelReadTimeout):
				d.PanicIfTrue(true, "Timed out waiting for next item in read-ahead channel: %s", cur)
			}
		}
	}
	return cur.seq.getChildSequence(cur.idx)
}

// current returns the value at the current cursor position
func (cur *sequenceCursor) current() sequenceItem {
	d.PanicIfFalse(cur.valid())
	return cur.getItem(cur.idx)
}

func (cur *sequenceCursor) valid() bool {
	return cur.idx >= 0 && cur.idx < cur.length()
}

func (cur *sequenceCursor) depth() int {
	if nil != cur.parent {
		return 1 + cur.parent.depth()
	}
	return 1
}

func (cur *sequenceCursor) indexInChunk() int {
	return cur.idx
}

func (cur *sequenceCursor) advance() bool {
	return cur.advanceMaybeAllowPastEnd(true)
}

func (cur *sequenceCursor) advanceMaybeAllowPastEnd(allowPastEnd bool) bool {
	if cur.idx < cur.length()-1 {
		cur.idx++
		return true
	}
	if cur.idx == cur.length() {
		return false
	}
	if cur.parent != nil && cur.parent.advanceMaybeAllowPastEnd(false) {
		// at end of current leaf chunk and there are more
		cur.sync()
		cur.idx = 0
		return true
	}
	if allowPastEnd {
		cur.idx++
	}
	return false
}

func (cur *sequenceCursor) retreat() bool {
	return cur.retreatMaybeAllowBeforeStart(true)
}

func (cur *sequenceCursor) retreatMaybeAllowBeforeStart(allowBeforeStart bool) bool {
	if cur.idx > 0 {
		cur.idx--
		return true
	}
	if cur.idx == -1 {
		return false
	}
	d.PanicIfFalse(0 == cur.idx)
	if cur.parent != nil && cur.parent.retreatMaybeAllowBeforeStart(false) {
		cur.sync()
		cur.idx = cur.length() - 1
		return true
	}
	if allowBeforeStart {
		cur.idx--
	}
	return false
}

// clone creates a copy of the cursor
//
// The clone does not inherit read-ahead behavior
func (cur *sequenceCursor) clone() *sequenceCursor {
	var parent *sequenceCursor
	if cur.parent != nil {
		parent = cur.parent.clone()
	}
	return newSequenceCursor(parent, cur.seq, cur.idx)
}

type cursorIterCallback func(item interface{}) bool

// iter iterates forward from the current position
func (cur *sequenceCursor) iter(cb cursorIterCallback) {
	cur.enableReadAhead() // ensure read-ahead is enabled to
	for cur.valid() && !cb(cur.getItem(cur.idx)) {
		cur.advance()
	}
}

// newCursorAtIndex creates a new cursor over seq positioned at idx.
//
// Implemented by searching down the tree to the leaf sequence containing idx. Each
// sequence cursor includes a back pointer to its parent so that it can follow the path
// to the next leaf chunk when the cursor exhausts the entries in the current chunk.
func newCursorAtIndex(seq sequence, idx uint64) *sequenceCursor {
	var cur *sequenceCursor
	for {
		cur = newSequenceCursor(cur, seq, 0)
		// Assume cursor starting at 0 will be moving forward and benefit from read-ahead
		// TODO: Inferring that a 0 idx means the cursor will be moving forward is
		// ugly. Another approach would be to provide a new method for creating a forward
		// cursor and always enable read-ahead in those cases.
		if idx == 0 {
			cur.enableReadAhead()
		}
		idx = idx - advanceCursorToOffset(cur, idx)
		cs := cur.getChildSequence()
		if cs == nil {
			break
		}
		seq = cs
	}

	d.PanicIfTrue(cur == nil)
	return cur
}
