package search

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/walk"
)

type LocalSearcher struct {
	chunks.ChunkStore
}

// Copys all chunks reachable from (and including) |r| but excluding all chunks reachable from (and including) |exclude| in |source| to |sink|.
func (ls LocalSearcher) CopyReachableChunksP(r, exclude ref.Ref, sink chunks.ChunkSink, concurrency int) {
	excludeRefs := map[ref.Ref]bool{}
	hasRef := func(r ref.Ref) bool {
		return excludeRefs[r]
	}

	if exclude != (ref.Ref{}) {
		refChan := make(chan ref.Ref, 1024)
		addRef := func(r ref.Ref) {
			refChan <- r
		}

		go func() {
			walk.AllP(exclude, ls, addRef, concurrency)
			close(refChan)
		}()

		for r := range refChan {
			excludeRefs[r] = true
		}
	}

	tcs := &teeChunkSource{ls, sink}
	walk.SomeP(r, tcs, hasRef, concurrency)
}

// teeChunkSource just serves the purpose of writing to |sink| every chunk that is read from |source|.
type teeChunkSource struct {
	source chunks.ChunkSource
	sink   chunks.ChunkSink
}

func (trs *teeChunkSource) Get(ref ref.Ref) chunks.Chunk {
	c := trs.source.Get(ref)
	if c == nil {
		return nil
	}

	trs.sink.Put(c)
	return c
}

func (trs *teeChunkSource) Has(ref ref.Ref) bool {
	return trs.source.Has(ref)
}
