package search

import (
	"flag"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

type Searcher interface {
	chunks.ChunkStore
	CopyReachableChunksP(r, exclude ref.Ref, sink chunks.ChunkSink, concurrency int)
}

type Flags struct {
	cflags chunks.Flags
	host   *string
}

func NewFlags() Flags {
	return NewFlagsWithPrefix("")
}

func NewFlagsWithPrefix(prefix string) Flags {
	return Flags{
		chunks.NewFlagsWithPrefix(prefix),
		flag.String(prefix+"h", "", "http host to connect to"),
	}
}

func (f Flags) CreateSearcher() (Searcher, bool) {
	cs := f.cflags.CreateStore()
	if cs != nil {
		return LocalSearcher{cs}, true
	}

	if *f.host == "" {
		return LocalSearcher{}, false
	}

	return NewRemoteSearcher(*f.host), true
}
