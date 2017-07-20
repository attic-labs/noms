// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"gopkg.in/attic-labs/noms.v7/go/hash"
)

// Number is a Noms Value wrapper around the primitive float64 type.
type Number float64

// Value interface
func (v Number) Value(vrw ValueReadWriter) Value {
	return v
}

func (v Number) Equals(other Value) bool {
	return v == other
}

func (v Number) Less(other Value) bool {
	if v2, ok := other.(Number); ok {
		return v < v2
	}
	return NumberKind < other.Kind()
}

func (v Number) Hash() hash.Hash {
	return getHash(v)
}

func (v Number) WalkValues(cb ValueCallback) {
}

func (v Number) WalkRefs(cb RefCallback) {
}

func (v Number) typeOf() *Type {
	return NumberType
}

func (v Number) Kind() NomsKind {
	return NumberKind
}
