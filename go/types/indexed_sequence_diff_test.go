// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestIndexedSequenceDiffWithMetaNodeGap(t *testing.T) {
	assert := assert.New(t)

	newSequenceMetaTuple := func(vs ...Value) metaTuple {
		seq := newListLeafSequence(nil, vs...)
		list := newList(seq)
		return newMetaTuple(NewRef(list), orderedKeyFromInt(len(vs)), uint64(len(vs)), list)
	}

	m1 := newSequenceMetaTuple(Number(1), Number(2))
	m2 := newSequenceMetaTuple(Number(3), Number(4))
	m3 := newSequenceMetaTuple(Number(5), Number(6))
	m4 := newSequenceMetaTuple(Number(7), Number(8))

	// Compared to the previous patch, this test now passes. However:
	// - Changing `m4` to `m3` in `s1` yields `{{1, 0, 1, 1}}` when it should yield `{{2, 0, 2, 2}}`.
	// - Appending `m4` to `s2` yields `{{1, 0, 2, 1}}` when it should yield `{{2, 0, 4, 2}}`.
	s1 := newListMetaSequence([]metaTuple{m1, m4}, nil)
	s2 := newListMetaSequence([]metaTuple{m1, m2, m3}, nil)

	changes := make(chan Splice)
	go func() {
		// TODO: Also test diff(s2, s1).
		depth := func(s indexedSequence) int { return newCursorAtIndex(s, 0).depth() }
		indexedSequenceDiff(s1, depth(s1), 0, s2, depth(s2), 0, changes, nil, DEFAULT_MAX_SPLICE_MATRIX_SIZE)
		close(changes)
	}()

	expected := []Splice{{2, 2, 4, 2}}
	i := 0
	for c := range changes {
		assert.Equal(expected[i], c)
		i++
	}
	assert.Equal(len(expected), i)
}
