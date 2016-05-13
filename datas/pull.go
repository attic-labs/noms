package datas

import (
	"sync"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/attic-labs/noms/walk"
)

// CopyMissingChunksP copies to |sink| all chunks in source that are reachable from (and including) |r|, skipping chunks that |sink| already has
func CopyMissingChunksP(source Database, sink *LocalDatabase, sourceRef types.Ref, concurrency int) {
	copyCallback := func(r types.Ref) (bool, types.Value) {
		return sink.has(r.TargetRef()), nil
	}
	copyWorker(source, sink, sourceRef, copyCallback, concurrency)
}

// CopyReachableChunksP copies to |sink| all chunks reachable from (and including) |r|, but that are not in the subtree rooted at |exclude|
func CopyReachableChunksP(source, sink Database, sourceRef, exclude types.Ref, concurrency int) {
	excludeRefs := map[ref.Ref]bool{}

	if !exclude.TargetRef().IsEmpty() {
		mu := sync.Mutex{}
		excludeCallback := func(r types.Ref) (bool, types.Value) {
			mu.Lock()
			excludeRefs[r.TargetRef()] = true
			mu.Unlock()
			return false, nil
		}

		walk.SomeChunksP(exclude, source, excludeCallback, concurrency)
	}

	copyCallback := func(r types.Ref) (bool, types.Value) {
		return excludeRefs[r.TargetRef()], nil
	}
	copyWorker(source, sink, sourceRef, copyCallback, concurrency)
}

func copyWorker(source Database, sink Database, sourceRef types.Ref, stopFn walk.SomeChunksCallback, concurrency int) {
	bs := sink.batchSink()

	walk.SomeChunksP(sourceRef, source, func(r types.Ref) (bool, types.Value) {
		if stop, _ := stopFn(r); stop {
			return true, nil
		}

		c := source.batchStore().Get(r.TargetRef())
		d.Chk.False(c.IsEmpty())
		bs.SchedulePut(c, types.Hints{})
		return false, types.DecodeChunk(c, source)
	}, concurrency)

	bs.Flush()
}
