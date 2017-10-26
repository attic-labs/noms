// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package chunks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	c := New([]byte("abc"))
	h := c.Hash()
	// See http://www.di-mgt.com.au/sha_testvectors.html
	assert.Equal(t, "rmnjb8cjc5tblj21ed4qs821649eduie", h.String())
}
