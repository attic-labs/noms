// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/sizecache"
)

type tablePersister interface {
	Persist(mt *memTable, haver chunkReader) chunkSource
	CompactAll(sources chunkSources) chunkSource
	Open(name addr, chunkCount uint32) chunkSource
}

type indexCache struct {
	cache *sizecache.SizeCache
}

// Returns an indexCache which will burn roughly |size| bytes of memory
func newIndexCache(size uint64) *indexCache {
	return &indexCache{sizecache.New(size)}
}

func (sic indexCache) get(name addr) (tableIndex, bool) {
	idx, found := sic.cache.Get(name)
	if found {
		return idx.(tableIndex), true
	}

	return tableIndex{}, false
}

func (sic indexCache) put(name addr, idx tableIndex) {
	indexSize := uint64(idx.chunkCount) * (addrSize + ordinalSize + lengthSize + uint64Size)
	sic.cache.Add(name, indexSize, idx)
}

type chunkSourcesByAscendingCount chunkSources

func (csbc chunkSourcesByAscendingCount) Len() int { return len(csbc) }
func (csbc chunkSourcesByAscendingCount) Less(i, j int) bool {
	srcI, srcJ := csbc[i], csbc[j]
	if srcI.count() == srcJ.count() {
		hi, hj := srcI.hash(), srcJ.hash()
		return bytes.Compare(hi[:], hj[:]) < 0
	}
	return srcI.count() < srcJ.count()
}
func (csbc chunkSourcesByAscendingCount) Swap(i, j int) { csbc[i], csbc[j] = csbc[j], csbc[i] }

type compactionPlan struct {
	sources             chunkSources
	chunkDataLens       []uint64
	mergedIndex         []byte
	chunkCount          uint32
	totalCompressedData uint64
}

func (cp compactionPlan) lengths() []byte {
	lengthsStart := uint64(cp.chunkCount) * prefixTupleSize
	return cp.mergedIndex[lengthsStart : lengthsStart+uint64(cp.chunkCount)*lengthSize]
}

func (cp compactionPlan) suffixes() []byte {
	suffixesStart := uint64(cp.chunkCount) * (prefixTupleSize + lengthSize)
	return cp.mergedIndex[suffixesStart : suffixesStart+uint64(cp.chunkCount)*addrSuffixSize]
}

func planCompaction(sources chunkSources) (plan compactionPlan) {
	var totalUncompressedData uint64
	for _, src := range sources {
		plan.chunkCount += src.count()
		totalUncompressedData += src.uncompressedLen()
	}
	plan.sources = sources

	lengthsPos := uint64(plan.chunkCount) * prefixTupleSize
	suffixesPos := lengthsPos + uint64(plan.chunkCount)*lengthSize
	plan.mergedIndex = make([]byte, indexSize(plan.chunkCount)+footerSize)
	plan.chunkDataLens = make([]uint64, len(sources))

	prefixIndexRecs := make(prefixIndexSlice, 0, plan.chunkCount)
	var ordinalOffset uint32
	for i, src := range sources {
		idx := src.index()

		// Add all the prefix tuples from this index to the list of all prefixIndexRecs, modifying the ordinals such that all entries from the 1st item in sources come after those in the 0th and so on.
		for j, prefix := range idx.prefixes {
			rec := prefixIndexRec{prefix: prefix, order: ordinalOffset + idx.ordinals[j]}
			prefixIndexRecs = append(prefixIndexRecs, rec)
		}
		ordinalOffset += src.count()

		// Calculate the amount of chunk data in |src|
		plan.chunkDataLens[i] = idx.offsets[src.count()-1] + uint64(idx.lengths[src.count()-1])
		plan.totalCompressedData += plan.chunkDataLens[i]

		// Bring over the lengths block, in order
		for _, length := range idx.lengths {
			binary.BigEndian.PutUint32(plan.mergedIndex[lengthsPos:], length)
			lengthsPos += lengthSize
		}

		// Bring over the suffixes block, in order
		n := copy(plan.mergedIndex[suffixesPos:], idx.suffixes)
		d.Chk.True(n == len(idx.suffixes))
		suffixesPos += uint64(n)
	}

	// Sort all prefixTuples by hash and then insert them starting at the beginning of plan.mergedIndex
	sort.Sort(prefixIndexRecs)
	var pfxPos uint64
	for _, pi := range prefixIndexRecs {
		binary.BigEndian.PutUint64(plan.mergedIndex[pfxPos:], pi.prefix)
		pfxPos += addrPrefixSize
		binary.BigEndian.PutUint32(plan.mergedIndex[pfxPos:], pi.order)
		pfxPos += ordinalSize
	}

	writeFooter(plan.mergedIndex[uint64(len(plan.mergedIndex))-footerSize:], plan.chunkCount, totalUncompressedData)
	return plan
}

func nameFromSuffixes(suffixes []byte) (name addr) {
	sha := sha512.New()
	sha.Write(suffixes)

	var h []byte
	h = sha.Sum(h) // Appends hash to h
	copy(name[:], h)
	return
}

func compactSourcesToBuffer(sources chunkSources, rl chan struct{}) (name addr, data []byte, chunkCount uint32) {
	d.Chk.True(rl != nil)
	totalData := uint64(0)
	for _, src := range sources {
		chunkCount += src.count()
		totalData += src.uncompressedLen()
	}
	if chunkCount == 0 {
		return
	}

	maxSize := maxTableSize(uint64(chunkCount), totalData)
	buff := make([]byte, maxSize) // This can blow up RAM (BUG 3130)
	tw := newTableWriter(buff, nil)

	// Use "channel of channels" ordered-concurrency pattern so that chunks from a given table stay together, preserving whatever locality was present in that table.
	chunkChans := make(chan chan extractRecord)
	go func() {
		defer close(chunkChans)
		wg := sync.WaitGroup{}
		for _, src := range sources {
			chunks := make(chan extractRecord)
			wg.Add(1)
			go func(s chunkSource, c chan<- extractRecord) {
				defer func() { close(c); wg.Done(); <-rl }()
				defer func() {
					if r := recover(); r != nil {
						c <- extractRecord{a: s.hash(), err: r}
					}
				}()
				rl <- struct{}{}

				s.extract(c)
			}(src, chunks)
			chunkChans <- chunks
		}
		wg.Wait()
	}()

	errString := ""
	known := map[addr]struct{}{}
	for chunks := range chunkChans {
		for rec := range chunks {
			if rec.err != nil {
				errString += fmt.Sprintf("Failed to extract %s:\n %v\n******\n\n", rec.a, rec.err)
				continue
			}
			if _, present := known[rec.a]; !present {
				tw.addChunk(rec.a, rec.data)
				known[rec.a] = struct{}{}
			}
		}
	}

	if errString != "" {
		panic(fmt.Errorf(errString))
	}
	tableSize, name := tw.finish()
	return name, buff[:tableSize], uint32(len(known))
}
