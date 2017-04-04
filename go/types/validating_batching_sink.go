// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
)

const batchSize = 100

type ValidatingBatchingSink struct {
	vs    *ValueStore
	cs    chunks.ChunkStore
	batch [batchSize]chunks.Chunk
	count int
	pool  sync.Pool
}

func NewValidatingBatchingSink(cs chunks.ChunkStore) *ValidatingBatchingSink {
	return &ValidatingBatchingSink{
		vs: newLocalValueStore(cs),
		cs: cs,
	}
}

// DecodedChunk holds a pointer to a Chunk and the Value that results from
// calling DecodeFromBytes(c.Data()).
type DecodedChunk struct {
	Chunk *chunks.Chunk
	Value *Value
}

// DecodeUnqueued decodes c and checks that the hash of the resulting value
// matches c.Hash(). It returns a DecodedChunk holding both c and a pointer to
// the decoded Value. However, if c has already been Enqueued, DecodeUnqueued
// returns an empty DecodedChunk.
func (vbs *ValidatingBatchingSink) DecodeUnqueued(c *chunks.Chunk) DecodedChunk {
	h := c.Hash()
	// TODO: Will thisstill be a thing?
	// if vbs.vs.isPresent(h) {
	// 	return DecodedChunk{}
	// }
	v := decodeFromBytesWithValidation(c.Data(), vbs.vs)
	if getHash(v) != h {
		d.Panic("Invalid hash found")
	}
	return DecodedChunk{c, &v}
}

// Enqueue adds c to the queue of Chunks waiting to be Put into vbs' backing
// ChunkStore. It is assumed that v is the Value decoded from c, and so v can
// be used to validate the ref-completeness of c.  The instance keeps an
// internal buffer of Chunks, spilling to the ChunkStore when the buffer is
// full.
// TODO: v is not used right now, but will almost certainly be needed for BUG 3180
func (vbs *ValidatingBatchingSink) Enqueue(c chunks.Chunk, v Value) {
	vbs.batch[vbs.count] = c
	vbs.count++

	if vbs.count == batchSize {
		vbs.cs.PutMany(vbs.batch[:vbs.count])
		vbs.count = 0
	}
}

// Flush Puts any Chunks buffered by Enqueue calls into the backing
// ChunkStore.
func (vbs *ValidatingBatchingSink) Flush() {
	if vbs.count > 0 {
		vbs.cs.PutMany(vbs.batch[:vbs.count])
	}
	vbs.cs.Flush()
	vbs.count = 0
}
