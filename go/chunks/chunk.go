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
	h                        hash.Hash
	uncompressed, compressed []byte
	hashVerified             bool
}

var EmptyChunk = New([]byte{})

func (c Chunk) Hash() hash.Hash {
	if !c.hashVerified {
		c.UncompressedData()
	}
	return c.h
}

func (c Chunk) UncompressedData() []byte {
	if c.uncompressed != nil {
		d.PanicIfFalse(c.hashVerified)
		return c.uncompressed
	}

	b, err := snappy.Decode(nil, c.compressed)
	d.PanicIfError(err)
	if !c.hashVerified {
		vh := hash.Of(b)
		d.PanicIfFalse(vh == c.h)
		c.hashVerified = true
	}
	return b
}

func (c Chunk) CompressedData() []byte {
	if c.compressed != nil {
		return c.compressed
	}
	return snappy.Encode(nil, c.uncompressed)
}

func (c Chunk) ByteLen() uint64 {
	if c.compressed != nil {
		return uint64(len(c.compressed))
	}

	return uint64(len(c.uncompressed))
}

func (c Chunk) Compress() {
	if c.compressed != nil {
		return
	}

	d.PanicIfTrue(c.IsEmpty())
	c.compressed = snappy.Encode(nil, c.uncompressed)
	c.uncompressed = nil
}

func (c Chunk) IsEmpty() bool {
	return c.compressed == nil && len(c.uncompressed) == 0
}

// NewChunk creates a new Chunk backed by data. This means that the returned Chunk has ownership of this slice of memory.
func New(data []byte) Chunk {
	r := hash.Of(data)
	return Chunk{r, data, nil, true}
}

// NewChunkWithHash creates a new chunk with a known hash. The hash is not re-calculated or verified. This should obviously only be used in cases where the caller already knows the specified hash is correct.
func FromStorage(r hash.Hash, compressed []byte) Chunk {
	return Chunk{r, nil, compressed, true}
}

// TODO: Verify Hash on Decode
func FromWire(r hash.Hash, compressed []byte) Chunk {
	return Chunk{r, nil, compressed, false}
}
