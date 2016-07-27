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

	depth := func(s indexedSequence) int {
		return newCursorAtIndex(s, 0).depth()
	}

	m1 := newSequenceMetaTuple(Number(1), Number(2))
	m2 := newSequenceMetaTuple(Number(3), Number(4))
	m3 := newSequenceMetaTuple(Number(5), Number(6))

	s1 := newListMetaSequence([]metaTuple{m1, m3}, nil)
	s2 := newListMetaSequence([]metaTuple{m1, m2, m3}, nil)

	changes := make(chan Splice)
	go func() {
		indexedSequenceDiff(s1, depth(s1), 0, s2, depth(s2), 0, changes, nil, DEFAULT_MAX_SPLICE_MATRIX_SIZE)
		close(changes)
	}()

	expected := []Splice{{2, 0, 2, 2}}
	i := 0
	for c := range changes {
		assert.Equal(expected[i], c)
		i++
	}
	assert.Equal(len(expected), i)
}
