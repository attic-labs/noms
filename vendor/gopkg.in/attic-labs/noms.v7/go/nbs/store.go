// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"gopkg.in/attic-labs/noms.v7/go/chunks"
	"gopkg.in/attic-labs/noms.v7/go/constants"
	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/hash"
)

// The root of a Noms Chunk Store is stored in a 'manifest', along with the
// names of the tables that hold all the chunks in the store. The number of
// chunks in each table is also stored in the manifest.

const (
	// StorageVersion is the version of the on-disk Noms Chunks Store data format.
	StorageVersion = "4"

	defaultMemTableSize uint64 = (1 << 20) * 128 // 128MB
	defaultMaxTables           = 256

	defaultIndexCacheSize    = (1 << 20) * 8 // 8MB
	defaultManifestCacheSize = 1 << 23       // 8MB
)

var (
	cacheOnce           = sync.Once{}
	globalIndexCache    *indexCache
	makeCachingManifest func(manifest) cachingManifest
	globalFDCache       *fdCache
	globalConjoiner     conjoiner
)

func makeGlobalCaches() {
	globalIndexCache = newIndexCache(defaultIndexCacheSize)
	globalFDCache = newFDCache(defaultMaxTables)
	globalConjoiner = newAsyncConjoiner(defaultMaxTables)

	manifestCache := newManifestCache(defaultManifestCacheSize)
	makeCachingManifest = func(mm manifest) cachingManifest { return cachingManifest{mm, manifestCache} }
}

type NomsBlockStore struct {
	mm           cachingManifest
	p            tablePersister
	c            conjoiner
	manifestLock addr
	nomsVersion  string

	mu     sync.RWMutex // protects the following state
	mt     *memTable
	tables tableSet
	root   hash.Hash

	mtSize   uint64
	putCount uint64

	stats *Stats
}

func NewAWSStore(table, ns, bucket string, s3 s3svc, ddb ddbsvc, memTableSize uint64) *NomsBlockStore {
	cacheOnce.Do(makeGlobalCaches)
	p := &s3TablePersister{
		s3,
		bucket,
		defaultS3PartSize,
		minS3PartSize,
		maxS3PartSize,
		globalIndexCache,
		make(chan struct{}, 32),
		nil,
	}
	mm := makeCachingManifest(newDynamoManifest(table, ns, ddb))
	return newNomsBlockStore(mm, p, globalConjoiner, memTableSize)
}

func NewLocalStore(dir string, memTableSize uint64) *NomsBlockStore {
	cacheOnce.Do(makeGlobalCaches)
	d.PanicIfError(checkDir(dir))

	mm := makeCachingManifest(fileManifest{dir})
	p := newFSTablePersister(dir, globalFDCache, globalIndexCache)
	return newNomsBlockStore(mm, p, globalConjoiner, memTableSize)
}

func newNomsBlockStore(mm cachingManifest, p tablePersister, c conjoiner, memTableSize uint64) *NomsBlockStore {
	if memTableSize == 0 {
		memTableSize = defaultMemTableSize
	}
	nbs := &NomsBlockStore{
		mm:          mm,
		p:           p,
		c:           c,
		tables:      newTableSet(p),
		nomsVersion: constants.NomsVersion,
		mtSize:      memTableSize,
		stats:       NewStats(),
	}

	t1 := time.Now()
	defer func() {
		nbs.stats.OpenLatency.SampleTimeSince(t1)
	}()

	if exists, contents := nbs.mm.ParseIfExists(nbs.stats, nil); exists {
		nbs.nomsVersion, nbs.manifestLock, nbs.root = contents.vers, contents.lock, contents.root
		nbs.tables = nbs.tables.Rebase(contents.specs)
	}

	return nbs
}

func newNomsBlockStoreWithContents(mm cachingManifest, mc manifestContents, p tablePersister, c conjoiner, memTableSize uint64) *NomsBlockStore {
	if memTableSize == 0 {
		memTableSize = defaultMemTableSize
	}
	return &NomsBlockStore{
		mm:     mm,
		p:      p,
		c:      c,
		mtSize: memTableSize,
		stats:  NewStats(),

		nomsVersion:  mc.vers,
		manifestLock: mc.lock,
		root:         mc.root,
		tables:       newTableSet(p).Rebase(mc.specs),
	}
}

func (nbs *NomsBlockStore) Put(c chunks.Chunk) {
	t1 := time.Now()
	a := addr(c.Hash())
	d.PanicIfFalse(nbs.addChunk(a, c.Data()))
	nbs.putCount++

	nbs.stats.PutLatency.SampleTimeSince(t1)
}

// TODO: figure out if there's a non-error reason for this to return false. If not, get rid of return value.
func (nbs *NomsBlockStore) addChunk(h addr, data []byte) bool {
	nbs.mu.Lock()
	defer nbs.mu.Unlock()
	if nbs.mt == nil {
		nbs.mt = newMemTable(nbs.mtSize)
	}
	if !nbs.mt.addChunk(h, data) {
		nbs.tables = nbs.tables.Prepend(nbs.mt, nbs.stats)
		nbs.mt = newMemTable(nbs.mtSize)
		return nbs.mt.addChunk(h, data)
	}
	return true
}

func (nbs *NomsBlockStore) Get(h hash.Hash) chunks.Chunk {
	t1 := time.Now()
	defer func() {
		nbs.stats.GetLatency.SampleTimeSince(t1)
		nbs.stats.ChunksPerGet.Sample(1)
	}()

	a := addr(h)
	data, tables := func() (data []byte, tables chunkReader) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		if nbs.mt != nil {
			data = nbs.mt.get(a, nbs.stats)
		}
		return data, nbs.tables
	}()
	if data != nil {
		return chunks.NewChunkWithHash(h, data)
	}
	if data := tables.get(a, nbs.stats); data != nil {
		return chunks.NewChunkWithHash(h, data)
	}

	return chunks.EmptyChunk
}

func (nbs *NomsBlockStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	t1 := time.Now()
	reqs := toGetRecords(hashes)

	defer func() {
		if len(hashes) > 0 {
			nbs.stats.GetLatency.SampleTimeSince(t1)
			nbs.stats.ChunksPerGet.Sample(uint64(len(reqs)))
		}
	}()

	wg := &sync.WaitGroup{}

	tables, remaining := func() (tables chunkReader, remaining bool) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		tables = nbs.tables
		remaining = true
		if nbs.mt != nil {
			remaining = nbs.mt.getMany(reqs, foundChunks, nil, nbs.stats)
		}

		return
	}()

	if remaining {
		tables.getMany(reqs, foundChunks, wg, nbs.stats)
		wg.Wait()
	}

}

func toGetRecords(hashes hash.HashSet) []getRecord {
	reqs := make([]getRecord, len(hashes))
	idx := 0
	for h := range hashes {
		a := addr(h)
		reqs[idx] = getRecord{
			a:      &a,
			prefix: a.Prefix(),
		}
		idx++
	}

	sort.Sort(getRecordByPrefix(reqs))
	return reqs
}

func (nbs *NomsBlockStore) CalcReads(hashes hash.HashSet, blockSize uint64) (reads int, split bool) {
	reqs := toGetRecords(hashes)
	tables := func() (tables tableSet) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		tables = nbs.tables

		return
	}()

	reads, split, remaining := tables.calcReads(reqs, blockSize)
	d.Chk.False(remaining)
	return
}

func (nbs *NomsBlockStore) extractChunks(chunkChan chan<- *chunks.Chunk) {
	ch := make(chan extractRecord, 1)
	go func() {
		defer close(ch)
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		// Chunks in nbs.tables were inserted before those in nbs.mt, so extract chunks there _first_
		nbs.tables.extract(ch)
		if nbs.mt != nil {
			nbs.mt.extract(ch)
		}
	}()
	for rec := range ch {
		c := chunks.NewChunkWithHash(hash.Hash(rec.a), rec.data)
		chunkChan <- &c
	}
}

func (nbs *NomsBlockStore) Count() uint32 {
	count, tables := func() (count uint32, tables chunkReader) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		if nbs.mt != nil {
			count = nbs.mt.count()
		}
		return count, nbs.tables
	}()
	return count + tables.count()
}

func (nbs *NomsBlockStore) Has(h hash.Hash) bool {
	t1 := time.Now()
	defer func() {
		nbs.stats.HasLatency.SampleTimeSince(t1)
		nbs.stats.AddressesPerHas.Sample(1)
	}()

	a := addr(h)
	has, tables := func() (bool, chunkReader) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		return nbs.mt != nil && nbs.mt.has(a), nbs.tables
	}()
	has = has || tables.has(a)

	return has
}

func (nbs *NomsBlockStore) HasMany(hashes hash.HashSet) hash.HashSet {
	t1 := time.Now()

	reqs := toHasRecords(hashes)

	tables, remaining := func() (tables chunkReader, remaining bool) {
		nbs.mu.RLock()
		defer nbs.mu.RUnlock()
		tables = nbs.tables

		remaining = true
		if nbs.mt != nil {
			remaining = nbs.mt.hasMany(reqs)
		}

		return
	}()

	if remaining {
		tables.hasMany(reqs)
	}

	if len(hashes) > 0 {
		nbs.stats.HasLatency.SampleTimeSince(t1)
		nbs.stats.AddressesPerHas.SampleLen(len(reqs))
	}

	absent := hash.HashSet{}
	for _, r := range reqs {
		if !r.has {
			absent.Insert(hash.New(r.a[:]))
		}
	}
	return absent
}

func toHasRecords(hashes hash.HashSet) []hasRecord {
	reqs := make([]hasRecord, len(hashes))
	idx := 0
	for h := range hashes {
		a := addr(h)
		reqs[idx] = hasRecord{
			a:      &a,
			prefix: a.Prefix(),
			order:  idx,
		}
		idx++
	}

	sort.Sort(hasRecordByPrefix(reqs))
	return reqs
}

func (nbs *NomsBlockStore) Rebase() {
	nbs.mu.Lock()
	defer nbs.mu.Unlock()
	if exists, contents := nbs.mm.ParseIfExists(nbs.stats, nil); exists {
		nbs.nomsVersion, nbs.manifestLock, nbs.root = contents.vers, contents.lock, contents.root
		nbs.tables = nbs.tables.Rebase(contents.specs)
	}
}

func (nbs *NomsBlockStore) Root() hash.Hash {
	nbs.mu.RLock()
	defer nbs.mu.RUnlock()
	return nbs.root
}

func (nbs *NomsBlockStore) Commit(current, last hash.Hash) bool {
	t1 := time.Now()
	defer func() {
		nbs.stats.CommitLatency.SampleTimeSince(t1)
	}()

	anyPossiblyNovelChunks := func() bool {
		nbs.mu.Lock()
		defer nbs.mu.Unlock()
		return nbs.mt != nil || len(nbs.tables.novel) > 0
	}

	if !anyPossiblyNovelChunks() && current == last {
		nbs.Rebase()
		return true
	}

	for {
		if err := nbs.updateManifest(current, last); err == nil {
			return true
		} else if err == errOptimisticLockFailedRoot || err == errLastRootMismatch {
			return false
		}
	}
}

var (
	errLastRootMismatch           = fmt.Errorf("last does not match nbs.Root()")
	errOptimisticLockFailedRoot   = fmt.Errorf("Root moved")
	errOptimisticLockFailedTables = fmt.Errorf("Tables changed")
)

func (nbs *NomsBlockStore) updateManifest(current, last hash.Hash) error {
	nbs.mu.Lock()
	defer nbs.mu.Unlock()
	if nbs.root != last {
		return errLastRootMismatch
	}

	handleOptimisticLockFailure := func(upstream manifestContents) error {
		nbs.manifestLock = upstream.lock
		nbs.root = upstream.root
		nbs.tables = nbs.tables.Rebase(upstream.specs)

		if last != upstream.root {
			return errOptimisticLockFailedRoot
		}
		return errOptimisticLockFailedTables
	}

	if upstream, doomed := nbs.mm.updateWillFail(nbs.manifestLock); doomed {
		// Pre-emptive optimistic lock failure. Someone else in-process moved to the root, the set of tables, or both out from under us.
		return handleOptimisticLockFailure(upstream)
	}

	if nbs.mt != nil && nbs.mt.count() > 0 {
		nbs.tables = nbs.tables.Prepend(nbs.mt, nbs.stats)
		nbs.mt = nil
	}

	if nbs.c.ConjoinRequired(nbs.tables) {
		nbs.c.Conjoin(nbs.mm, nbs.p, nbs.tables.Novel(), nbs.stats)
		exists, upstream := nbs.mm.ParseIfExists(nbs.stats, nil)
		d.PanicIfFalse(exists)

		nbs.manifestLock = upstream.lock
		nbs.root = upstream.root
		nbs.tables = nbs.tables.Rebase(upstream.specs)
		return errOptimisticLockFailedTables
	}

	specs := nbs.tables.ToSpecs()
	newContents := manifestContents{
		vers:  constants.NomsVersion,
		root:  current,
		lock:  generateLockHash(current, specs),
		specs: specs,
	}
	upstream := nbs.mm.Update(nbs.manifestLock, newContents, nbs.stats, nil)
	if newContents.lock != upstream.lock {
		// Optimistic lock failure. Someone else moved to the root, the set of tables, or both out from under us.
		return handleOptimisticLockFailure(upstream)
	}

	nbs.tables = nbs.tables.Flatten()
	nbs.nomsVersion, nbs.manifestLock, nbs.root = constants.NomsVersion, newContents.lock, current
	return nil
}

func (nbs *NomsBlockStore) Version() string {
	return nbs.nomsVersion
}

func (nbs *NomsBlockStore) Close() (err error) {
	return
}

func (nbs *NomsBlockStore) Stats() interface{} {
	return *nbs.stats
}
