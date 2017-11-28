// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package chunks provides facilities for representing, storing, and fetching content-addressed chunks of Noms data.
package chunks

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/golang/snappy"
)

// Chunk is a unit of stored data in noms. Chunk data is compressed in storage
// and on the wire.
type Chunk struct {
	h                          hash.Hash
	data                       []byte
	IsCompressed, hashVerified bool
}

var EmptyChunk = New([]byte{})

func (c Chunk) Hash() hash.Hash {
	if !c.hashVerified {
		c.UncompressedData()
	}
	return c.h
}

func (c Chunk) UncompressedData() []byte {
	if !c.IsCompressed {
		return c.data
	}

	ucData, err := snappy.Decode(nil, c.data)
	d.PanicIfError(err)
	if !c.hashVerified {
		vh := hash.Of(ucData)
		d.PanicIfFalse(vh == c.h)
	}

	return ucData
}

func (c Chunk) CompressedData() []byte {
	if c.IsCompressed {
		return c.data
	}

	return snappy.Encode(nil, c.data)
}

func (c Chunk) ByteLen() uint64 {
	return uint64(len(c.data))
}

func (c Chunk) Compress() Chunk {
	d.PanicIfTrue(c.IsCompressed)
	d.PanicIfTrue(c.IsEmpty())
	d.PanicIfFalse(c.hashVerified)
	return Chunk{c.h, snappy.Encode(nil, c.data), true, true}
}

func (c Chunk) IsEmpty() bool {
	return len(c.data) == 0
}

// NewChunk creates a new Chunk backed by data. This means that the returned Chunk has ownership of this slice of memory.
func New(data []byte) Chunk {
	r := hash.Of(data)
	return Chunk{r, data, false, true}
}

// NewChunkWithHash creates a new chunk with a known hash. The hash is not re-calculated or verified. This should obviously only be used in cases where the caller already knows the specified hash is correct.
func FromStorage(r hash.Hash, compressed []byte) Chunk {
	return Chunk{r, compressed, true, true}
}

// TODO: Verify Hash on Decode
func FromWire(r hash.Hash, compressed []byte) Chunk {
	return Chunk{r, compressed, true, false}
}
