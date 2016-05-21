package ref

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/attic-labs/noms/d"
)

var (
	// In the future we will allow different digest types, so this will get more complicated. For now sha1 is fine.
	pattern   = regexp.MustCompile("^sha1-([0-9a-f]{40})$")
	emptyHash = Ref{}
)

type Sha1Digest [sha1.Size]byte

type Ref struct {
	// In the future, we will also store the algorithm, and digest will thus probably have to be a slice (because it can vary in size)
	digest Sha1Digest
}

// Digest returns a *copy* of the digest that backs Ref.
func (r Ref) Digest() Sha1Digest {
	return r.digest
}

func (r Ref) IsEmpty() bool {
	return r.digest == emptyHash.digest
}

// DigestSlice returns a slice of the digest that backs A NEW COPY of Ref, because the receiver of this method is not a pointer.
func (r Ref) DigestSlice() []byte {
	return r.digest[:]
}

func (r Ref) String() string {
	return fmt.Sprintf("sha1-%s", hex.EncodeToString(r.digest[:]))
}

func New(digest Sha1Digest) Ref {
	return Ref{digest}
}

func FromData(data []byte) Ref {
	return New(sha1.Sum(data))
}

// FromSlice creates a new Ref backed by data, ensuring that data is an acceptable length.
func FromSlice(data []byte) Ref {
	d.Chk.Len(data, sha1.Size)
	digest := Sha1Digest{}
	copy(digest[:], data)
	return New(digest)
}

func MaybeParse(s string) (r Ref, ok bool) {
	match := pattern.FindStringSubmatch(s)
	if match == nil {
		return
	}

	// TODO: The new temp byte array is kinda bummer. Would be better to walk the string and decode each byte into result.digest. But can't find stdlib functions to do that.
	n, err := hex.Decode(r.digest[:], []byte(match[1]))
	d.Chk.NoError(err) // The regexp above should have validated the input

	// If there was no error, we should have decoded exactly one digest worth of bytes.
	d.Chk.Equal(sha1.Size, n)
	ok = true
	return
}

func Parse(s string) Ref {
	r, ok := MaybeParse(s)
	if !ok {
		d.Exp.Fail(fmt.Sprintf("Cound not parse ref: %s", s))
	}
	return r
}

func (r Ref) Less(other Ref) bool {
	d1, d2 := r.digest, other.digest
	for k := 0; k < len(d1); k++ {
		b1, b2 := d1[k], d2[k]
		if b1 < b2 {
			return true
		} else if b1 > b2 {
			return false
		}
	}

	return false
}

func (r Ref) Greater(other Ref) bool {
	return !r.Less(other) && r != other
}
