// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"crypto/sha1"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type nomsComparer struct{}

func (nomsComparer) Compare(a, b []byte) int {
	if compared, res := compareEmpties(a, b); compared {
		return res
	}
	aKind, bKind := NomsKind(a[0]), NomsKind(b[0])
	switch aKind {
	default:
		if bKind <= StringKind {
			return 1
		}
		a, b = a[1:], b[1:]
		d.Chk.True(len(a) == sha1.Size && len(b) == sha1.Size, "Compared objects should be %d bytes long, not %d and %d", sha1.Size, len(a), len(b))
		aHash, bHash := hash.FromSlice(a), hash.FromSlice(b)
		if aHash == bHash {
			d.Chk.True(aKind == bKind, "%d != %d, but Values with the same hash MUST be the same Kind", aKind, bKind)
			return 0
		}
		if aHash.Less(bHash) {
			return -1
		}
		return 1
	case BoolKind, NumberKind, StringKind:
		if res := compareKinds(aKind, bKind); res != 0 {
			return res
		}
		vA := newValueDecoder(&binaryNomsReader{a, 0}, nil).readValue()
		vB := newValueDecoder(&binaryNomsReader{b, 0}, nil).readValue()
		if vA.Equals(vB) {
			return 0
		}
		if vA.Less(vB) {
			return -1
		}
		return 1
	}
}

func compareEmpties(a, b []byte) (bool, int) {
	aLen, bLen := len(a), len(b)
	if aLen > 0 && bLen > 0 {
		return false, 0
	}
	if aLen == 0 {
		if bLen == 0 {
			return true, 0
		}
		return true, -1
	}
	return true, 1
}

func compareKinds(aKind, bKind NomsKind) (res int) {
	if aKind < bKind {
		res = -1
	} else if aKind > bKind {
		res = 1
	}
	return
}

func (nomsComparer) Name() string {
	return "noms.ValueComparator"
}

func (nomsComparer) Successor(dst, b []byte) []byte {
	return nil
}

func (nomsComparer) Separator(dst, a, b []byte) []byte {
	return nil
}
