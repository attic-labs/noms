package hash

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestString(t *testing.T) {
	assert := assert.New(t)
	h := Hash{}
	assert.Equal(
		"00000000000000000000000000000000000000000000000000",
		h.String())

	h.digest[len(h.digest)-1] = 0x10
	assert.Equal(
		"0000000000000000000000000000000000000000000000000g",
		h.String())

	h.digest[len(h.digest)-1] = 0x23
	assert.Equal(
		"0000000000000000000000000000000000000000000000000z",
		h.String())

	h.digest[len(h.digest)-1] = 0xff
	assert.Equal(
		"00000000000000000000000000000000000000000000000073",
		h.String())

	for i, _ := range h.digest {
		h.digest[i] = 0xff
	}
	assert.Equal(
		"6dp5qcb22im238nr3wvp0ic7q99w035jmy2iw7i6n43d37jtof",
		h.String())
}
