// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/attic-labs/noms.v7/go/chunks"
	"gopkg.in/attic-labs/noms.v7/go/constants"
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/hash"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

const testMemTableSize = 1 << 8

func TestBlockStoreSuite(t *testing.T) {
	suite.Run(t, &BlockStoreSuite{})
}

type BlockStoreSuite struct {
	suite.Suite
	dir        string
	store      *NomsBlockStore
	putCountFn func() int
}

func (suite *BlockStoreSuite) SetupTest() {
	var err error
	suite.dir, err = ioutil.TempDir("", "")
	suite.NoError(err)
	suite.store = NewLocalStore(suite.dir, testMemTableSize)
	suite.putCountFn = func() int {
		return int(suite.store.putCount)
	}
}

func (suite *BlockStoreSuite) TearDownTest() {
	suite.store.Close()
	os.RemoveAll(suite.dir)
}

func (suite *BlockStoreSuite) TestChunkStoreMissingDir() {
	newDir := filepath.Join(suite.dir, "does-not-exist")
	suite.Panics(func() { NewLocalStore(newDir, testMemTableSize) })
}

func (suite *BlockStoreSuite) TestChunkStoreNotDir() {
	existingFile := filepath.Join(suite.dir, "path-exists-but-is-a-file")
	os.Create(existingFile)
	suite.Panics(func() { NewLocalStore(existingFile, testMemTableSize) })
}

func (suite *BlockStoreSuite) TestChunkStorePut() {
	input := []byte("abc")
	c := chunks.NewChunk(input)
	suite.store.Put(c)
	h := c.Hash()

	// See http://www.di-mgt.com.au/sha_testvectors.html
	suite.Equal("rmnjb8cjc5tblj21ed4qs821649eduie", h.String())

	suite.store.Commit(h, suite.store.Root()) // Commit writes

	// And reading it via the API should work...
	assertInputInStore(input, h, suite.store, suite.Assert())
	if suite.putCountFn != nil {
		suite.Equal(1, suite.putCountFn())
	}

	// Re-writing the same data should cause a second put
	c = chunks.NewChunk(input)
	suite.store.Put(c)
	suite.Equal(h, c.Hash())
	assertInputInStore(input, h, suite.store, suite.Assert())
	suite.store.Commit(h, suite.store.Root()) // Commit writes

	if suite.putCountFn != nil {
		suite.Equal(2, suite.putCountFn())
	}
}

func (suite *BlockStoreSuite) TestChunkStorePutMany() {
	input1, input2 := []byte("abc"), []byte("def")
	c1, c2 := chunks.NewChunk(input1), chunks.NewChunk(input2)
	suite.store.Put(c1)
	suite.store.Put(c2)

	suite.store.Commit(c1.Hash(), suite.store.Root()) // Commit writes

	// And reading it via the API should work...
	assertInputInStore(input1, c1.Hash(), suite.store, suite.Assert())
	assertInputInStore(input2, c2.Hash(), suite.store, suite.Assert())
	if suite.putCountFn != nil {
		suite.Equal(2, suite.putCountFn())
	}
}

func (suite *BlockStoreSuite) TestChunkStorePutMoreThanMemTable() {
	input1, input2 := make([]byte, testMemTableSize/2+1), make([]byte, testMemTableSize/2+1)
	rand.Read(input1)
	rand.Read(input2)
	c1, c2 := chunks.NewChunk(input1), chunks.NewChunk(input2)
	suite.store.Put(c1)
	suite.store.Put(c2)

	suite.store.Commit(c1.Hash(), suite.store.Root()) // Commit writes

	// And reading it via the API should work...
	assertInputInStore(input1, c1.Hash(), suite.store, suite.Assert())
	assertInputInStore(input2, c2.Hash(), suite.store, suite.Assert())
	if suite.putCountFn != nil {
		suite.Equal(2, suite.putCountFn())
	}
	suite.Len(suite.store.tables.ToSpecs(), 2)
}

func (suite *BlockStoreSuite) TestChunkStoreGetMany() {
	inputs := [][]byte{make([]byte, testMemTableSize/2+1), make([]byte, testMemTableSize/2+1), []byte("abc")}
	rand.Read(inputs[0])
	rand.Read(inputs[1])
	chnx := make([]chunks.Chunk, len(inputs))
	for i, data := range inputs {
		chnx[i] = chunks.NewChunk(data)
		suite.store.Put(chnx[i])
	}
	suite.store.Commit(chnx[0].Hash(), suite.store.Root()) // Commit writes

	hashes := make(hash.HashSlice, len(chnx))
	for i, c := range chnx {
		hashes[i] = c.Hash()
	}

	chunkChan := make(chan *chunks.Chunk, len(hashes))
	suite.store.GetMany(hashes.HashSet(), chunkChan)
	close(chunkChan)

	found := make(hash.HashSlice, 0)
	for c := range chunkChan {
		found = append(found, c.Hash())
	}

	sort.Sort(found)
	sort.Sort(hashes)
	suite.True(found.Equals(hashes))
}

func (suite *BlockStoreSuite) TestChunkStoreHasMany() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	for _, c := range chnx {
		suite.store.Put(c)
	}
	suite.store.Commit(chnx[0].Hash(), suite.store.Root()) // Commit writes
	notPresent := chunks.NewChunk([]byte("ghi")).Hash()

	hashes := hash.NewHashSet(chnx[0].Hash(), chnx[1].Hash(), notPresent)
	absent := suite.store.HasMany(hashes)

	suite.Len(absent, 1)
	for _, c := range chnx {
		suite.False(absent.Has(c.Hash()), "%s present in %v", c.Hash(), absent)
	}
	suite.True(absent.Has(notPresent))
}

func (suite *BlockStoreSuite) TestChunkStoreExtractChunks() {
	input1, input2 := make([]byte, testMemTableSize/2+1), make([]byte, testMemTableSize/2+1)
	rand.Read(input1)
	rand.Read(input2)
	chnx := []chunks.Chunk{chunks.NewChunk(input1), chunks.NewChunk(input2)}
	for _, c := range chnx {
		suite.store.Put(c)
	}

	chunkChan := make(chan *chunks.Chunk)
	go func() { suite.store.extractChunks(chunkChan); close(chunkChan) }()
	i := 0
	for c := range chunkChan {
		suite.Equal(chnx[i].Data(), c.Data())
		suite.Equal(chnx[i].Hash(), c.Hash())
		i++
	}
}

func (suite *BlockStoreSuite) TestChunkStoreFlushOptimisticLockFail() {
	input1, input2 := []byte("abc"), []byte("def")
	c1, c2 := chunks.NewChunk(input1), chunks.NewChunk(input2)
	root := suite.store.Root()

	interloper := NewLocalStore(suite.dir, testMemTableSize)
	interloper.Put(c1)
	suite.True(interloper.Commit(interloper.Root(), interloper.Root()))

	suite.store.Put(c2)
	suite.True(suite.store.Commit(suite.store.Root(), suite.store.Root()))

	// Reading c2 via the API should work...
	assertInputInStore(input2, c2.Hash(), suite.store, suite.Assert())
	// And so should reading c1 via the API
	assertInputInStore(input1, c1.Hash(), suite.store, suite.Assert())

	suite.True(interloper.Commit(c1.Hash(), interloper.Root())) // Commit root

	// Updating from stale root should fail...
	suite.False(suite.store.Commit(c2.Hash(), root))
	// ...but new root should succeed
	suite.True(suite.store.Commit(c2.Hash(), suite.store.Root()))
}

func (suite *BlockStoreSuite) TestChunkStoreRebaseOnNoOpFlush() {
	input1 := []byte("abc")
	c1 := chunks.NewChunk(input1)

	interloper := NewLocalStore(suite.dir, testMemTableSize)
	interloper.Put(c1)
	suite.True(interloper.Commit(c1.Hash(), interloper.Root()))

	suite.False(suite.store.Has(c1.Hash()))
	suite.Equal(hash.Hash{}, suite.store.Root())
	// Should Rebase, even though there's no work to do.
	suite.True(suite.store.Commit(suite.store.Root(), suite.store.Root()))

	// Reading c1 via the API should work
	assertInputInStore(input1, c1.Hash(), suite.store, suite.Assert())
	suite.True(suite.store.Has(c1.Hash()))
}

func (suite *BlockStoreSuite) TestChunkStorePutWithRebase() {
	input1, input2 := []byte("abc"), []byte("def")
	c1, c2 := chunks.NewChunk(input1), chunks.NewChunk(input2)
	root := suite.store.Root()

	interloper := NewLocalStore(suite.dir, testMemTableSize)
	interloper.Put(c1)
	suite.True(interloper.Commit(interloper.Root(), interloper.Root()))

	suite.store.Put(c2)

	// Reading c2 via the API should work pre-rebase
	assertInputInStore(input2, c2.Hash(), suite.store, suite.Assert())
	// Shouldn't have c1 yet.
	suite.False(suite.store.Has(c1.Hash()))

	suite.store.Rebase()

	// Reading c2 via the API should work post-rebase
	assertInputInStore(input2, c2.Hash(), suite.store, suite.Assert())
	// And so should reading c1 via the API
	assertInputInStore(input1, c1.Hash(), suite.store, suite.Assert())

	// Commit interloper root
	suite.True(interloper.Commit(c1.Hash(), interloper.Root()))

	// suite.store should still have its initial root
	suite.EqualValues(root, suite.store.Root())
	suite.store.Rebase()

	// Rebase grabbed the new root, so updating should now succeed!
	suite.True(suite.store.Commit(c2.Hash(), suite.store.Root()))

	// Interloper shouldn't see c2 yet....
	suite.False(interloper.Has(c2.Hash()))
	interloper.Rebase()
	// ...but post-rebase it must
	assertInputInStore(input2, c2.Hash(), interloper, suite.Assert())
}

func TestBlockStoreConjoinOnCommit(t *testing.T) {
	stats := &Stats{}
	assertContainAll := func(t *testing.T, store chunks.ChunkStore, srcs ...chunkSource) {
		rdrs := make(chunkReaderGroup, len(srcs))
		for i, src := range srcs {
			rdrs[i] = src
		}
		chunkChan := make(chan extractRecord, rdrs.count())
		rdrs.extract(chunkChan)
		close(chunkChan)

		for rec := range chunkChan {
			assert.True(t, store.Has(hash.Hash(rec.a)))
		}
	}

	newChunk := chunks.NewChunk([]byte("gnu"))

	t.Run("NoConjoin", func(t *testing.T) {
		mm := cachingManifest{&fakeManifest{}, newManifestCache(0)}
		p := newFakeTablePersister()
		c := &fakeConjoiner{}

		smallTableStore := newNomsBlockStore(mm, p, c, testMemTableSize)

		root := smallTableStore.Root()
		smallTableStore.Put(newChunk)
		assert.True(t, smallTableStore.Commit(newChunk.Hash(), root))
		assert.True(t, smallTableStore.Has(newChunk.Hash()))
	})

	makeCanned := func(conjoinees, keepers []tableSpec, p tablePersister) cannedConjoin {
		srcs := chunkSources{}
		for _, sp := range conjoinees {
			srcs = append(srcs, p.Open(sp.name, sp.chunkCount))
		}
		conjoined := p.ConjoinAll(srcs, stats)
		cannedSpecs := []tableSpec{{conjoined.hash(), conjoined.count()}}
		return cannedConjoin{true, append(cannedSpecs, keepers...)}
	}

	t.Run("ConjoinSuccess", func(t *testing.T) {
		fm := &fakeManifest{}
		p := newFakeTablePersister()

		srcs := makeTestSrcs([]uint32{1, 1, 3, 7}, p)
		upstream := toSpecs(srcs)
		fm.set(constants.NomsVersion, computeAddr([]byte{0xbe}), hash.Of([]byte{0xef}), upstream)
		c := &fakeConjoiner{
			[]cannedConjoin{makeCanned(upstream[:2], upstream[2:], p)},
		}

		smallTableStore := newNomsBlockStore(cachingManifest{fm, newManifestCache(0)}, p, c, testMemTableSize)

		root := smallTableStore.Root()
		smallTableStore.Put(newChunk)
		assert.True(t, smallTableStore.Commit(newChunk.Hash(), root))
		assert.True(t, smallTableStore.Has(newChunk.Hash()))
		assertContainAll(t, smallTableStore, srcs...)
	})

	t.Run("ConjoinRetry", func(t *testing.T) {
		fm := &fakeManifest{}
		p := newFakeTablePersister()

		srcs := makeTestSrcs([]uint32{1, 1, 3, 7, 13}, p)
		upstream := toSpecs(srcs)
		fm.set(constants.NomsVersion, computeAddr([]byte{0xbe}), hash.Of([]byte{0xef}), upstream)
		c := &fakeConjoiner{
			[]cannedConjoin{
				makeCanned(upstream[:2], upstream[2:], p),
				makeCanned(upstream[:4], upstream[4:], p),
			},
		}

		smallTableStore := newNomsBlockStore(cachingManifest{fm, newManifestCache(0)}, p, c, testMemTableSize)

		root := smallTableStore.Root()
		smallTableStore.Put(newChunk)
		assert.True(t, smallTableStore.Commit(newChunk.Hash(), root))
		assert.True(t, smallTableStore.Has(newChunk.Hash()))
		assertContainAll(t, smallTableStore, srcs...)
	})
}

type cannedConjoin struct {
	should bool
	specs  []tableSpec // Must name tables that are already persisted
}

type fakeConjoiner struct {
	canned []cannedConjoin
}

func (fc *fakeConjoiner) ConjoinRequired(ts tableSet) bool {
	if len(fc.canned) == 0 {
		return false
	}
	return fc.canned[0].should
}

func (fc *fakeConjoiner) Conjoin(mm manifest, p tablePersister, novelCount int, stats *Stats) {
	d.PanicIfTrue(len(fc.canned) == 0)
	canned := fc.canned[0]
	fc.canned = fc.canned[1:]

	_, contents := mm.ParseIfExists(stats, nil)
	newContents := manifestContents{
		vers:  constants.NomsVersion,
		root:  contents.root,
		specs: canned.specs,
		lock:  generateLockHash(contents.root, canned.specs),
	}
	contents = mm.Update(contents.lock, newContents, stats, nil)
	d.PanicIfFalse(contents.lock == newContents.lock)
}

func assertInputInStore(input []byte, h hash.Hash, s chunks.ChunkStore, assert *assert.Assertions) {
	c := s.Get(h)
	assert.False(c.IsEmpty(), "Shouldn't get empty chunk for %s", h.String())
	assert.Zero(bytes.Compare(input, c.Data()), "%s != %s", string(input), string(c.Data()))
}

func (suite *BlockStoreSuite) TestChunkStoreGetNonExisting() {
	h := hash.Parse("11111111111111111111111111111111")
	c := suite.store.Get(h)
	suite.True(c.IsEmpty())
}
