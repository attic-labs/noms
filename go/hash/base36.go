// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"fmt"
	"math/big"
	"regexp"
)

const (
	// The length of the String() form of Noms Hashes in bytes/chars.
	StringLen = 50
)

var (
	pattern = regexp.MustCompile(fmt.Sprintf("^([0-9a-z]{%d})$", StringLen))

	alphabet = [...]byte{
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
		'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
		'u', 'v', 'w', 'x', 'y', 'z',
	}

	alphabetSize = big.NewInt(int64(len(alphabet)))
	byteSize     = big.NewInt(int64(256))
	zero         = &big.Int{}
	lookup       = map[byte]int{}
)

func init() {
	for i, b := range alphabet {
		lookup[b] = i
	}
}

// Returns a string form of the hash, for use in user interface.
//
// Noms uses big-endian base36 for hash stringification with the alphabet {0-9,a-z}.
// Stringified hashes are left-padded with zero.
//
// Goals driving this decision included:
// - We want the serialization to be as dense as possible
// - We want all hashes to be the same length when stringified
// - We want numeric sort to be the same as lexicographic sort for ease of visual inspection and
//   debugging
// - We would like to not have to distinguish upper vs lower case letters when speaking hashes
//
// Some of these goals are in conflict, but a 50-digit base36 string seemed like a good balance of
// forces.
func (h Hash) String() string {
	// Decode into a bigint from base256 big-endian. We use big-endian for the binary encoding just
	// for consistency with the string encoding.
	n := &big.Int{}
	for i, b := range h.digest {
		e := len(h.digest) - i - 1
		v := &big.Int{}
		v.Exp(byteSize, big.NewInt(int64(e)), nil)
		v.Mul(v, big.NewInt(int64(b)))
		n.Add(n, v)
	}

	// Encode the bigint as base36 big-endian left-padded with zero
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
	// - decode from big-endian base36 left-padded with zero
	// - encode as big-endian base256 byte array left-padded with zero
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
