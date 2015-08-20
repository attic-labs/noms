package util

import (
	"bytes"
	"io"
	"os"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

// StdoutChunkSink implements chunks.ChunkSink by writing data to stdout.
type StdoutChunkSink struct {
	chunks.ChunkSink
}

func (f StdoutChunkSink) Has(ref ref.Ref) bool {
	return false
}

func (f StdoutChunkSink) Put(ref ref.Ref, data []byte) {
	io.Copy(os.Stdout, bytes.NewReader(data))
}
