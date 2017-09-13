// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type leafSequence struct {
	vrw     ValueReadWriter
	buff    []byte
	offsets []uint32
}

func newLeafSequence(kind NomsKind, count uint64, vrw ValueReadWriter, vs ...Value) leafSequence {
	d.PanicIfTrue(vrw == nil)
	w := newBinaryNomsWriter()
	enc := newValueEncoder(w)
	enc.writeKind(kind)
	kindPos := w.offset
	enc.writeCount(0) // level
	levelPos := w.offset
	enc.writeCount(count)
	offsets := make([]uint32, len(vs)+3)
	offsets[0] = kindPos
	offsets[1] = levelPos
	offsets[2] = w.offset
	for i, v := range vs {
		enc.writeValue(v)
		offsets[i+3] = w.offset
	}
	return leafSequence{vrw, w.data(), offsets}
}

// readLeafSequence reads the data provided by a decoder and moves the decoder forward.
func readLeafSequence(dec *valueDecoder) leafSequence {
	start := dec.pos()
	offsets := skipLeafSequence(dec)
	end := dec.pos()
	return leafSequence{dec.vrw, dec.byteSlice(start, end), offsets}
}

func skipLeafSequence(dec *valueDecoder) []uint32 {
	dec.skipKind()
	kindPos := dec.pos()
	dec.skipCount() // level
	levelPos := dec.pos()
	count := dec.readCount()
	offsets := make([]uint32, count+3)
	offsets[0] = kindPos
	offsets[1] = levelPos
	offsets[2] = dec.pos()
	for i := uint64(0); i < count; i++ {
		dec.skipValue()
		offsets[i+3] = dec.pos()
	}
	return offsets
}

func (seq leafSequence) decoder() *valueDecoder {
	return newValueDecoder(seq.buff, seq.vrw)
}

func (seq leafSequence) decoderAtOffset(offset int) *valueDecoder {
	return newValueDecoder(seq.buff[offset:], seq.vrw)
}

func (seq leafSequence) decoderSkipToValues() (*valueDecoder, uint64) {
	dec := seq.decoder()
	dec.skipKind()
	dec.skipCount() // level
	count := dec.readCount()
	return dec, count
}

func (seq leafSequence) decoderSkipToIndex(idx int) *valueDecoder {
	offset := seq.getItemOffset(idx)
	return seq.decoderAtOffset(offset)
}

func (seq leafSequence) writeTo(enc *valueEncoder) {
	enc.writeRaw(seq.buff)
}

func (seq leafSequence) values() []Value {
	dec, count := seq.decoderSkipToValues()
	vs := make([]Value, count)
	for i := uint64(0); i < count; i++ {
		vs[i] = dec.readValue()
	}
	return vs
}

func (seq leafSequence) getCompareFnHelper(other leafSequence) compareFn {
	dec := seq.decoder()
	otherDec := other.decoder()

	return func(idx, otherIdx int) bool {
		offset := seq.getItemOffset(idx)
		otherOffset := other.getItemOffset(otherIdx)
		dec.offset = uint32(offset)
		otherDec.offset = uint32(otherOffset)
		return dec.readValue().Equals(otherDec.readValue())
	}
}

func (seq leafSequence) typeOf() *Type {
	dec := seq.decoder()
	kind := dec.readKind()
	dec.skipCount() // level
	count := dec.readCount()
	ts := make([]*Type, count)
	for i := uint64(0); i < count; i++ {
		v := dec.readValue()
		ts[i] = v.typeOf()
	}
	return makeCompoundType(kind, makeCompoundType(UnionKind, ts...))
}

func (seq leafSequence) seqLen() int {
	return int(seq.numLeaves())
}

func (seq leafSequence) numLeaves() uint64 {
	_, count := seq.decoderSkipToValues()
	return count
}

func (seq leafSequence) valueReadWriter() ValueReadWriter {
	return seq.vrw
}

func (seq leafSequence) getChildSequence(idx int) sequence {
	return nil
}

func (seq leafSequence) Kind() NomsKind {
	dec := seq.decoder()
	return dec.readKind()
}

func (seq leafSequence) treeLevel() uint64 {
	return 0
}

func (seq leafSequence) isLeaf() bool {
	return true
}

func (seq leafSequence) cumulativeNumberOfLeaves(idx int) uint64 {
	return uint64(idx) + 1
}

func (seq leafSequence) getCompositeChildSequence(start uint64, length uint64) sequence {
	panic("getCompositeChildSequence called on a leaf sequence")
}

// func (seq *leafSequence) initOffsets() {
// 	dec := seq.decoder()
// 	seq.offsets = skipLeafSequence(dec)
// }

func (seq leafSequence) getItemOffset(idx int) int {
	// kind, level, count, elements....
	//      0      1      2
	if idx+3 > len(seq.offsets) {
		// +1 because the offsets contain one extra offset after the last entry.
		return -1
	}
	return int(seq.offsets[idx+2])
}

func (seq leafSequence) getItem(idx int) sequenceItem {
	// fmt.Printf("getItem dec, %v\n", seq.decoder())
	// fmt.Printf("getItem offset, %d\n", offset)
	dec := seq.decoderSkipToIndex(idx)
	return dec.readValue()
	// dec, count := seq.decoderSkipToValues()
	// if idx >= int(count) {
	// 	return nil
	// }
	// for ; idx > 0; idx-- {
	// 	dec.skipValue()
	// }
	// return dec.readValue()
}

func (seq leafSequence) WalkRefs(cb RefCallback) {
	dec, count := seq.decoderSkipToValues()
	for i := uint64(0); i < count; i++ {
		dec.readValue().WalkRefs(cb)
	}
}

// Collection interface

func (seq leafSequence) Len() uint64 {
	_, count := seq.decoderSkipToValues()
	return count
}

func (seq leafSequence) Empty() bool {
	return seq.Len() == uint64(0)
}

func (seq leafSequence) hash() hash.Hash {
	return hash.Of(seq.buff)
}
