// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"bytes"
	"crypto/sha512"
	"fmt"

	"github.com/attic-labs/noms/go/d"
)

const (
	HashSize = 32
)

var (
	emptyHash = Hash{}
)

// The hash of a Noms value.
//
// Noms serialization version 1 uses the first 32 bytes of sha512. We could use sha512/256, the
// official standard for a half-size sha512, but it is much less commonly used, meaning that there
// is less likelihood of having a good fast implementation on any given platform. For example, at
// time of writing, there is no native implementation in Node.
//
// sha512 was chosen because:
// - sha1 is no longer recommended
// - blake is not commonly used, not a lot of platform support
// - sha3 is brand new, no library support
// - within sha2, sha512 is faster than sha256 on 64 bit
// - we don't need 512 bit hashes - we don't really even need 256 bit, but 256 is very common and
//   leads to a nice round number of base36 digits: 50.
//
// In Noms, the hash function is a component of the serialization version, which is constant over
// the entire lifetime of a single database. So clients do not need to worry about encountering
// multiple hash functions in the same database.
type Hash struct {
	digest Digest
}

type Digest [HashSize]byte

// Digest returns a *copy* of the digest that backs Hash.
func (r Hash) Digest() Digest {
	return r.digest
}

func (r Hash) IsEmpty() bool {
	return r.digest == emptyHash.digest
}

// DigestSlice returns a slice of the digest that backs A NEW COPY of Hash, because the receiver of this method is not a pointer.
func (r Hash) DigestSlice() []byte {
	return r.digest[:]
}

func New(digest Digest) Hash {
	return Hash{digest}
}

func FromData(data []byte) Hash {
	r := sha512.Sum512(data)
	d := Digest{}
	copy(d[:], r[:HashSize])
	return New(d)
}

// FromSlice creates a new Hash backed by data, ensuring that data is an acceptable length.
func FromSlice(data []byte) Hash {
	d.Chk.True(len(data) == HashSize)
	digest := Digest{}
	copy(digest[:], data)
	return New(digest)
}

func Parse(s string) Hash {
	r, ok := MaybeParse(s)
	if !ok {
		d.Exp.Fail(fmt.Sprintf("Cound not parse Hash: %s", s))
	}
	return r
}

func (r Hash) Less(other Hash) bool {
	return bytes.Compare(r.digest[:], other.digest[:]) < 0
}

func (r Hash) Greater(other Hash) bool {
	return bytes.Compare(r.digest[:], other.digest[:]) > 0
}
