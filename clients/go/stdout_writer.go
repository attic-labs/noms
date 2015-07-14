package util

import (
	"hash"
	"io"
	"io/ioutil"
	"os"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/dbg"
	"github.com/attic-labs/noms/ref"
)

type StdoutChunkSink struct {
	chunks.ChunkSink
}

func (f StdoutChunkSink) Put() chunks.ChunkWriter {
	h := ref.NewHash()
	return &stdoutChunkWriter{
		ioutil.NopCloser(nil),
		os.Stdout,
		io.MultiWriter(os.Stdout, h),
		h,
	}
}

type stdoutChunkWriter struct {
	io.Closer
	file   *os.File
	writer io.Writer
	hash   hash.Hash
}

func (w *stdoutChunkWriter) Write(data []byte) (int, error) {
	dbg.Chk.NotNil(w.file, "Write() cannot be called after Ref() or Close().")
	return w.writer.Write(data)
}

func (w *stdoutChunkWriter) Ref() (ref.Ref, error) {
	w.file = nil
	return ref.FromHash(w.hash), nil
}
