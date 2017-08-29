// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"github.com/attic-labs/noms/go/hash"
)

type Ref struct {
	r nomsReader
	h *hash.Hash
}

func NewRef(v Value) Ref {
	// TODO: Taking the hash will duplicate the work of computing the type
	return constructRef(v.Hash(), TypeOf(v), maxChunkHeight(v)+1)
}

// ToRefOfValue returns a new Ref that points to the same target as |r|, but
// with the type 'Ref<Value>'.
func ToRefOfValue(r Ref) Ref {
	return constructRef(r.TargetHash(), ValueType, r.Height())
}

func constructRef(targetHash hash.Hash, targetType *Type, height uint64) Ref {
	w := newBinaryNomsWriter()
	enc := newValueEncoder(w)
	writeRefPartsToEncoder(enc, targetHash, targetType, height)
	return Ref{w.reader(), &hash.Hash{}}
}

// readRef reads the data provided by a decoder and moves the decoder forward.
func readRef(dec *valueDecoder) Ref {
	start := dec.pos()
	skipRef(dec)
	end := dec.pos()
	return Ref{dec.slice(start, end), &hash.Hash{}}
}

// readRef reads the data provided by a decoder and moves the decoder forward.
func skipRef(dec *valueDecoder) {
	dec.skipKind()
	dec.skipHash()  // targetHash
	dec.skipType()  // targetType
	dec.skipCount() // height
}

func (r Ref) writeTo(enc *valueEncoder) {
	// The NomsKind has already been written.
	if enc.canWriteRaw(r.r) {
		enc.writeRaw(r.r)
	} else {
		writeRefPartsToEncoder(enc, r.TargetHash(), r.TargetType(), r.Height())
	}
}

func writeRefPartsToEncoder(enc *valueEncoder, targetHash hash.Hash, targetType *Type, height uint64) {
	enc.writeKind(RefKind)
	enc.writeHash(targetHash)
	enc.writeType(targetType, map[string]*Type{})
	enc.writeCount(height)
}

func maxChunkHeight(v Value) (max uint64) {
	v.WalkRefs(func(r Ref) {
		if height := r.Height(); height > max {
			max = height
		}
	})
	return
}

func (r Ref) decoder() *valueDecoder {
	return newValueDecoder(r.r.clone(), nil)
}

func (r Ref) TargetHash() hash.Hash {
	dec := r.decoder()
	dec.skipKind()
	return dec.readHash()
}

func (r Ref) Height() uint64 {
	dec := r.decoder()
	dec.skipKind()
	dec.skipHash()
	dec.skipType()
	return dec.readCount()
}

func (r Ref) TargetValue(vr ValueReader) Value {
	return vr.ReadValue(r.TargetHash())
}

func (r Ref) TargetType() *Type {
	dec := r.decoder()
	dec.skipKind()
	dec.skipHash()
	return dec.readType()
}

// Value interface
func (r Ref) Value() Value {
	return r
}

func (r Ref) Equals(other Value) bool {
	return r.Hash() == other.Hash()
}

func (r Ref) Less(other Value) bool {
	return valueLess(r, other)
}

func (r Ref) Hash() hash.Hash {
	if r.h.IsEmpty() {
		*r.h = getHash(r)
	}

	return *r.h
}

func (r Ref) WalkValues(cb ValueCallback) {
}

func (r Ref) WalkRefs(cb RefCallback) {
	cb(r)
}

func (r Ref) typeOf() *Type {
	return makeCompoundType(RefKind, r.TargetType())
}

func (r Ref) Kind() NomsKind {
	return RefKind
}
