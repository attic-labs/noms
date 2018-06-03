// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"encoding/binary"

	"github.com/attic-labs/noms/go/hash"
)

// Integer is a Noms Value wrapper around the primitive int64 type.
type Integer int64

// Value interface
func (v Integer) Value() Value {
	return v
}

func (v Integer) Equals(other Value) bool {
	return v == other
}

func (v Integer) Less(other Value) bool {
	if v2, ok := other.(Integer); ok {
		return v < v2
	}
	return IntegerKind < other.Kind()
}

func (v Integer) Hash() hash.Hash {
	return getHash(v)
}

func (v Integer) WalkValues(cb ValueCallback) {
}

func (v Integer) WalkRefs(cb RefCallback) {
}

func (v Integer) typeOf() *Type {
	return IntegerType
}

func (v Integer) Kind() NomsKind {
	return IntegerKind
}

func (v Integer) valueReadWriter() ValueReadWriter {
	return nil
}

func (v Integer) writeTo(w nomsWriter) {
	IntegerKind.writeTo(w)
	w.writeInteger(v)
}

func (v Integer) valueBytes() []byte {
	// We know the size of the buffer here so allocate it once.
	// IntegerKind, int (Varint), exp (Varint)
	buff := make([]byte, 1+binary.MaxVarintLen64)
	w := binaryNomsWriter{buff, 0}
	v.writeTo(&w)
	// TODO(ORBAT): note that the backing array of the returned slice is still of length 1+binary.MaxVarintLen64.
	return buff[:w.offset]
}
