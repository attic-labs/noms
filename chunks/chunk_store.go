package chunks

import (
	"io"

	"github.com/attic-labs/noms/ref"
)

// ChunkStore is the core storage abstraction in noms. We can put data anyplace we have a ChunkStore implementation for.
type ChunkStore interface {
	ChunkSource
	ChunkSink
	RootTracker
}

// RootTracker allows querying and management of the root of an entire tree of references. The "root" is the single mutable variable in a ChunkStore. It can store any ref, but it is typically used by higher layers (such as DataStore) to store a ref to a value that represents the current state and entire history of a datastore.
type RootTracker interface {
	Root() ref.Ref
	UpdateRoot(current, last ref.Ref) bool
}

// ChunkSource is a place to get chunks from.
type ChunkSource interface {
	// Get gets a reader for the value of the Ref in the store. If the ref is absent from the store nil and no error is returned.
	Get(ref ref.Ref) (io.ReadCloser, error)
}

// ChunkSink is a place to put chunks.
type ChunkSink interface {
	Put() ChunkWriter
}

// ChunkWriter wraps an io.WriteCloser, additionally providing the ability to grab a Ref for all data written through the interface. Calling Ref() or Close() on an instance disallows further writing.
type ChunkWriter interface {
	io.WriteCloser
	// Ref returns the ref.Ref for all data written at the time of call.
	Ref() (ref.Ref, error)
}

// NewFlags creates a new instance of Flags, which declares a number of ChunkStore-related command-line flags using the golang flag package. Call this before flag.Parse().
func NewFlags() Flags {
	return NewFlagsWithPrefix("")
}

// NewFlagsWithPrefix creates a new instance of Flags with the names of all flags declared therein prefixed by the given string.
func NewFlagsWithPrefix(prefix string) Flags {
	return Flags{
		awsFlags(prefix),
		levelDBFlags(prefix),
		fileFlags(prefix),
		memoryFlags(prefix),
		nopFlags(prefix),
	}
}

// Flags abstracts away definitions for and handling of command-line flags for all ChunkStore implementations.
type Flags struct {
	aws    awsStoreFlags
	db     ldbStoreFlags
	file   fileStoreFlags
	memory memoryStoreFlags
	nop    nopStoreFlags
}

// CreateStore creates a ChunkStore implementation based on the values of command-line flags.
func (f Flags) CreateStore() (cs ChunkStore) {
	if cs = f.aws.createStore(); cs != nil {
	} else if cs = f.db.createStore(); cs != nil {
	} else if cs = f.file.createStore(); cs != nil {
	} else if cs = f.memory.createStore(); cs != nil {
	} else if cs = f.nop.createStore(); cs != nil {
	}
	return cs
}
