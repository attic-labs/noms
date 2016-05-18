package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberEquals(t *testing.T) {
	assert := assert.New(t)
	n1 := NewNumber(2)
	n2 := NewNumber(2.0)
	n3 := n2
	n4 := NewNumber(uint64(3))
	assert.True(n1.Equals(n2))
	assert.True(n2.Equals(n1))
	assert.True(n1.Equals(n3))
	assert.True(n3.Equals(n1))
	assert.False(n1.Equals(n4))
	assert.False(n4.Equals(n1))
}
