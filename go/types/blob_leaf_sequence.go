// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

type blobLeafSequence struct {
	leafSequence
}

func newBlobLeafSequence(vrw ValueReadWriter, data []byte) sequence {
	d.PanicIfTrue(vrw == nil)
	offsets := make([]uint32, leafSequencePartValues+1)
	w := newBinaryNomsWriter()
	enc := newValueEncoder(w)
	offsets[leafSequencePartKind] = 0
	enc.writeKind(BlobKind)
	offsets[leafSequencePartLevel] = w.offset
	enc.writeCount(0) // level
	offsets[leafSequencePartCount] = w.offset
	enc.writeCount(uint64(len(data)))
	offsets[leafSequencePartValues] = w.offset
	enc.writeBytes(data)
	return blobLeafSequence{leafSequence{vrw, w.data(), offsets}}
}

func (bl blobLeafSequence) writeTo(enc *valueEncoder) {
	enc.writeRaw(bl.buff)
}

// sequence interface

func (bl blobLeafSequence) data() []byte {
	offset := bl.offsets[leafSequencePartValues] - bl.offsets[leafSequencePartKind]
	return bl.buff[offset:]
}

func (bl blobLeafSequence) getCompareFn(other sequence) compareFn {
	return func(idx, otherIdx int) bool {
		return bl.getItem(idx) == other.(blobLeafSequence).getItem(otherIdx)
	}
}

func (bl blobLeafSequence) getItem(idx int) sequenceItem {
	offset := bl.offsets[leafSequencePartValues] - bl.offsets[leafSequencePartKind] + uint32(idx)
	return bl.buff[offset]
}

func (bl blobLeafSequence) WalkRefs(cb RefCallback) {
}

func (bl blobLeafSequence) typeOf() *Type {
	return BlobType
}
