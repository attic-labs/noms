package types

import (
	"bytes"
	"io"
	"testing"

	"github.com/attic-labs/noms/ref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	suite.Suite
	blob                   Blob
	buff                   []byte
	expectRef              ref.Ref
	expectChunkCount       int
	expectPrependChunkDiff int
	expectAppendChunkDiff  int
}

func newBlobTestSuite(size uint, expectRefStr string, expectChunkCount int, expectPrependChunkDiff int, expectAppendChunkDiff int) *blobTestSuite {
	buff := randomBuff(size)
	blob := NewBlob(bytes.NewReader(buff))
	return &blobTestSuite{
		blob:                   blob,
		buff:                   buff,
		expectRef:              ref.Parse(expectRefStr),
		expectChunkCount:       expectChunkCount,
		expectPrependChunkDiff: expectPrependChunkDiff,
		expectAppendChunkDiff:  expectAppendChunkDiff,
	}
}

func TestBlobSuite1K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(10, "sha1-a8a3656441ac61edd9cb734559a6c081b5268a6d", 3, 2, 2))
}

func TestBlobSuite4K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(12, "sha1-7fef6ea1ab709b1ea06446976d7fce02be164398", 9, 2, 2))
}

func TestBlobSuite16K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(14, "sha1-16a7098cc077def534932f5a566d0b0ef60283b5", 33, 2, 2))
}

func TestBlobSuite64K(t *testing.T) {
	suite.Run(t, newBlobTestSuite(16, "sha1-a8c6563067edd1c0a49c8b30a2c75cc9d53c0fd3", 4, 2, 2))
}

func TestBlobSuite256K(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	suite.Run(t, newBlobTestSuite(18, "sha1-b924c4a13eb0542766a93f4329f4d13315ef823d", 13, 2, 2))
}

func (suite *blobTestSuite) TestRef() {
	suite.Equal(suite.expectRef.String(), suite.blob.Ref().String())
}

func (suite *blobTestSuite) TestType() {
	suite.True(BlobType.Equals(suite.blob.Type()))
}

func (suite *blobTestSuite) TestLen() {
	suite.Equal(len(suite.buff), int(suite.blob.Len()))
}

func (suite *blobTestSuite) TestEquals() {
	b2 := suite.blob
	suite.True(suite.blob.Equals(b2))
	suite.True(b2.Equals(suite.blob))
}

func (suite *blobTestSuite) TestChunkCount() {
	suite.Equal(suite.expectChunkCount, len(suite.blob.Chunks()))
}

func (suite *blobTestSuite) TestChunkRefType() {
	for _, r := range suite.blob.Chunks() {
		suite.True(RefOfBlobType.Equals(r.Type()))
	}
}

func (suite *blobTestSuite) TestRoundTripAndReadFull() {
	vs := NewTestValueStore()
	r := vs.WriteValue(suite.blob)
	v2 := vs.ReadValue(r.TargetRef()).(Blob)
	suite.True(v2.Equals(suite.blob))
	out := make([]byte, len(suite.buff))
	n, err := io.ReadFull(v2.Reader(), out)
	suite.NoError(err)
	suite.Equal(len(suite.buff), n)
	suite.Equal(suite.buff, out)
}

// Checks the first 1/2 of the bytes, then 1/2 of the remainder, then 1/2 of the remainder, etc...
func (suite *blobTestSuite) TestRandomRead() {
	buffReader := bytes.NewReader(suite.buff)
	blobReader := suite.blob.Reader()

	readByteRange := func(r io.ReadSeeker, start int64, count int64) []byte {
		bytes := make([]byte, count)
		n, err := r.Seek(start, 0)
		suite.NoError(err)
		suite.Equal(start, n)
		n2, err := io.ReadFull(r, bytes)
		suite.NoError(err)
		suite.Equal(int(count), n2)
		return bytes
	}

	checkByteRange := func(start int64, count int64) {
		expect := readByteRange(buffReader, start, count)
		actual := readByteRange(blobReader, start, count)
		suite.Equal(expect, actual)
	}

	length := int64(len(suite.buff))
	start := int64(0)
	count := int64(length / 2)
	for count > 2 {
		checkByteRange(start, count)
		start = start + count
		count = (length - start) / 2
	}
}

func chunkDiffCount(c1 []Ref, c2 []Ref) int {
	count := 0
	refs := make(map[ref.Ref]int)

	for _, r := range c1 {
		refs[r.TargetRef()]++
	}

	for _, r := range c2 {
		if c, ok := refs[r.TargetRef()]; ok {
			if c == 1 {
				delete(refs, r.TargetRef())
			} else {
				refs[r.TargetRef()] = c - 1
			}
		} else {
			count++
		}
	}

	count += len(refs)
	return count
}

func (suite *blobTestSuite) TestPrependChunkDiff() {
	dup := make([]byte, len(suite.buff)+1)
	dup[0] = 0
	copy(dup[1:], suite.buff)
	b2 := NewBlob(bytes.NewReader(dup))
	suite.Equal(suite.expectPrependChunkDiff, chunkDiffCount(suite.blob.Chunks(), b2.Chunks()))
}

func (suite *blobTestSuite) TestAppendChunkDiff() {
	dup := make([]byte, len(suite.buff)+1)
	copy(dup, suite.buff)
	dup[len(dup)-1] = 0
	b2 := NewBlob(bytes.NewReader(dup))
	suite.Equal(suite.expectAppendChunkDiff, chunkDiffCount(suite.blob.Chunks(), b2.Chunks()))
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
