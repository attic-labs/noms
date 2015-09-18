package search

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/http"
	"github.com/attic-labs/noms/ref"
)

type Searcher interface {
	chunks.ChunkStore
	CopyReachableChunksP(r, exclude ref.Ref, sink chunks.ChunkSink, concurrency int)
}

type Flags struct {
	cflags chunks.Flags
	hflags http.Flags
}

func NewFlags() Flags {
	return NewFlagsWithPrefix("")
}

func NewFlagsWithPrefix(prefix string) Flags {
	return Flags{
		chunks.NewFlagsWithPrefix(prefix),
		http.NewFlagsWithPrefix(prefix),
	}
}

func (f Flags) CreateSearcher() (Searcher, bool) {
	cs := f.cflags.CreateStore()
	if cs != nil {
		return LocalSearcher{cs}, true
	}

	ht := f.hflags.CreateClient()
	if ht == nil {
		return LocalSearcher{}, false
	}

	return ht, true
}
