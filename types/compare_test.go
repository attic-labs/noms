package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTotalOrdering(t *testing.T) {
	assert := assert.New(t)
	vs := NewTestValueStore()

	// values in increasing order. Some of these are compared by ref so changing the serialization might change the ordering.
	values := []Value{
		Bool(false), Bool(true),
		NewNumber(-10), NewNumber(0), NewNumber(10),
		NewString("a"), NewString("b"), NewString("c"),

		// The order of these are done by the hash.
		vs.WriteValue(NewNumber(10)),
		NewSet(NewNumber(0), NewNumber(1), NewNumber(2), NewNumber(3)),
		NewMap(NewNumber(0), NewNumber(1), NewNumber(2), NewNumber(3)),
		BoolType,
		NewBlob(bytes.NewBuffer([]byte{0x00, 0x01, 0x02, 0x03})),
		NewList(NewNumber(0), NewNumber(1), NewNumber(2), NewNumber(3)),
		NewStruct("a", structData{"x": NewNumber(1), "s": NewString("a")}),

		// Value - values cannot be value
		// Parent - values cannot be parent
		// Union - values cannot be unions
	}

	for i, vi := range values {
		for j, vj := range values {
			if i == j {
				assert.True(vi.Equals(vj))
			} else if i < j {
				x := vi.Less(vj)
				assert.True(x)
			} else {
				x := vi.Less(vj)
				assert.False(x)
			}
		}
	}
}
