// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"math/rand"
	"sync"
	"testing"

	"github.com/attic-labs/testify/assert"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestS3TablePersisterPersist(t *testing.T) {
	assert := assert.New(t)
	mt := newMemTable(testMemTableSize)

	for _, c := range testChunks {
		assert.True(mt.addChunk(computeAddr(c), c))
	}

	s3svc := makeFakeS3(assert)
	cache := newIndexCache(1024)
	s3p := s3TablePersister{s3: s3svc, bucket: "bucket", partSize: calcPartSize(mt, 3), indexCache: cache}

	src := s3p.Persist(mt, nil)
	assert.NotNil(cache.get(src.hash()))

	if assert.True(src.count() > 0) {
		if r := s3svc.readerForTable(src.hash()); assert.NotNil(r) {
			assertChunksInReader(testChunks, r, assert)
		}
	}
}

func calcPartSize(rdr chunkReader, maxPartNum int) int {
	return int(maxTableSize(uint64(rdr.count()), rdr.uncompressedLen())) / maxPartNum
}

func TestS3TablePersisterPersistSinglePart(t *testing.T) {
	assert := assert.New(t)
	mt := newMemTable(testMemTableSize)

	for _, c := range testChunks {
		assert.True(mt.addChunk(computeAddr(c), c))
	}

	s3svc := makeFakeS3(assert)
	s3p := s3TablePersister{s3: s3svc, bucket: "bucket", partSize: calcPartSize(mt, 1)}

	src := s3p.Persist(mt, nil)
	if assert.True(src.count() > 0) {
		if r := s3svc.readerForTable(src.hash()); assert.NotNil(r) {
			assertChunksInReader(testChunks, r, assert)
		}
	}
}

func TestS3TablePersisterPersistAbort(t *testing.T) {
	assert := assert.New(t)
	mt := newMemTable(testMemTableSize)

	for _, c := range testChunks {
		assert.True(mt.addChunk(computeAddr(c), c))
	}

	numParts := 4
	s3svc := &failingFakeS3{makeFakeS3(assert), sync.Mutex{}, 1}
	s3p := s3TablePersister{s3: s3svc, bucket: "bucket", partSize: calcPartSize(mt, numParts)}

	assert.Panics(func() { s3p.Persist(mt, nil) })
}

type failingFakeS3 struct {
	*fakeS3
	mu           sync.Mutex
	numSuccesses int
}

func (m *failingFakeS3) UploadPart(input *s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.numSuccesses > 0 {
		m.numSuccesses--
		return m.fakeS3.UploadPart(input)
	}
	return nil, mockAWSError("MalformedXML")
}

func TestS3TablePersisterCompactNoData(t *testing.T) {
	assert := assert.New(t)
	mt := newMemTable(testMemTableSize)
	existingTable := newMemTable(testMemTableSize)

	for _, c := range testChunks {
		assert.True(mt.addChunk(computeAddr(c), c))
		assert.True(existingTable.addChunk(computeAddr(c), c))
	}

	s3svc := makeFakeS3(assert)
	s3p := s3TablePersister{s3: s3svc, bucket: "bucket", partSize: 1 << 10}

	src := s3p.Persist(mt, existingTable)
	assert.True(src.count() == 0)

	_, present := s3svc.data[src.hash().String()]
	assert.False(present)
}

func TestS3TablePersisterCompactAll(t *testing.T) {
	assert := assert.New(t)

	s3svc := makeFakeS3(assert)
	cache := newIndexCache(1024)
	rl := make(chan struct{}, 8)
	defer close(rl)

	s3p := s3TablePersister{s3: s3svc, bucket: "bucket", partSize: defaultS3PartSize, indexCache: cache, readRl: rl}

	chunx := make([][][]byte, 3)
	sources := make(chunkSources, len(chunx))
	rnd := rand.New(rand.NewSource(0))
	MiB := 1 << 20
	for i := 0; i < len(chunx); i++ {
		chunx[i] = make([][]byte, (defaultS3PartSize/MiB)-1+i)
		mt := newMemTable(2 * defaultS3PartSize)

		for j := 0; j < len(chunx[i]); j++ {
			chunx[i][j] = make([]byte, MiB)
			rnd.Read(chunx[i][j])
			mt.addChunk(computeAddr(chunx[i][j]), chunx[i][j])
		}
		sources[i] = s3p.Persist(mt, nil)
	}
	src := s3p.CompactAll(sources)
	assert.NotNil(cache.get(src.hash()))

	if assert.True(src.count() > 0) {
		if r := s3svc.readerForTable(src.hash()); assert.NotNil(r) {
			for _, slice := range chunx {
				assertChunksInReader(slice, r, assert)
			}
		}
	}
}

func bytesToChunkSource(bs ...[]byte) chunkSource {
	sum := 0
	for _, b := range bs {
		sum += len(b)
	}
	maxSize := maxTableSize(uint64(len(bs)), uint64(sum))
	buff := make([]byte, maxSize)
	tw := newTableWriter(buff, nil)
	for _, b := range bs {
		tw.addChunk(computeAddr(b), b)
	}
	tableSize, name := tw.finish()
	data := buff[:tableSize]
	rdr := newTableReader(parseTableIndex(data), bytes.NewReader(data), fileBlockSize)
	return chunkSourceAdapter{rdr, name}
}

func TestCompactSourcesToBufferPanic(t *testing.T) {
	assert := assert.New(t)
	rl := make(chan struct{}, 1)
	defer close(rl)

	src := bytesToChunkSource([]byte("hello"))
	pcs := panicingChunkSource{src}

	assert.Panics(func() { compactSourcesToBuffer(chunkSources{pcs}, rl) })
}

type panicingChunkSource struct {
	chunkSource
}

func (pcs panicingChunkSource) extract(chunks chan<- extractRecord) {
	panic("onoes")
}
