// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"encoding/binary"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/testify/assert"
)

func TestChunkStoreZeroValue(t *testing.T) {
	assert := assert.New(t)
	_, _, store := makeStoreWithFakes(t)
	defer store.Close()

	// No manifest file gets written until the first call to UpdateRoot(). Prior to that, Root() will simply return hash.Hash{}.
	assert.Equal(hash.Hash{}, store.Root())
	assert.Equal(constants.NomsVersion, store.Version())
}

func TestChunkStoreVersion(t *testing.T) {
	assert := assert.New(t)
	_, _, store := makeStoreWithFakes(t)
	defer store.Close()

	assert.Equal(constants.NomsVersion, store.Version())
	newRoot := hash.Of([]byte("new root"))
	if assert.True(store.UpdateRoot(newRoot, hash.Hash{})) {
		assert.Equal(constants.NomsVersion, store.Version())
	}
}

func TestChunkStoreUpdateRoot(t *testing.T) {
	assert := assert.New(t)
	_, _, store := makeStoreWithFakes(t)
	defer store.Close()

	assert.Equal(hash.Hash{}, store.Root())

	newRootChunk := chunks.NewChunk([]byte("new root"))
	newRoot := newRootChunk.Hash()
	store.Put(newRootChunk)
	if assert.True(store.UpdateRoot(newRoot, hash.Hash{})) {
		assert.True(store.Has(newRoot))
		assert.Equal(newRoot, store.Root())
	}

	secondRootChunk := chunks.NewChunk([]byte("newer root"))
	secondRoot := secondRootChunk.Hash()
	store.Put(secondRootChunk)
	if assert.True(store.UpdateRoot(secondRoot, newRoot)) {
		assert.Equal(secondRoot, store.Root())
		assert.True(store.Has(newRoot))
		assert.True(store.Has(secondRoot))
	}
}

func TestChunkStoreManifestAppearsAfterConstruction(t *testing.T) {
	assert := assert.New(t)
	fm, tm, store := makeStoreWithFakes(t)
	defer store.Close()

	assert.Equal(hash.Hash{}, store.Root())
	assert.Equal(constants.NomsVersion, store.Version())

	// Simulate another process writing a manifest after construction.
	chunks := [][]byte{[]byte("hello2"), []byte("goodbye2"), []byte("badbye2")}
	newRoot := hash.Of([]byte("new root"))
	h, _ := tm.Compact(createMemTable(chunks), nil)
	fm.Set(constants.NomsVersion, newRoot, []tableSpec{{h, uint32(len(chunks))}})

	// state in store shouldn't change
	assert.Equal(hash.Hash{}, store.Root())
	assert.Equal(constants.NomsVersion, store.Version())
}

func TestChunkStoreManifestFirstWriteByOtherProcess(t *testing.T) {
	assert := assert.New(t)
	fm := &fakeManifest{}
	tm := newFakeTableManager()

	// Simulate another process having already written a manifest.
	chunks := [][]byte{[]byte("hello2"), []byte("goodbye2"), []byte("badbye2")}
	newRoot := hash.Of([]byte("new root"))
	h, _ := tm.Compact(createMemTable(chunks), nil)
	fm.Set(constants.NomsVersion, newRoot, []tableSpec{{h, uint32(len(chunks))}})

	store := newNomsBlockStore(fm, tm, defaultMemTableSize)
	defer store.Close()

	assert.Equal(newRoot, store.Root())
	assert.Equal(constants.NomsVersion, store.Version())
	assertDataInStore(chunks, store, assert)
}

func TestChunkStoreUpdateRootOptimisticLockFail(t *testing.T) {
	assert := assert.New(t)
	fm, tm, store := makeStoreWithFakes(t)
	defer store.Close()

	// Simulate another process writing a manifest behind store's back.
	chunks := [][]byte{[]byte("hello2"), []byte("goodbye2"), []byte("badbye2")}
	newRoot := hash.Of([]byte("new root"))
	h, _ := tm.Compact(createMemTable(chunks), nil)
	fm.Set(constants.NomsVersion, newRoot, []tableSpec{{h, uint32(len(chunks))}})

	newRoot2 := hash.Of([]byte("new root 2"))
	assert.False(store.UpdateRoot(newRoot2, hash.Hash{}))
	assertDataInStore(chunks, store, assert)
	assert.True(store.UpdateRoot(newRoot2, newRoot))
}

func makeStoreWithFakes(t *testing.T) (fm *fakeManifest, tm tableManager, store *NomsBlockStore) {
	fm = &fakeManifest{}
	tm = newFakeTableManager()
	store = newNomsBlockStore(fm, tm, 0)
	return
}

func createMemTable(chunks [][]byte) *memTable {
	mt := newMemTable(1 << 10)
	for _, c := range chunks {
		mt.addChunk(computeAddr(c), c)
	}
	return mt
}

func assertDataInStore(slices [][]byte, store chunks.ChunkSource, assert *assert.Assertions) {
	for _, data := range slices {
		assert.True(store.Has(chunks.NewChunk(data).Hash()))
	}
}

type fakeManifest struct {
	version    string
	root       hash.Hash
	tableSpecs []tableSpec
}

func (fm *fakeManifest) ParseIfExists(readHook func()) (exists bool, vers string, root hash.Hash, tableSpecs []tableSpec) {
	if fm.root != (hash.Hash{}) {
		return true, fm.version, fm.root, fm.tableSpecs
	}
	return false, constants.NomsVersion, hash.Hash{}, nil
}

func (fm *fakeManifest) Update(tables chunkSources, root, newRoot hash.Hash, writeHook func()) (actual hash.Hash, tableSpecs []tableSpec) {
	if fm.root != root {
		return fm.root, fm.tableSpecs
	}
	fm.version = constants.NomsVersion
	fm.root = newRoot

	newTables := make([]tableSpec, len(fm.tableSpecs))
	known := map[addr]struct{}{}
	for i, t := range fm.tableSpecs {
		known[t.name] = struct{}{}
		newTables[i] = t
	}

	for _, t := range tables {
		if _, present := known[t.hash()]; !present {
			newTables = append(newTables, tableSpec{t.hash(), t.count()})
		}
	}
	fm.tableSpecs = newTables
	return fm.root, fm.tableSpecs
}

func (fm *fakeManifest) Set(version string, root hash.Hash, specs []tableSpec) {
	fm.version, fm.root, fm.tableSpecs = version, root, specs
}

func newFakeTableManager() fakeTableManager {
	return fakeTableManager{map[addr]*memTable{}}
}

type fakeTableManager struct {
	sources map[addr]*memTable
}

func (ftm fakeTableManager) Compact(mt *memTable, haver chunkReader) (name addr, count uint32) {
	scratch := [binary.MaxVarintLen64]byte{}
	binary.PutUvarint(scratch[:], uint64(len(ftm.sources)))
	name = computeAddr(scratch[:])
	ftm.sources[name] = mt
	return name, uint32(len(ftm.sources))
}

func (ftm fakeTableManager) Open(name addr, chunkCount uint32) chunkSource {
	return chunkSourceAdapter{ftm.sources[name], name}
}

type chunkSourceAdapter struct {
	*memTable
	h addr
}

func (csa chunkSourceAdapter) close() error {
	return nil
}

func (csa chunkSourceAdapter) hash() addr {
	return csa.h
}
