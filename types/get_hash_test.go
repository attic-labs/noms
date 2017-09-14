// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"github.com/attic-labs/noms/hash"
	"github.com/stretchr/testify/assert"
)

func TestEnsureHash(t *testing.T) {
	assert := assert.New(t)
	vs := newTestValueStore()
	count := byte(1)
	mockGetRef := func(v Value) hash.Hash {
		h := hash.Hash{}
		h[0] = count
		count++
		return h
	}
	testRef := func(h hash.Hash, expected byte) {
		assert.Equal(expected, h[0])
		for i := 1; i < hash.ByteLen; i++ {
			assert.Equal(byte(0), h[i])
		}
	}

	getHashOverride = mockGetRef
	defer func() {
		getHashOverride = nil
	}()

	bl := newBlob(newBlobLeafSequence(vs, []byte("hi")))
	cb := newBlob(newBlobMetaSequence(1, []metaTuple{{vs.WriteValue(bl), newOrderedKey(Number(2)), 2}}, vs))

	ll := newList(newListLeafSequence(vs, String("foo")))
	cl := newList(newMetaSequence(ListKind, 1, []metaTuple{{vs.WriteValue(ll), newOrderedKey(Number(1)), 1}}, vs))

	newStringOrderedKey := func(s string) orderedKey {
		return newOrderedKey(String(s))
	}

	ml := newMap(newMapLeafSequence(vs, mapEntry{String("foo"), String("bar")}))
	cm := newMap(newMetaSequence(MapKind, 1, []metaTuple{{vs.WriteValue(ml), newStringOrderedKey("foo"), 1}}, vs))

	sl := newSet(newSetLeafSequence(vs, String("foo")))
	cps := newSet(newMetaSequence(SetKind, 1, []metaTuple{{vs.WriteValue(sl), newStringOrderedKey("foo"), 1}}, vs))

	count = byte(1)
	values := []Value{
		newBlob(newBlobLeafSequence(vs, []byte{})),
		cb,
		newList(newListLeafSequence(vs, String("bar"))),
		cl,
		cm,
		newMap(newMapLeafSequence(vs)),
		cps,
		newSet(newSetLeafSequence(vs)),
	}
	for i := 0; i < 2; i++ {
		for j, v := range values {
			testRef(v.Hash(), byte(j+1))
		}
	}

	for _, v := range values {
		expected := byte(0x42)
		h := hash.Hash{}
		h[0] = expected
		assignHash(v.(hashCacher), h)
		testRef(v.Hash(), expected)
	}

	count = byte(1)
	values = []Value{
		Bool(false),
		Number(0),
		String(""),
	}
	for i := 0; i < 2; i++ {
		for j, v := range values {
			testRef(v.Hash(), byte(i*len(values)+(j+1)))
		}
	}
}
