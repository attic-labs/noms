// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

// BatchStoreAdapter provides a naive implementation of BatchStore should only be used with ChunkStores that can Put relatively quickly. It provides no actual batching or validation. Its intended use is for adapting a ChunkStore for use in something that requires a BatchStore.
type BatchStoreAdapter struct {
	cs   chunks.ChunkStore
	once *sync.Once
}

// NewBatchStoreAdapter returns a BatchStore instance backed by a ChunkStore. Takes ownership of cs and manages its lifetime; calling Close on the returned BatchStore will Close cs.
func NewBatchStoreAdapter(cs chunks.ChunkStore) BatchStore {
	return &BatchStoreAdapter{cs, &sync.Once{}}
}

func (bsa *BatchStoreAdapter) IsValidating() bool {
	return false
}

// Get simply proxies to the backing ChunkStore
func (bsa *BatchStoreAdapter) Get(h hash.Hash) chunks.Chunk {
	bsa.once.Do(bsa.expectVersion)
	return bsa.cs.Get(h)
}

// SchedulePut simply calls Put on the underlying ChunkStore, and ignores hints.
func (bsa *BatchStoreAdapter) SchedulePut(c chunks.Chunk, refHeight uint64, hints Hints) {
	bsa.once.Do(bsa.expectVersion)
	bsa.cs.Put(c)
}

func (bsa *BatchStoreAdapter) expectVersion() {
	dataVersion := bsa.cs.Version()
	d.PanicIfTrue(constants.NomsVersion != dataVersion, "SDK version %s incompatible with data of version %s", constants.NomsVersion, dataVersion)
}

// AddHints is a noop.
func (bsa *BatchStoreAdapter) AddHints(hints Hints) {}

// Flush is a noop.
func (bsa *BatchStoreAdapter) Flush() {}

// Close closes the underlying ChunkStore
func (bsa *BatchStoreAdapter) Close() error {
	bsa.Flush()
	return bsa.cs.Close()
}

type ValidatingBatchStoreAdapter struct {
	cs     chunks.ChunkStore
	pc     *chunks.OrderedChunkCache
	vbs    *ValidatingBatchingSink
	hints  Hints
	hashes hash.HashSet
	mu     *sync.Mutex
	once   *sync.Once
}

func NewValidatingBatchStoreAdapter(vs *ValueStore, cs chunks.ChunkStore) BatchStore {
	return &ValidatingBatchStoreAdapter{
		cs,
		chunks.NewOrderedChunkCache(),
		NewValidatingBatchingSink(vs, cs, staticTypeCache),
		Hints{},
		hash.HashSet{},
		&sync.Mutex{},
		&sync.Once{},
	}
}

func (lvbs *ValidatingBatchStoreAdapter) expectVersion() {
	dataVersion := lvbs.cs.Version()
	d.PanicIfTrue(constants.NomsVersion != dataVersion, "SDK version %s incompatible with data of version %s", constants.NomsVersion, dataVersion)
}

func (lvbs *ValidatingBatchStoreAdapter) IsValidating() bool {
	return true
}

func (lvbs *ValidatingBatchStoreAdapter) Get(h hash.Hash) chunks.Chunk {
	lvbs.once.Do(lvbs.expectVersion)
	return lvbs.pc.Get(h)
}

func (lvbs *ValidatingBatchStoreAdapter) SchedulePut(c chunks.Chunk, refHeight uint64, hints Hints) {
	lvbs.once.Do(lvbs.expectVersion)

	lvbs.mu.Lock()
	defer lvbs.mu.Unlock()
	lvbs.pc.Insert(c, refHeight)
	lvbs.hashes.Insert(c.Hash())
	lvbs.AddHints(hints)
}

func (lvbs *ValidatingBatchStoreAdapter) AddHints(hints Hints) {
	for h := range hints {
		lvbs.hints[h] = struct{}{}
	}
}

func (lvbs *ValidatingBatchStoreAdapter) Flush() {
	lvbs.once.Do(lvbs.expectVersion)

	lvbs.vbs.Prepare(lvbs.hints)

	defer lvbs.pc.Clear(lvbs.hashes)
	chunkChan := make(chan *chunks.Chunk, 1024)
	lvbs.pc.ExtractChunks(lvbs.hashes, chunkChan)
	for c := range chunkChan {
		lvbs.vbs.Enqueue(*c)
	}
	lvbs.vbs.Flush()
}

func (lvbs *ValidatingBatchStoreAdapter) Close() error {
	lvbs.pc.Destroy()
	return nil
}
