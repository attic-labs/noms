// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"gopkg.in/attic-labs/noms.v7/go/hash"
	"github.com/attic-labs/testify/assert"
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

	bl := newBlob(newBlobLeafSequence(nil, []byte("hi")))
	cb := newBlob(newBlobMetaSequence(1, []metaTuple{{Ref{}, newOrderedKey(Number(2)), 2, bl}}, vs))

	ll := newList(newListLeafSequence(nil, String("foo")))
	cl := newList(newMetaSequence(ListKind, 1, []metaTuple{{Ref{}, newOrderedKey(Number(1)), 1, ll}}, vs))

	newStringOrderedKey := func(s string) orderedKey {
		return newOrderedKey(String(s))
	}

	ml := newMap(newMapLeafSequence(nil, mapEntry{String("foo"), String("bar")}))
	cm := newMap(newMetaSequence(MapKind, 1, []metaTuple{{Ref{}, newStringOrderedKey("foo"), 1, ml}}, vs))

	sl := newSet(newSetLeafSequence(nil, String("foo")))
	cps := newSet(newMetaSequence(SetKind, 1, []metaTuple{{Ref{}, newStringOrderedKey("foo"), 1, sl}}, vs))

	count = byte(1)
	values := []Value{
		newBlob(newBlobLeafSequence(nil, []byte{})),
		cb,
		newList(newListLeafSequence(nil, String("bar"))),
		cl,
		cm,
		newMap(newMapLeafSequence(nil)),
		cps,
		newSet(newSetLeafSequence(nil)),
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
