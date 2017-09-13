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
	offsets := make([]uint32, 4)
	w := newBinaryNomsWriter()
	enc := newValueEncoder(w)
	offsets[0] = 0
	enc.writeKind(BlobKind)
	offsets[1] = w.offset
	enc.writeCount(0) // level
	offsets[2] = w.offset
	enc.writeBytes(data)
	offsets[3] = w.offset
	return blobLeafSequence{leafSequence{vrw, w.data(), offsets}}
}

func (bl blobLeafSequence) writeTo(enc *valueEncoder) {
	enc.writeRaw(bl.buff)
}

// sequence interface

func (bl blobLeafSequence) data() []byte {
	dec := bl.decoder()
	dec.skipKind()
	dec.skipCount() // level
	return dec.readBytes()
}

func (bl blobLeafSequence) getCompareFn(other sequence) compareFn {
	data := bl.data()
	otherData := other.(blobLeafSequence).data()
	return func(idx, otherIdx int) bool {
		return data[idx] == otherData[otherIdx]
	}
}

func (bl blobLeafSequence) getItem(idx int) sequenceItem {

	return bl.data()[idx]
}

func (bl blobLeafSequence) WalkRefs(cb RefCallback) {
}

func (bl blobLeafSequence) typeOf() *Type {
	return BlobType
}
