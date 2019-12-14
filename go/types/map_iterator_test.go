// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapIterator(t *testing.T) {
	assert := assert.New(t)

	vrw := newTestValueStore()

	me := NewMap(vrw).Edit()
	for i := 0; i < 5; i++ {
		me.Set(String(string(byte(65+i))), Number(i))
	}

	m := me.Map()

	tc := []struct {
		reverse  bool
		iter     bool
		iterAt   uint64
		iterFrom string
		expected []string
	}{
		{false, true, 0, "", []string{"A", "B", "C", "D", "E"}},
		{false, false, 0, "", []string{"A", "B", "C", "D", "E"}},
		{false, false, 2, "", []string{"C", "D", "E"}},
		{false, false, 4, "", []string{"E"}},
		{false, false, 5, "", []string{}},
		{false, false, 0, "A", []string{"A", "B", "C", "D", "E"}},
		{false, false, 0, "C", []string{"C", "D", "E"}},
		{false, false, 4, "E", []string{"E"}},
		{false, false, 0, "AA", []string{"B", "C", "D", "E"}},
		{false, false, 0, "F", []string{}},
		{true, false, 0, "", []string{}},
		{true, true, 0, "", []string{}},
		{true, false, 2, "", []string{"C", "B", "A"}},
		{true, false, 4, "", []string{"E", "D", "C", "B", "A"}},
		{true, false, 5, "", []string{}},
		{true, false, 0, "A", []string{"A"}},
		{true, false, 0, "C", []string{"C", "B", "A"}},
		{true, false, 0, "E", []string{"E", "D", "C", "B", "A"}},
		{true, false, 0, "AA", []string{"B", "A"}},
		{true, false, 0, "F", []string{}},
	}

	for i, t := range tc {
		lbl := fmt.Sprintf("test case %d", i)
		var it *MapIterator
		if t.iter {
			it = m.Iterator()
		} else if t.iterFrom != "" {
			it = m.IteratorFrom(String(t.iterFrom))
		} else {
			it = m.IteratorAt(t.iterAt)
		}
		for i, e := range t.expected {
			lbl := fmt.Sprintf("%s: iteration %d", lbl, i)
			assert.True(it.Valid(), lbl)

			assert.Equal(e, string(it.Key().(String)), lbl)
			assert.True(m.Get(it.Key()).Equals(it.Value()), lbl)

			k, v := it.Entry()
			assert.Equal(e, string(k.(String)), lbl)
			assert.True(m.Get(it.Key()).Equals(v), lbl)

			assert.True(m.Get(it.Key()).Equals(Number(it.Position())), lbl)

			var last bool
			if t.reverse {
				last = it.Prev()
			} else {
				last = it.Next()
			}
			assert.Equal(i < len(t.expected)-1, last, lbl)
			assert.Equal(i < len(t.expected)-1, it.Valid(), lbl)
		}
	}
}
