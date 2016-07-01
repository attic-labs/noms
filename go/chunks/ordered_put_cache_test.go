// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package chunks

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/testify/suite"
)

func TestLevelDBPutCacheSuite(t *testing.T) {
	suite.Run(t, &LevelDBPutCacheSuite{})
}

type LevelDBPutCacheSuite struct {
	suite.Suite
	cache  *OrderedChunkCache
	chunks []Chunk
	chnx   map[hash.Hash]Chunk
}

func (suite *LevelDBPutCacheSuite) SetupTest() {
	suite.cache = NewOrderedChunkCache()
	suite.chunks = []Chunk{
		NewChunk([]byte("abc")),
		NewChunk([]byte("def")),
		NewChunk([]byte("ghi")),
		NewChunk([]byte("jkl")),
		NewChunk([]byte("mno")),
	}
	suite.chnx = map[hash.Hash]Chunk{}
	for _, c := range suite.chunks {
		suite.chnx[c.Hash()] = c
	}
}

func (suite *LevelDBPutCacheSuite) TearDownTest() {
	suite.cache.Destroy()
}

func (suite *LevelDBPutCacheSuite) TestAddTwice() {
	chunk := suite.chnx[suite.chunks[0].Hash()]
	suite.True(suite.cache.Insert(chunk, 1))
	suite.False(suite.cache.Insert(chunk, 1))
}

func (suite *LevelDBPutCacheSuite) TestAddParallel() {
	hashes := make(chan hash.Hash)
	for _, chunk := range suite.chnx {
		go func(c Chunk) {
			suite.cache.Insert(c, 1)
			hashes <- c.Hash()
		}(chunk)
	}

	for i := 0; i < len(suite.chunks); i++ {
		hash := <-hashes
		suite.True(suite.cache.Has(hash))
		delete(suite.chnx, hash)
	}
	close(hashes)
	suite.Len(suite.chnx, 0)
}

func (suite *LevelDBPutCacheSuite) TestGetParallel() {
	for _, c := range suite.chnx {
		suite.cache.Insert(c, 1)
	}

	chunkChan := make(chan Chunk)
	for h := range suite.chnx {
		go func(h hash.Hash) {
			chunkChan <- suite.cache.Get(h)
		}(h)
	}

	for i := 0; i < len(suite.chunks); i++ {
		c := <-chunkChan
		delete(suite.chnx, c.Hash())
	}
	close(chunkChan)
	suite.Len(suite.chnx, 0)
}

func (suite *LevelDBPutCacheSuite) TestClearParallel() {
	keepIdx := 2
	toClear1, toClear2 := hash.HashSet{}, hash.HashSet{}
	for i, c := range suite.chunks {
		suite.cache.Insert(c, 1)
		if i < keepIdx {
			toClear1.Insert(c.Hash())
		} else if i > keepIdx {
			toClear2.Insert(c.Hash())
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	clear := func(hs hash.HashSet) {
		suite.cache.Clear(hs)
		wg.Done()
	}

	go clear(toClear1)
	go clear(toClear2)

	wg.Wait()
	for i, c := range suite.chunks {
		if i == keepIdx {
			suite.True(suite.cache.Has(c.Hash()))
			continue
		}
		suite.False(suite.cache.Has(c.Hash()))
	}
}

func (suite *LevelDBPutCacheSuite) TestReaderSubset() {
	toExtract := hash.HashSet{}
	for hash, c := range suite.chnx {
		if len(toExtract) < 2 {
			toExtract.Insert(hash)
		}
		suite.cache.Insert(c, 1)
	}

	// Only iterate over the first 2 elements in the DB
	chunkChan := suite.extractChunks(toExtract)
	count := 0
	for c := range chunkChan {
		if suite.Contains(toExtract, c.Hash()) {
			count++
		}
	}
	suite.Equal(len(toExtract), count)
}

func (suite *LevelDBPutCacheSuite) TestExtractChunksOrder() {
	maxHeight := len(suite.chnx)
	orderedHashes := make(hash.HashSlice, maxHeight)
	toExtract := hash.HashSet{}
	heights := rand.Perm(maxHeight)
	for hash, c := range suite.chnx {
		toExtract.Insert(hash)
		orderedHashes[heights[0]] = hash
		suite.cache.Insert(c, uint64(heights[0]))
		heights = heights[1:]
	}

	chunkChan := suite.extractChunks(toExtract)
	for c := range chunkChan {
		suite.Equal(orderedHashes[0], c.Hash())
		orderedHashes = orderedHashes[1:]
	}
	suite.Len(orderedHashes, 0)
}

func (suite *LevelDBPutCacheSuite) extractChunks(hashes hash.HashSet) <-chan *Chunk {
	chunkChan := make(chan *Chunk)
	suite.cache.ExtractChunks(hashes, chunkChan)
	return chunkChan
}
