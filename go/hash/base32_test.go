// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package hash

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestBase32Encode(t *testing.T) {
	assert := assert.New(t)

	d := make([]byte, 32, 32)
	assert.Equal("0000000000000000000000000000000000000000000000000000", encode(d))
	d[31] = 1
	assert.Equal("000000000000000000000000000000000000000000000000000g", encode(d))
	d[31] = 10
	assert.Equal("0000000000000000000000000000000000000000000000000050", encode(d))
	d[31] = 20
	assert.Equal("00000000000000000000000000000000000000000000000000a0", encode(d))
	d[31] = 31
	assert.Equal("00000000000000000000000000000000000000000000000000fg", encode(d))
	d[31] = 32
	assert.Equal("00000000000000000000000000000000000000000000000000g0", encode(d))
	d[31] = 63
	assert.Equal("00000000000000000000000000000000000000000000000000vg", encode(d))
	d[31] = 64
	assert.Equal("0000000000000000000000000000000000000000000000000100", encode(d))

	// Largest!
	for i := 0; i < 32; i++ {
		d[i] = 0xff
	}
	assert.Equal("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvg", encode(d))
}

func TestBase32Decode(t *testing.T) {
	assert := assert.New(t)

	d := make([]byte, 32, 32)
	assert.Equal(d, decode("0000000000000000000000000000000000000000000000000000"))

	d[31] = 1
	assert.Equal(d, decode("000000000000000000000000000000000000000000000000000g"))
	d[31] = 10
	assert.Equal(d, decode("0000000000000000000000000000000000000000000000000050"))
	d[31] = 20
	assert.Equal(d, decode("00000000000000000000000000000000000000000000000000a0"))
	d[31] = 31
	assert.Equal(d, decode("00000000000000000000000000000000000000000000000000fg"))
	d[31] = 32
	assert.Equal(d, decode("00000000000000000000000000000000000000000000000000g0"))
	d[31] = 63
	assert.Equal(d, decode("00000000000000000000000000000000000000000000000000vg"))
	d[31] = 64
	assert.Equal(d, decode("0000000000000000000000000000000000000000000000000100"))

	// Largest!
	for i := 0; i < 32; i++ {
		d[i] = 0xff
	}
	assert.Equal(d, decode("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvg"))
}
