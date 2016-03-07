package chunks

import (
	"fmt"
	"io"

	"github.com/attic-labs/noms/ref"
)

// ChunkStore is the core storage abstraction in noms. We can put data anyplace we have a ChunkStore implementation for.
type ChunkStore interface {
	ChunkSource
	ChunkSink
	RootTracker
}

// Factory allows the creation of namespaced ChunkStore instances. The details of how namespaces are separated is left up to the particular implementation of Factory and ChunkStore.
type Factory interface {
	CreateStore(ns string) ChunkStore

	// Shutter shuts down the factory. Subsequent calls to CreateStore() will fail.
	Shutter()
}

// RootTracker allows querying and management of the root of an entire tree of references. The "root" is the single mutable variable in a ChunkStore. It can store any ref, but it is typically used by higher layers (such as DataStore) to store a ref to a value that represents the current state and entire history of a datastore.
type RootTracker interface {
	Root() ref.Ref
	UpdateRoot(current, last ref.Ref) bool
}

// ChunkSource is a place to get chunks from.
type ChunkSource interface {
	// Get gets a reader for the value of the Ref in the store. If the ref is absent from the store nil is returned.
	Get(ref ref.Ref) Chunk

	// Returns true iff the value at the address |ref| is contained in the source
	Has(ref ref.Ref) bool
}

// ChunkSink is a place to put chunks.
type ChunkSink interface {
	// Put writes c into the ChunkSink. c may or may not be persisted when Put() returns.
	Put(c Chunk)

	// PutMany tries to write chunks into the sink. It will handle as many as possible, then return a BackpressureError containing the rest (if any). Handled chunks may or may not be persisted when PutMany() returns.
	PutMany(chunks ...Chunk) BackpressureError

	io.Closer
}

// BackpressureError is a slice of Chunk that indicates some chunks could not be Put(). Caller is free to try to Put them again later.
type BackpressureError []Chunk

func (b BackpressureError) Error() string {
	return fmt.Sprintf("Tried to Put %d too many Chunks", len(b))
}
