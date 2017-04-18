// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/testify/assert"
)

func TestValidatingBatchingSinkDecode(t *testing.T) {
	v := Number(42)
	c := EncodeValue(v, nil)
	vdc := NewValidatingDecoder(chunks.NewTestStore())

	dc := vdc.Decode(&c)
	assert.True(t, v.Equals(*dc.Value))
}

func assertPanicsOnInvalidChunk(t *testing.T, data []interface{}) {
	cs := chunks.NewTestStore()
	vs := NewValueStore(cs)
	r := &nomsTestReader{data, 0}
	dec := newValueDecoder(r, vs)
	v := dec.readValue()

	c := EncodeValue(v, nil)
	vdc := NewValidatingDecoder(cs)

	assert.Panics(t, func() {
		vdc.Decode(&c)
	})
}

func TestValidatingBatchingSinkDecodeInvalidUnion(t *testing.T) {
	data := []interface{}{
		uint8(TypeKind),
		uint8(UnionKind), uint64(2) /* len */, uint8(NumberKind), uint8(BoolKind),
	}
	assertPanicsOnInvalidChunk(t, data)
}

func TestValidatingBatchingSinkDecodeInvalidStructFieldOrder(t *testing.T) {
	data := []interface{}{
		uint8(TypeKind),
		uint8(StructKind), "S", uint64(2), /* len */
		"b", "a",
		uint8(NumberKind), uint8(NumberKind),
		false, false,
	}
	assertPanicsOnInvalidChunk(t, data)
}

func TestValidatingBatchingSinkDecodeInvalidStructName(t *testing.T) {
	data := []interface{}{
		uint8(TypeKind),
		uint8(StructKind), "S ", uint64(0), /* len */
	}
	assertPanicsOnInvalidChunk(t, data)
}

func TestValidatingBatchingSinkDecodeInvalidStructFieldName(t *testing.T) {
	data := []interface{}{
		uint8(TypeKind),
		uint8(StructKind), "S", uint64(1), /* len */
		"b ", uint8(NumberKind), false,
	}
	assertPanicsOnInvalidChunk(t, data)
}
