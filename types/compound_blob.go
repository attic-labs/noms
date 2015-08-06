package types

import (
	"errors"
	"io"
	"sort"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

// compoundBlob represents a list of Blobs.
// It implements the Blob interface.
type compoundBlob struct {
	length  uint64
	offsets []uint64
	blobs   []Future
	ref     *ref.Ref
	cs      chunks.ChunkSource
}

// Reader implements the Blob interface
func (cb compoundBlob) Reader() io.ReadSeeker {
	return &compoundBlobReader{cb: cb}
}

type compoundBlobReader struct {
	cb               compoundBlob
	currentReader    io.ReadSeeker
	currentBlobIndex int
	offset           int64
}

func (cbr *compoundBlobReader) Read(p []byte) (n int, err error) {
	for cbr.currentBlobIndex < len(cbr.cb.blobs) {
		if cbr.currentReader == nil {
			if err = cbr.updateReader(); err != nil {
				return
			}
		}

		n, err = cbr.currentReader.Read(p)
		if n > 0 || err != io.EOF {
			if err == io.EOF {
				err = nil
			}
			cbr.offset += int64(n)
			return
		}

		cbr.currentBlobIndex++
		cbr.currentReader = nil
	}
	return 0, io.EOF
}

func (cbr *compoundBlobReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case 0:
		abs = offset
	case 1:
		abs = int64(cbr.offset) + offset
	case 2:
		abs = int64(cbr.cb.Len()) + offset
	default:
		return 0, errors.New("Blob.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Blob.Reader.Seek: negative position")
	}

	cbr.offset = abs
	currentBlobIndex := cbr.currentBlobIndex
	cbr.currentBlobIndex = cbr.findBlobOffset(uint64(abs))
	if currentBlobIndex != cbr.currentBlobIndex {
		if err := cbr.updateReader(); err != nil {
			return int64(0), err
		}
	}
	if cbr.currentReader != nil {
		offset := abs - int64(cbr.cb.offsets[cbr.currentBlobIndex])
		if _, err := cbr.currentReader.Seek(offset, 0); err != nil {
			return 0, err
		}
	}

	return abs, nil
}

func (cbr *compoundBlobReader) findBlobOffset(abs uint64) int {
	// TODO(arv): The -1 at the end is bad. If the offsets was shifted one to the right things would be cleaner.
	if abs >= cbr.cb.Len() {
		return len(cbr.cb.blobs)
	}
	return sort.Search(len(cbr.cb.offsets), func(i int) bool {
		return cbr.cb.offsets[i] > abs
	}) - 1
}

func (cbr *compoundBlobReader) updateReader() error {
	if cbr.currentBlobIndex < len(cbr.cb.blobs) {
		v, err := cbr.cb.blobs[cbr.currentBlobIndex].Deref(cbr.cb.cs)
		if err != nil {
			return err
		}
		cbr.currentReader = v.(Blob).Reader()
	} else {
		cbr.currentReader = nil
	}
	return nil
}

// Len implements the Blob interface
func (cb compoundBlob) Len() uint64 {
	return cb.length
}

func (cb compoundBlob) Ref() ref.Ref {
	return ensureRef(cb.ref, cb)
}

func (cb compoundBlob) Equals(other Value) bool {
	if other == nil {
		return false
	} else {
		return cb.Ref() == other.Ref()
	}
}

func (cb compoundBlob) Chunks() (futures []Future) {
	for _, f := range cb.blobs {
		if f, ok := f.(*unresolvedFuture); ok {
			futures = append(futures, f)
		}
	}
	return
}
