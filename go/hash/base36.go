// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"math/big"
)

var (
	alphabet = [...]byte{
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
		'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
		'u', 'v', 'w', 'x', 'y', 'z',
	}
	alphabetSize = big.NewInt(int64(len(alphabet)))
	byteSize     = big.NewInt(int64(256))
	zero         = &big.Int{}
	lookup 		 = map[byte]int{}
)

func init() {
	for i, b := range alphabet {
		lookup[b] = i
	}
}

// Returns a serialized version of the hash.
// Noms uses big endian base36 for hash serialization with the alphabet {0-9,a-z}.
// Goals driving this decision:
// - serialization should be as dense as possible
// - want all hashes to be same length
// - want numeric sort to be same as lexicographic sort for easier visual inspection
// - would like to not have to distinguish upper vs lower case letters when
//   spelling hash out loud
func (h Hash) String() string {
	// Treat the entire digest as a big-endian bigint.
	// We use big-endian for consistency with the base36 version, below.
	n := &big.Int{}
	for i, b := range h.digest {
		e := len(h.digest) - i - 1
		v := &big.Int{}
		v.Exp(byteSize, big.NewInt(int64(e)), nil)
		v.Mul(v, big.NewInt(int64(b)))
		n.Add(n, v)
	}

	// Re-encode the same number in big-endian base36.
	// We use big-endian because it results in nicer sorting characteristics:
	// - lexicographic sort is equivalent to numeric sort
	// - when displaying lists of sorted hashes, they will
	//   appear sorted in the typical human way.
	r := [50]byte{}
	for i := len(r) - 1; i >= 0; i-- {
		remainder := big.Int{}
		n.DivMod(n, alphabetSize, &remainder)
		r[i] = alphabet[int(remainder.Int64())]
	}
	return string(r[:])
}

func MaybeParse(s string) (r Hash, ok bool) {
	match := pattern.FindStringSubmatch(s)
	if match == nil {
		return
	}

	// Opposite procedure as String():
	// - decode the string as a big-endian bigint
	// - encode as big-endian byte array 
	n := &big.Int{}
	for i := 0; i < len(s); i++ {
		e := len(s) - i - 1
		v := &big.Int{}
		v.Exp(alphabetSize, big.NewInt(int64(e)), nil)
		v.Mul(v, big.NewInt(int64(lookup[s[i]])))
		n.Add(n, v)
	}

	for i := len(r.digest) - 1; i >= 0; i-- {
		remainder := big.Int{}
		n.DivMod(n, byteSize, &remainder)
		r.digest[i] = byte(remainder.Int64())
	}

	ok = true
	return
}
