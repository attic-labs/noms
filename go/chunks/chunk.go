// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package chunks provides facilities for representing, storing, and fetching content-addressed chunks of Noms data.
package chunks

import (
	"github.com/attic-labs/noms/go/hash"
)

// Chunk is a unit of stored data in noms
type Chunk struct {
	r    hash.Hash
	data []byte
}

var EmptyChunk = NewChunk([]byte{})

func (c Chunk) Hash() hash.Hash {
	return c.r
}

func (c Chunk) Data() []byte {
	return c.data
}

func (c Chunk) IsEmpty() bool {
	return len(c.data) == 0
}

// NewChunk creates a new Chunk backed by data. This means that the returned Chunk has ownership of this slice of memory.
func NewChunk(data []byte) Chunk {
	r := hash.Of(data)
	return Chunk{r, data}
}

// NewChunkWithHash creates a new chunk with a known hash. The hash is not re-calculated or verified. This should obviously only be used in cases where the caller already knows the specified hash is correct.
func NewChunkWithHash(r hash.Hash, data []byte) Chunk {
	return Chunk{r, data}
}
