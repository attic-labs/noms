// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

const batchSize = 100

type ValidatingBatchingSink struct {
	vs      *ValueStore
	cs      chunks.ChunkStore
	batch   [batchSize]chunks.Chunk
	count   int
	novel   hash.HashSet
	targets hash.HashSet
}

func NewValidatingBatchingSink(cs chunks.ChunkStore) *ValidatingBatchingSink {
	return &ValidatingBatchingSink{
		vs:      newLocalValueStore(cs),
		cs:      cs,
		novel:   hash.HashSet{},
		targets: hash.HashSet{},
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
// the decoded Value.
func (vbs *ValidatingBatchingSink) DecodeUnqueued(c *chunks.Chunk) DecodedChunk {
	h := c.Hash()
	v := decodeFromBytesWithValidation(c.Data(), vbs.vs)
	if getHash(v) != h {
		d.Panic("Invalid hash found")
	}
	return DecodedChunk{c, &v}
}

// Enqueue adds c to the queue of Chunks waiting to be Put into vbs' backing
// ChunkStore. It is assumed that v is the Value decoded from c, and so v can
// be used to validate the ref-completeness of c. The instance keeps an
// internal buffer of Chunks, spilling to the ChunkStore when the buffer is
// full.
func (vbs *ValidatingBatchingSink) Enqueue(c chunks.Chunk, v Value) {
	h := c.Hash()
	vbs.novel.Insert(h)
	vbs.targets.Remove(h)
	v.WalkRefs(func(ref Ref) {
		if target := ref.TargetHash(); !vbs.novel.Has(target) {
			vbs.targets.Insert(target)
		}
	})

	vbs.batch[vbs.count] = c
	vbs.count++

	if vbs.count == batchSize {
		vbs.Flush()
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

// PanicIfDangling does a Has check on all the references encountered
// while enqueuing novel chunks. It panics if any of these refs point
// to Chunks that don't exist in the backing ChunkStore.
func (vbs *ValidatingBatchingSink) PanicIfDangling() {
	present := vbs.cs.HasMany(vbs.targets)
	absent := hash.HashSlice{}
	for h := range vbs.targets {
		if !present.Has(h) {
			absent = append(absent, h)
		}
	}
	if len(absent) != 0 {
		d.Panic("Found dangling references to %v", absent)
	}
}
