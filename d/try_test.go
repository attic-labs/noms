package d

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func IsUsageError(a *assert.Assertions, f func()) {
	e := Try(f)
	a.IsType(UsageError{}, e)
}

func TestTry(t *testing.T) {
	assert := assert.New(t)

	IsUsageError(assert, func() { Exp.Fail("hey-o") })

	assert.Panics(func() {
		Try(func() { Chk.Fail("hey-o") })
	})

	assert.Panics(func() {
		Try(func() { panic("hey-o") })
	})
}
