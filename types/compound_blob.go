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
	return &compoundBlobReader{length: cb.length, blobs: cb.blobs, offsets: cb.offsets, cs: cb.cs}
}

type compoundBlobReader struct {
	length              uint64
	blobs               []Future
	offsets             []uint64
	currentReader       io.ReadSeeker
	currentReaderIndex  int
	currentReaderOffset int64
	offset              int64
	currentBlobIndex    int
	cs                  chunks.ChunkSource
}

func (cbr *compoundBlobReader) Read(p []byte) (n int, err error) {
	for cbr.currentBlobIndex < len(cbr.blobs) {
		if cbr.currentReader == nil || cbr.currentBlobIndex != cbr.currentReaderIndex {
			if err = cbr.updateReader(); err != nil {
				return
			}
		}
		if err = cbr.seekIfNeeded(); err != nil {
			return
		}

		n, err = cbr.currentReader.Read(p)
		if n > 0 || err != io.EOF {
			if err == io.EOF {
				err = nil
			}
			cbr.offset += int64(n)
			cbr.currentReaderOffset += int64(n)
			return
		}

		cbr.currentBlobIndex++
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
		abs = int64(cbr.length) + offset
	default:
		return 0, errors.New("Blob.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Blob.Reader.Seek: negative position")
	}

	cbr.offset = abs
	cbr.currentBlobIndex = cbr.findBlobOffset(uint64(abs))
	return abs, nil
}

func (cbr *compoundBlobReader) findBlobOffset(abs uint64) int {
	return sort.Search(len(cbr.offsets), func(i int) bool {
		return cbr.offsets[i] > abs
	}) - 1
}

func (cbr *compoundBlobReader) seekIfNeeded() error {
	offset := cbr.offset - int64(cbr.offsets[cbr.currentBlobIndex])
	if offset != cbr.currentReaderOffset {
		n, err := cbr.currentReader.Seek(offset, 0)
		if err != nil {
			return err
		}
		cbr.currentReaderOffset += n
	}

	return nil
}

func (cbr *compoundBlobReader) updateReader() error {
	v, err := cbr.blobs[cbr.currentBlobIndex].Deref(cbr.cs)
	if err != nil {
		return err
	}
	cbr.currentReader = v.(Blob).Reader()
	cbr.currentReaderIndex = cbr.currentBlobIndex

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
