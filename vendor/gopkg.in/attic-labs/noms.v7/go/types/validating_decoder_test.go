// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"gopkg.in/attic-labs/noms.v7/go/chunks"
	"github.com/attic-labs/testify/assert"
)

func TestValidatingBatchingSinkDecode(t *testing.T) {
	v := Number(42)
	c := EncodeValue(v)
	storage := &chunks.TestStorage{}
	vdc := NewValidatingDecoder(storage.NewView())

	dc := vdc.Decode(&c)
	assert.True(t, v.Equals(*dc.Value))
}

func assertPanicsOnInvalidChunk(t *testing.T, data []interface{}) {
	storage := &chunks.TestStorage{}
	vs := NewValueStore(storage.NewView())
	r := &nomsTestReader{data, 0}
	dec := newValueDecoder(r, vs)
	v := dec.readValue()

	c := EncodeValue(v)
	vdc := NewValidatingDecoder(storage.NewView())

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
