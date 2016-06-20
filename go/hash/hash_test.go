// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"testing"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/testify/assert"
)

func TestParseError(t *testing.T) {
	assert := assert.New(t)

	assertParseError := func(s string) {
		e := d.Try(func() { Parse(s) })
		assert.IsType(d.UsageError{}, e)
	}

	assertParseError("foo")
	assertParseError("sha512")
	assertParseError("sha512-00000000000000000000000000000000000000000000000000")

	// not enough digits
	assertParseError("0000000000000000000000000000000000000000000000001")

	// too many digits
	assertParseError("000000000000000000000000000000000000000000000000001")

	// '?' not valid char
	assertParseError("0000000000000000000000000000000000000000000000000?")

	r := Parse("00000000000000000000000000000000000000000000000000")
	assert.NotNil(r)
}

func TestMaybeParse(t *testing.T) {
	assert := assert.New(t)

	parse := func(s string, success bool) {
		r, ok := MaybeParse(s)
		assert.Equal(success, ok, "Expected success=%t for %s", success, s)
		if ok {
			assert.Equal(s, r.String())
		} else {
			assert.Equal(emptyHash, r)
		}
	}

	parse("00000000000000000000000000000000000000000000000000", true)
	parse("00000000000000000000000000000000000000000000000001", true)
	parse("", false)
}

func TestEquals(t *testing.T) {
	assert := assert.New(t)

	r0 := Parse("00000000000000000000000000000000000000000000000000")
	r01 := Parse("00000000000000000000000000000000000000000000000000")
	r1 := Parse("00000000000000000000000000000000000000000000000001")

	assert.Equal(r0, r01)
	assert.Equal(r01, r0)
	assert.NotEqual(r0, r1)
	assert.NotEqual(r1, r0)
}

func TestRoundTrip(t *testing.T) {
	s := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcd"
	r := Parse(s)
	assert.Equal(t, s, r.String())
}

func TestDigest(t *testing.T) {
	r := New(Digest{})
	d := r.Digest()
	assert.Equal(t, r.Digest(), d)
	// Digest() must return a copy otherwise things get weird.
	d[0] = 0x01
	assert.NotEqual(t, r.Digest(), d)
}

func TestDigestSlice(t *testing.T) {
	r := New(Digest{})
	d := r.DigestSlice()
	assert.Equal(t, r.DigestSlice(), d)
	// DigestSlice() must return a copy otherwise things get weird.
	d[0] = 0x01
	assert.NotEqual(t, r.DigestSlice(), d)
}

func TestFromData(t *testing.T) {
	r := FromData([]byte("abc"))
	assert.Equal(t, "22hla8qgsgx7cvkw3g39vdw211bqpmgxsfrngm9uuyqgbro3kz", r.String())
}

func TestIsEmpty(t *testing.T) {
	r1 := Hash{}
	assert.True(t, r1.IsEmpty())

	r2 := Parse("00000000000000000000000000000000000000000000000000")
	assert.True(t, r2.IsEmpty())

	r3 := Parse("00000000000000000000000000000000000000000000000001")
	assert.False(t, r3.IsEmpty())
}

func TestLess(t *testing.T) {
	assert := assert.New(t)

	r1 := Parse("00000000000000000000000000000000000000000000000001")
	r2 := Parse("00000000000000000000000000000000000000000000000002")

	assert.False(r1.Less(r1))
	assert.True(r1.Less(r2))
	assert.False(r2.Less(r1))
	assert.False(r2.Less(r2))

	r0 := Hash{}
	assert.False(r0.Less(r0))
	assert.True(r0.Less(r2))
	assert.False(r2.Less(r0))
}

func TestGreater(t *testing.T) {
	assert := assert.New(t)

	r1 := Parse("00000000000000000000000000000000000000000000000001")
	r2 := Parse("00000000000000000000000000000000000000000000000002")

	assert.False(r1.Greater(r1))
	assert.False(r1.Greater(r2))
	assert.True(r2.Greater(r1))
	assert.False(r2.Greater(r2))

	r0 := Hash{}
	assert.False(r0.Greater(r0))
	assert.False(r0.Greater(r2))
	assert.True(r2.Greater(r0))
}
