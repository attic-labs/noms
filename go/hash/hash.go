// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"regexp"

	"github.com/attic-labs/noms/go/d"
)

const (
	// Because 36^49 < 256^32 < 36^50
	// IOW, this is the smallest number of base36 chars all our hashes can fit in
	hashLen = 50
)

var (
	pattern   = regexp.MustCompile(fmt.Sprintf("^([0-9a-z]{%d})$", hashLen))
	emptyHash = Hash{}
)

type Digest [sha512.Size256]byte

// The core hash datastructure used throughout Noms.
// The current version of the serialization always uses sha512/256.
// If we ever change this, it will be a serialization change, meaning we can mostly assume it doesn't ever change.
type Hash struct {
	digest Digest
}

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
	return New(sha512.Sum512_256(data))
}

// FromSlice creates a new Hash backed by data, ensuring that data is an acceptable length.
func FromSlice(data []byte) Hash {
	d.Chk.True(len(data) == sha1.Size)
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
