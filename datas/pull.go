package datas

import (
	"fmt"
	"sync"
	"time"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/attic-labs/noms/walk"
)

// CopyMissingChunksP copies to |sink| all chunks in source that are reachable from (and including) |r|, skipping chunks that |sink| already has
func CopyMissingChunksP(source Database, sink *LocalDatabase, sourceRef types.Ref, concurrency int) {
	stopCallback := func(r types.Ref) bool {
		return sink.has(r.TargetRef())
	}
	copyWorker(source, sink, sourceRef, stopCallback, concurrency)
}

// CopyReachableChunksP copies to |sink| all chunks reachable from (and including) |r|, but that are not in the subtree rooted at |exclude|
func CopyReachableChunksP(source, sink Database, sourceRef, exclude types.Ref, concurrency int) {
	excludeRefs := map[ref.Ref]bool{}

	if !exclude.TargetRef().IsEmpty() {
		mu := sync.Mutex{}
		excludeCallback := func(r types.Ref) bool {
			mu.Lock()
			defer mu.Unlock()
			excludeRefs[r.TargetRef()] = true
			return false
		}

		walk.SomeChunksP(exclude, source.batchStore(), excludeCallback, nil, concurrency)
	}

	stopCallback := func(r types.Ref) bool {
		return excludeRefs[r.TargetRef()]
	}
	copyWorker(source, sink, sourceRef, stopCallback, concurrency)
}

func copyWorker(source, sink Database, sourceRef types.Ref, stopCb walk.SomeChunksStopCallback, concurrency int) {
	bs := sink.batchSink()

	var expect, sofar uint64

	walk.SomeChunksP(sourceRef, source.batchStore(), func(r types.Ref) bool {
		sofar++

		// Percent is dubiously useful.
		percent := float32(0)
		if expect > 0 {
			percent = 100 * (float32(sofar) / float32(expect))
		}
		// Printing should be pulled out into the client; or at least it should be opt-in.
		fmt.Printf("\r%d/%d (%.2f%%)", sofar, expect, percent)
		time.Sleep(100 * time.Millisecond) // for testing

		return stopCb(r)
	}, func(r types.Ref, c chunks.Chunk, val types.Value) {
		expect += uint64(len(val.Chunks()))
		bs.SchedulePut(c, r.Height(), types.Hints{})
	}, concurrency)

	// There is an off-by-something error in here which means I have to cheat at the end. Fix.
	fmt.Printf("\r%d/%d (100%%)\n", expect, expect)

	// We should experiment a bit to see if Flush() needs another progress, but it would be separate
	// from the previous walk stage. For push I would expect it to be significant, for pull less so.
	bs.Flush()
}
