// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "gopkg.in/attic-labs/noms.v7/go/hash"

// String is a Noms Value wrapper around the primitive string type.
type String string

// Value interface
func (s String) Value(vrw ValueReadWriter) Value {
	return s
}

func (s String) Equals(other Value) bool {
	return s == other
}

func (s String) Less(other Value) bool {
	if s2, ok := other.(String); ok {
		return s < s2
	}
	return StringKind < other.Kind()
}

func (s String) Hash() hash.Hash {
	return getHash(s)
}

func (s String) WalkValues(cb ValueCallback) {
}

func (s String) WalkRefs(cb RefCallback) {
}

func (s String) typeOf() *Type {
	return StringType
}

func (s String) Kind() NomsKind {
	return StringKind
}
