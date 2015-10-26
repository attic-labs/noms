package ref

import (
	"testing"

	"crypto/sha1"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/d"
)

func TestParseError(t *testing.T) {
	assert := assert.New(t)

	assertParseError := func(s string) {
		e := d.Try(func() { Parse(s) })
		assert.IsType(d.UsageError{}, e)
	}

	assertParseError("foo")
	assertParseError("sha1")
	assertParseError("sha1-0")

	// too many digits
	assertParseError("sha1-00000000000000000000000000000000000000000")

	// 'g' not valid hex
	assertParseError("sha1-	000000000000000000000000000000000000000g")

	// sha2 not supported
	assertParseError("sha2-0000000000000000000000000000000000000000")

	r := Parse("sha1-0000000000000000000000000000000000000000")
	assert.NotNil(r)
}

func TestEquals(t *testing.T) {
	assert := assert.New(t)

	r0 := Parse("sha1-0000000000000000000000000000000000000000")
	r01 := Parse("sha1-0000000000000000000000000000000000000000")
	r1 := Parse("sha1-0000000000000000000000000000000000000001")

	assert.Equal(r0, r01)
	assert.Equal(r01, r0)
	assert.NotEqual(r0, r1)
	assert.NotEqual(r1, r0)
}

func TestString(t *testing.T) {
	s := "sha1-0123456789abcdef0123456789abcdef01234567"
	r := Parse(s)
	assert.Equal(t, s, r.String())
}

func TestDigest(t *testing.T) {
	r := New(Sha1Digest{})
	d := r.Digest()
	assert.Equal(t, r.Digest(), d)
	// Digest() must return a copy otherwise things get weird.
	d[0] = 0x01
	assert.NotEqual(t, r.Digest(), d)
}

func TestDigestSlice(t *testing.T) {
	r := New(Sha1Digest{})
	d := r.DigestSlice()
	assert.Equal(t, r.DigestSlice(), d)
	// DigestSlice() must return a copy otherwise things get weird.
	d[0] = 0x01
	assert.NotEqual(t, r.DigestSlice(), d)
}

func TestFromHash(t *testing.T) {
	h := sha1.New()
	h.Write([]byte("abc"))
	r := FromHash(h)
	assert.Equal(t, "sha1-a9993e364706816aba3e25717850c26c9cd0d89d", r.String())
}

func TestIsEmpty(t *testing.T) {
	r1 := Ref{}
	assert.True(t, r1.IsEmpty())

	r2 := Parse("sha1-0000000000000000000000000000000000000000")
	assert.True(t, r2.IsEmpty())

	r3 := Parse("sha1-a9993e364706816aba3e25717850c26c9cd0d89d")
	assert.False(t, r3.IsEmpty())
}

func TestLess(t *testing.T) {
	assert := assert.New(t)

	r1 := Parse("sha1-0000000000000000000000000000000000000001")
	r2 := Parse("sha1-0000000000000000000000000000000000000002")

	assert.False(Less(r1, r1))
	assert.True(Less(r1, r2))
	assert.False(Less(r2, r1))
	assert.False(Less(r2, r2))

	r0 := Ref{}
	assert.False(Less(r0, r0))
	assert.True(Less(r0, r2))
	assert.False(Less(r2, r0))
}

func TestGreater(t *testing.T) {
	assert := assert.New(t)

	r1 := Parse("sha1-0000000000000000000000000000000000000001")
	r2 := Parse("sha1-0000000000000000000000000000000000000002")

	assert.False(Greater(r1, r1))
	assert.False(Greater(r1, r2))
	assert.True(Greater(r2, r1))
	assert.False(Greater(r2, r2))

	r0 := Ref{}
	assert.False(Greater(r0, r0))
	assert.False(Greater(r0, r2))
	assert.True(Greater(r2, r0))
}
