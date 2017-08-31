// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"errors"
	"io"
	"sync"

	"runtime"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

// Blob represents a list of Blobs.
type Blob struct {
	seq sequence
	h   *hash.Hash
}

func newBlob(seq sequence) Blob {
	return Blob{seq, &hash.Hash{}}
}

func NewEmptyBlob(vrw ValueReadWriter) Blob {
	return Blob{newBlobLeafSequence(vrw, []byte{}), &hash.Hash{}}
}

func (b Blob) Edit() *BlobEditor {
	return NewBlobEditor(b)
}

// ReaderAt interface. Eagerly loads requested byte-range from the blob p-tree.
func (b Blob) ReadAt(p []byte, off int64) (n int, err error) {
	// TODO: Support negative off?
	d.PanicIfTrue(off < 0)

	startIdx := uint64(off)
	if startIdx >= b.Len() {
		return 0, io.EOF
	}

	endIdx := startIdx + uint64(len(p))
	if endIdx > b.Len() {
		endIdx = b.Len()
	}

	if endIdx == b.Len() {
		err = io.EOF
	}

	if startIdx == endIdx {
		return
	}

	leaves, localStart := loadLeafNodes([]Collection{b}, startIdx, endIdx)
	endIdx = localStart + endIdx - startIdx
	startIdx = localStart

	for _, b := range leaves {
		bl := b.sequence().(blobLeafSequence)

		localEnd := endIdx
		leafLength := uint64(len(bl.data))
		if localEnd > leafLength {
			localEnd = leafLength
		}
		src := bl.data[startIdx:localEnd]

		copy(p[n:], src)
		n += len(src)
		endIdx -= localEnd
		startIdx = 0
	}

	return
}

func (b Blob) Reader() *BlobReader {
	return &BlobReader{b, 0}
}

func (b Blob) Copy(w io.Writer) (n int64) {
	return b.CopyReadAhead(w, 1<<23 /* 8MB */, 6)
}

// CopyReadAhead copies the entire contents of |b| to |w|, and attempts to stay
// |concurrency| |chunkSize| blocks of bytes ahead of the last byte written to
// |w|.
func (b Blob) CopyReadAhead(w io.Writer, chunkSize uint64, concurrency int) (n int64) {
	bChan := make(chan chan []byte, concurrency)

	go func() {
		idx := uint64(0)
		for idx < b.Len() {
			bc := make(chan []byte)
			bChan <- bc

			start := idx
			blockLength := b.Len() - start
			if blockLength > chunkSize {
				blockLength = chunkSize
			}
			idx += blockLength

			go func() {
				buff := make([]byte, blockLength)
				b.ReadAt(buff, int64(start))
				bc <- buff
			}()
		}
		close(bChan)
	}()

	// Ensure read-ahead goroutines can exit
	defer func() {
		for range bChan {
		}
	}()

	for b := range bChan {
		ln, err := w.Write(<-b)
		n += int64(ln)
		if err != nil {
			return
		}
	}
	return
}

// Concat returns a new Blob comprised of this joined with other. It only needs
// to visit the rightmost prolly tree chunks of this Blob, and the leftmost
// prolly tree chunks of other, so it's efficient.
func (b Blob) Concat(other Blob) Blob {
	seq := concat(b.seq, other.seq, func(cur *sequenceCursor, vrw ValueReadWriter) *sequenceChunker {
		return b.newChunker(cur, vrw)
	})
	return newBlob(seq)
}

func (b Blob) newChunker(cur *sequenceCursor, vrw ValueReadWriter) *sequenceChunker {
	return newSequenceChunker(cur, 0, vrw, makeBlobLeafChunkFn(vrw), newIndexedMetaSequenceChunkFn(BlobKind, vrw), hashValueByte)
}

// Collection interface
func (b Blob) Len() uint64 {
	return b.seq.numLeaves()
}

func (b Blob) Empty() bool {
	return b.Len() == 0
}

func (b Blob) sequence() sequence {
	return b.seq
}

func (b Blob) hashPointer() *hash.Hash {
	return b.h
}

// Value interface
func (b Blob) Value() Value {
	return b
}

func (b Blob) Equals(other Value) bool {
	return b.Hash() == other.Hash()
}

func (b Blob) Less(other Value) bool {
	return valueLess(b, other)
}

func (b Blob) Hash() hash.Hash {
	if b.h.IsEmpty() {
		*b.h = getHash(b)
	}

	return *b.h
}

func (b Blob) WalkValues(cb ValueCallback) {
}

func (b Blob) WalkRefs(cb RefCallback) {
	b.seq.WalkRefs(cb)
}

func (b Blob) typeOf() *Type {
	return b.seq.typeOf()
}

func (b Blob) Kind() NomsKind {
	return BlobKind
}

func (b Blob) valueReadWriter() ValueReadWriter {
	return b.seq.valueReadWriter()
}

type BlobReader struct {
	b   Blob
	pos int64
}

func (cbr *BlobReader) Read(p []byte) (n int, err error) {
	n, err = cbr.b.ReadAt(p, cbr.pos)
	cbr.pos += int64(n)
	return
}

func (cbr *BlobReader) Seek(offset int64, whence int) (int64, error) {
	abs := int64(cbr.pos)

	switch whence {
	case 0:
		abs = offset
	case 1:
		abs += offset
	case 2:
		abs = int64(cbr.b.Len()) + offset
	default:
		return 0, errors.New("Blob.Reader.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("Blob.Reader.Seek: negative position")
	}

	cbr.pos = int64(abs)
	return abs, nil
}

func makeBlobLeafChunkFn(vrw ValueReadWriter) makeChunkFn {
	return func(level uint64, items []sequenceItem) (Collection, orderedKey, uint64) {
		d.PanicIfFalse(level == 0)
		buff := make([]byte, len(items))

		for i, v := range items {
			buff[i] = v.(byte)
		}

		return chunkBlobLeaf(vrw, buff)
	}
}

func chunkBlobLeaf(vrw ValueReadWriter, buff []byte) (Collection, orderedKey, uint64) {
	blob := newBlob(newBlobLeafSequence(vrw, buff))
	return blob, orderedKeyFromInt(len(buff)), uint64(len(buff))
}

// NewBlob creates a Blob by reading from every Reader in rs and
// concatenating the result. NewBlob uses one goroutine per Reader.
// If vrw is not nil, chunks are written to vrw instead of kept in memory.
func NewBlob(vrw ValueReadWriter, rs ...io.Reader) Blob {
	return readBlobsP(vrw, rs...)
}

func readBlobsP(vrw ValueReadWriter, rs ...io.Reader) Blob {
	switch len(rs) {
	case 0:
		return NewEmptyBlob(vrw)
	case 1:
		return readBlob(rs[0], vrw)
	}

	blobs := make([]Blob, len(rs))

	wg := &sync.WaitGroup{}
	wg.Add(len(rs))

	for i, r := range rs {
		i2, r2 := i, r
		go func() {
			blobs[i2] = readBlob(r2, vrw)
			wg.Done()
		}()
	}

	wg.Wait()

	b := blobs[0]
	for i := 1; i < len(blobs); i++ {
		b = b.Concat(blobs[i])
	}
	return b
}

func readBlob(r io.Reader, vrw ValueReadWriter) Blob {
	sc := newEmptySequenceChunker(vrw, makeBlobLeafChunkFn(vrw), newIndexedMetaSequenceChunkFn(BlobKind, vrw), func(item sequenceItem, rv *rollingValueHasher) {
		rv.HashByte(item.(byte))
	})

	// TODO: The code below is temporary. It's basically a custom leaf-level chunker for blobs. There are substational perf gains by doing it this way as it avoids the cost of boxing every single byte which is chunked.
	chunkBuff := [8192]byte{}
	chunkBytes := chunkBuff[:]
	rv := newRollingValueHasher(0)
	offset := 0
	addByte := func(b byte) bool {
		if offset >= len(chunkBytes) {
			tmp := make([]byte, len(chunkBytes)*2)
			copy(tmp, chunkBytes)
			chunkBytes = tmp
		}
		chunkBytes[offset] = b
		offset++
		rv.HashByte(b)
		return rv.crossedBoundary
	}

	mtChan := make(chan chan metaTuple, runtime.NumCPU())

	makeChunk := func() {
		rv.Reset()
		cp := make([]byte, offset)
		copy(cp, chunkBytes[0:offset])

		ch := make(chan metaTuple)
		mtChan <- ch

		go func(ch chan metaTuple, cp []byte) {
			col, key, numLeaves := chunkBlobLeaf(vrw, cp)
			ch <- newMetaTuple(vrw.WriteValue(col), key, numLeaves)
		}(ch, cp)

		offset = 0
	}

	go func() {
		readBuff := [8192]byte{}
		for {
			n, err := r.Read(readBuff[:])
			for i := 0; i < n; i++ {
				if addByte(readBuff[i]) {
					makeChunk()
				}
			}
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				if offset > 0 {
					makeChunk()
				}
				close(mtChan)
				break
			}
		}
	}()

	for ch := range mtChan {
		mt := <-ch
		if sc.parent == nil {
			sc.createParent()
		}
		sc.parent.Append(mt)
	}

	return newBlob(sc.Done())
}
