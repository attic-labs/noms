// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"math"
	"math/rand"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/golang/snappy"
)

type PullProgress struct {
	DoneCount, KnownCount, ApproxWrittenBytes uint64
}

const bytesWrittenSampleRate = .10

// PullWithFlush calls Pull and then manually flushes data to sinkDB. This is
// an unfortunate current necessity. The Flush() can't happen at the end of
// regular Pull() because that breaks tests that try to ensure we're not
// reading more data from the sinkDB than expected. Flush() triggers
// validation, which triggers sinkDB reads, which means that the code can no
// longer tell which reads were caused by Pull() and which by Flush().
// TODO: Get rid of this (BUG 2982)
func PullWithFlush(srcDB, sinkDB Database, sourceRef types.Ref, progressCh chan PullProgress) {
	Pull(srcDB, sinkDB, sourceRef, progressCh)
	persistChunks(sinkDB.chunkStore())
}

// Pull objects that descend from sourceRef from srcDB to sinkDB.
func Pull(srcDB, sinkDB Database, sourceRef types.Ref, progressCh chan PullProgress) {
	// Sanity Check
	d.PanicIfFalse(srcDB.chunkStore().Has(sourceRef.TargetHash()))

	if sinkDB.chunkStore().Has(sourceRef.TargetHash()) {
		return // already up to date
	}

	var doneCount, knownCount, approxBytesWritten uint64
	updateProgress := func(moreDone, moreKnown, moreApproxBytesWritten uint64) {
		if progressCh == nil {
			return
		}
		doneCount, knownCount, approxBytesWritten = doneCount+moreDone, knownCount+moreKnown, approxBytesWritten+moreApproxBytesWritten
		progressCh <- PullProgress{doneCount, knownCount, approxBytesWritten}
	}
	var sampleSize, sampleCount uint64

	cc := newCompletenessChecker()
	absent := hash.NewHashSet(sourceRef.TargetHash())
	for len(absent) != 0 {
		updateProgress(0, uint64(len(absent)), 0)

		// Concurrently pull all the chunks the sink is missing out of the source
		// For each chunk, put it into the sink, then decode it into a value so we can later iterate all the refs in the chunk.
		absentValues := types.ValueSlice{}
		foundChunks := make(chan *chunks.Chunk)
		go func() { defer close(foundChunks); srcDB.chunkStore().GetMany(absent, foundChunks) }()
		for c := range foundChunks {
			sinkDB.chunkStore().Put(*c)
			absentValues = append(absentValues, types.DecodeValue(*c, srcDB))

			// Randomly sample amount of data written
			if rand.Float64() < bytesWrittenSampleRate {
				sampleSize += uint64(len(snappy.Encode(nil, c.Data())))
				sampleCount++
			}
			updateProgress(1, 0, sampleSize/uint64(math.Max(1, float64(sampleCount))))
		}
		// Descend to the next level of the tree by gathering up the pointers from every chunk we just pulled over to the sink in the loop above
		nextLevel := hash.HashSet{}
		for _, v := range absentValues {
			cc.AddRefs(v)
			v.WalkRefs(func(r types.Ref) {
				nextLevel.Insert(r.TargetHash())
			})
		}

		// Ask sinkDB which of the next level's hashes it doesn't have.
		absent = sinkDB.chunkStore().HasMany(nextLevel)
	}

	cc.PanicIfDangling(sinkDB.chunkStore())
}
