// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"io"
	"testing"

	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

// IMPORTANT: These tests and in particular the hash of the values should stay in sync with the
// corresponding tests in js

type countingReader struct {
	last uint32
	val  uint32
	bc   uint8
}

func newCountingReader() *countingReader {
	return &countingReader{0, 0, 4}
}

func (rr *countingReader) next() byte {
	if rr.bc == 0 {
		rr.last = rr.last + 1
		rr.val = rr.last
		rr.bc = 4
	}

	retval := byte(uint64(rr.val) & 0xff)
	rr.bc--
	rr.val = rr.val >> 8
	return retval
}

func (rr *countingReader) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = rr.next()
	}
	return len(p), nil
}

func randomBuff(powOfTwo uint) []byte {
	length := 1 << powOfTwo
	rr := newCountingReader()
	buff := make([]byte, length)
	rr.Read(buff)
	return buff
}

type blobTestSuite struct {
	collectionTestSuite
	buff []byte
}

func newBlobTestSuite(size uint, expectRefStr string, expectChunkCount int, expectPrependChunkDiff int, expectAppendChunkDiff int) *blobTestSuite {
	length := 1 << size
	buff := randomBuff(size)
	blob := NewBlob(bytes.NewReader(buff))
	return &blobTestSuite{
		collectionTestSuite: collectionTestSuite{
			col:                    blob,
			expectType:             BlobType,
			expectLen:              uint64(length),
			expectRef:              expectRefStr,
			expectChunkCount:       expectChunkCount,
			expectPrependChunkDiff: expectPrependChunkDiff,
			expectAppendChunkDiff:  expectAppendChunkDiff,
			validate: func(v2 Collection) bool {
				b2 := v2.(Blob)
				out := make([]byte, length)
				io.ReadFull(b2.Reader(), out)
				return bytes.Compare(out, buff) == 0
			},
			prependOne: func() Collection {
				dup := make([]byte, length+1)
				dup[0] = 0
				copy(dup[1:], buff)
				return NewBlob(bytes.NewReader(dup))
			},
			appendOne: func() Collection {
				dup := make([]byte, length+1)
				copy(dup, buff)
				dup[len(dup)-1] = 0
				return NewBlob(bytes.NewReader(dup))
			},
		},
		buff: buff,
	}
}

func TestBlobSuite4K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(12, "5bkejsg826116lmjv13qb2bcp42bntb8", 0, 0, 0))
}

func TestBlobSuite64K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(16, "k7pjegmih3lf4mp35vt8prk808c4itcs", 2, 2, 2))
}

func TestBlobSuite256K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(18, "9c7kamht186iaomc0pnrojkd8jli6qdf", 8, 2, 2))
}

func TestBlobSuite1M(t *testing.T) {
	suite.Run(t, newBlobTestSuite(20, "qfljcaes345d7g2fi5ta6ujdlok7tjf6", 29, 2, 2))
}

// Checks the first 1/2 of the bytes, then 1/2 of the remainder, then 1/2 of the remainder, etc...
func (suite *blobTestSuite) TestRandomRead() {
	buffReader := bytes.NewReader(suite.buff)
	blobReader := suite.col.(Blob).Reader()

	readByteRange := func(r io.ReadSeeker, start, rel, count int64) []byte {
		bytes := make([]byte, count)
		n, err := r.Seek(start, 0)
		suite.NoError(err)
		suite.Equal(start, n)
		n2, err := r.Seek(rel, 1)
		suite.NoError(err)
		suite.Equal(start+rel, n2)
		n3, err := io.ReadFull(r, bytes)
		suite.NoError(err)
		suite.Equal(int(count), n3)
		return bytes
	}

	readByteRangeFromEnd := func(r io.ReadSeeker, length, offset, count int64) []byte {
		bytes := make([]byte, count)
		n, err := r.Seek(offset, 2)
		suite.NoError(err)
		suite.Equal(length+offset, n)
		n2, err := io.ReadFull(r, bytes)
		suite.NoError(err)
		suite.Equal(int(count), n2)
		return bytes
	}

	checkByteRange := func(start, rel, count int64) {
		expect := readByteRange(buffReader, start, rel, count)
		actual := readByteRange(blobReader, start, rel, count)
		suite.Equal(expect, actual)
	}

	checkByteRangeFromEnd := func(length, offset, count int64) {
		expect := readByteRangeFromEnd(buffReader, length, offset, count)
		actual := readByteRangeFromEnd(blobReader, length, offset, count)
		suite.Equal(expect, actual)
	}

	length := int64(len(suite.buff))
	start := int64(0)
	count := int64(length / 2)
	for count > 2 {
		checkByteRange(start, 0, count)
		checkByteRange(0, start, count)
		checkByteRange(start/2, start-(start/2), count)
		checkByteRangeFromEnd(length, start-length, count)
		start = start + count
		count = (length - start) / 2
	}
}

type testReader struct {
	readCount int
	buf       *bytes.Buffer
}

func (r *testReader) Read(p []byte) (n int, err error) {
	r.readCount++

	switch r.readCount {
	case 1:
		for i := 0; i < len(p); i++ {
			p[i] = 0x01
		}
		io.Copy(r.buf, bytes.NewReader(p))
		return len(p), nil
	case 2:
		p[0] = 0x02
		r.buf.WriteByte(p[0])
		return 1, io.EOF
	default:
		return 0, io.EOF
	}
}

func TestBlobFromReaderThatReturnsDataAndError(t *testing.T) {
	// See issue #264.
	// This tests the case of building a Blob from a reader who returns both data and an error for the final Read() call.
	assert := assert.New(t)
	tr := &testReader{buf: &bytes.Buffer{}}

	b := NewBlob(tr)

	actual := &bytes.Buffer{}
	io.Copy(actual, b.Reader())

	assert.True(bytes.Equal(actual.Bytes(), tr.buf.Bytes()))
	assert.Equal(byte(2), actual.Bytes()[len(actual.Bytes())-1])
}

func TestBlobSplice(t *testing.T) {
	assert := assert.New(t)

	blob := NewEmptyBlob()
	buf := new(bytes.Buffer)

	blob = blob.Splice(0, 0, []byte("I'll do anything"))
	buf.Reset()
	buf.ReadFrom(blob.Reader())
	assert.Equal(buf.String(), "I'll do anything")

	blob = blob.Splice(16, 0, []byte(" for arv"))
	buf.Reset()
	buf.ReadFrom(blob.Reader())
	assert.Equal(buf.String(), "I'll do anything for arv")

	blob = blob.Splice(0, 0, []byte("Yes, "))
	buf.Reset()
	buf.ReadFrom(blob.Reader())
	assert.Equal(buf.String(), "Yes, I'll do anything for arv")

	blob = blob.Splice(5, 20, []byte("it's hard to satisfy"))
	buf.Reset()
	buf.ReadFrom(blob.Reader())
	assert.Equal(buf.String(), "Yes, it's hard to satisfy arv")
}
