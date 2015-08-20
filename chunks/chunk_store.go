package chunks

import (
	"bytes"
	"hash"
	"io"

	"github.com/attic-labs/noms/d"
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
	// Get gets a reader for the value of the Ref in the store. If the ref is absent from the store nil is returned.
	Get(ref ref.Ref) io.ReadCloser
}

// ChunkSink is a place to put chunks.
type ChunkSink interface {
	Put() ChunkWriter

	// Returns true iff the value at the address |ref| is contained in the source
	Has(ref ref.Ref) bool
}

// ChunkWriter wraps an io.WriteCloser, additionally providing the ability to grab a Ref for all data written through the interface. Calling Ref() or Close() on an instance disallows further writing.
type ChunkWriter interface {
	// Note that the Write(p []byte) (int, error) method of WriterCloser must be retained, but implementations of ChunkWriter should never return an error.
	io.WriteCloser
	Ref() ref.Ref
}

// ChunkWriter wraps an io.WriteCloser, additionally providing the ability to grab a Ref for all data written through the interface. Calling Ref() or Close() on an instance disallows further writing.
type hasFn func(ref ref.Ref) bool
type putFn func(ref ref.Ref, buff *bytes.Buffer)

type chunkWriter struct {
	// Note that the Write(p []byte) (int, error) method of WriterCloser must be retained, but implementations of ChunkWriter should never return an error.
	hfn    hasFn
	pfn    putFn
	buffer *bytes.Buffer
	writer io.Writer
	hash   hash.Hash
	ref    ref.Ref
}

func newChunkWriter(hfn hasFn, pfn putFn) *chunkWriter {
	b := &bytes.Buffer{}
	h := ref.NewHash()
	return &chunkWriter{
		hfn:    hfn,
		pfn:    pfn,
		buffer: b,
		writer: io.MultiWriter(b, h),
		hash:   h,
	}
}

func (w *chunkWriter) Write(data []byte) (int, error) {
	d.Chk.NotNil(w.buffer, "Write() cannot be called after Ref() or Close().")
	size, err := w.writer.Write(data)
	d.Chk.NoError(err)
	return size, nil
}

func (w *chunkWriter) Ref() ref.Ref {
	d.Chk.NoError(w.Close())
	return w.ref
}

func (w *chunkWriter) Close() error {
	if w.buffer == nil {
		return nil
	}

	w.ref = ref.FromHash(w.hash)
	if !w.hfn(w.ref) {
		w.pfn(w.ref, w.buffer)
	}
	w.buffer = nil
	return nil
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
	ldb    ldbStoreFlags
	file   fileStoreFlags
	memory memoryStoreFlags
	nop    nopStoreFlags
}

// CreateStore creates a ChunkStore implementation based on the values of command-line flags.
func (f Flags) CreateStore() (cs ChunkStore) {
	if cs = f.aws.createStore(); cs != nil {
	} else if cs = f.ldb.createStore(); cs != nil {
	} else if cs = f.file.createStore(); cs != nil {
	} else if cs = f.memory.createStore(); cs != nil {
	} else if cs = f.nop.createStore(); cs != nil {
	}
	return cs
}
