// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"fmt"

	"github.com/attic-labs/noms/go/d"
)

type valueDecoder struct {
	nomsReader
	vrw        ValueReadWriter
	validating bool
}

func newValueDecoder(nr nomsReader, vrw ValueReadWriter) *valueDecoder {
	return &valueDecoder{nr, vrw, false}
}

func newValueDecoderWithValidation(nr nomsReader, vrw ValueReadWriter) *valueDecoder {
	return &valueDecoder{nr, vrw, true}
}

func (r *valueDecoder) copyString(w nomsWriter) {
	if w.canWriteRaw(r) {
		start := r.pos()
		r.skipString()
		end := r.pos()
		w.writeRaw(r.slice(start, end))
	} else {
		w.writeString(r.readString())
	}
}

func (r *valueDecoder) peekKind() NomsKind {
	return NomsKind(r.peekUint8())
}

func (r *valueDecoder) readKind() NomsKind {
	return NomsKind(r.readUint8())
}

func (r *valueDecoder) skipKind() {
	r.skipUint8()
}

func (r *valueDecoder) readRef() Ref {
	return readRef(r)
}

func (r *valueDecoder) skipRef() {
	skipRef(r)
}

func (r *valueDecoder) readType() *Type {
	t := r.readTypeInner(map[string]*Type{})
	if r.validating {
		validateType(t)
	}
	return t
}

func (r *valueDecoder) skipType() {
	if r.validating {
		r.readType()
		return
	}
	r.skipTypeInner()
}

func (r *valueDecoder) readTypeInner(seenStructs map[string]*Type) *Type {
	k := r.readKind()
	switch k {
	case ListKind:
		return makeCompoundType(ListKind, r.readTypeInner(seenStructs))
	case MapKind:
		return makeCompoundType(MapKind, r.readTypeInner(seenStructs), r.readTypeInner(seenStructs))
	case RefKind:
		return makeCompoundType(RefKind, r.readTypeInner(seenStructs))
	case SetKind:
		return makeCompoundType(SetKind, r.readTypeInner(seenStructs))
	case StructKind:
		return r.readStructType(seenStructs)
	case UnionKind:
		return r.readUnionType(seenStructs)
	case CycleKind:
		name := r.readString()
		d.PanicIfTrue(name == "") // cycles to anonymous structs are disallowed
		t, ok := seenStructs[name]
		d.PanicIfFalse(ok)
		return t
	}

	d.PanicIfFalse(IsPrimitiveKind(k))
	return MakePrimitiveType(k)
}

func (r *valueDecoder) skipTypeInner() {
	k := r.readKind()
	switch k {
	case ListKind, RefKind, SetKind:
		r.skipTypeInner()
	case MapKind:
		r.skipTypeInner()
		r.skipTypeInner()
	case StructKind:
		r.skipStructType()
	case UnionKind:
		r.skipUnionType()
	case CycleKind:
		r.skipString()
	default:
		d.PanicIfFalse(IsPrimitiveKind(k))
	}
}

func (r *valueDecoder) readBlobLeafSequence() sequence {
	b := r.readBytes()
	return newBlobLeafSequence(r.vrw, b)
}

func (r *valueDecoder) skipBlobLeafSequence() {
	r.skipBytes()
}

func (r *valueDecoder) readValueSequence() ValueSlice {
	count := uint32(r.readCount())

	data := ValueSlice{}
	for i := uint32(0); i < count; i++ {
		v := r.readValue()
		data = append(data, v)
	}

	return data
}

func (r *valueDecoder) skipValueSequence() {
	count := r.readCount()
	for i := uint64(0); i < count; i++ {
		r.skipValue()
	}
}

func (r *valueDecoder) readListLeafSequence() sequence {
	data := r.readValueSequence()
	return listLeafSequence{leafSequence{r.vrw, len(data), ListKind}, data}
}

func (r *valueDecoder) skipListLeafSequence() {
	r.skipValueSequence()
}

func (r *valueDecoder) readSetLeafSequence() orderedSequence {
	data := r.readValueSequence()
	return setLeafSequence{leafSequence{r.vrw, len(data), SetKind}, data}
}

func (r *valueDecoder) skipSetLeafSequence() {
	r.skipValueSequence()
}

func (r *valueDecoder) readMapLeafSequence() orderedSequence {
	count := r.readCount()
	data := []mapEntry{}
	for i := uint64(0); i < count; i++ {
		k := r.readValue()
		v := r.readValue()
		data = append(data, mapEntry{k, v})
	}

	return mapLeafSequence{leafSequence{r.vrw, len(data), MapKind}, data}
}

func (r *valueDecoder) skipMapLeafSequence() {
	count := r.readCount()
	for i := uint64(0); i < count; i++ {
		r.skipValue() // k
		r.skipValue() // v
	}
}

func (r *valueDecoder) readMetaSequence(k NomsKind, level uint64) metaSequence {
	count := r.readCount()

	data := []metaTuple{}
	for i := uint64(0); i < count; i++ {
		ref := r.readValue().(Ref)
		v := r.readValue()
		var key orderedKey
		if r, ok := v.(Ref); ok {
			// See https://github.com/attic-labs/noms/issues/1688#issuecomment-227528987
			key = orderedKeyFromHash(r.TargetHash())
		} else {
			key = newOrderedKey(v)
		}
		numLeaves := r.readCount()
		data = append(data, newMetaTuple(ref, key, numLeaves))
	}

	return newMetaSequence(k, level, data, r.vrw)
}

func (r *valueDecoder) skipMetaSequence(k NomsKind, level uint64) {
	count := r.readCount()
	for i := uint64(0); i < count; i++ {
		r.skipValue() // ref
		r.skipValue() // v
		r.skipCount() // numLeaves
	}
}

func (r *valueDecoder) readValue() Value {
	k := r.peekKind()
	switch k {
	case BlobKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			return newBlob(r.readMetaSequence(k, level))
		}

		return newBlob(r.readBlobLeafSequence())
	case BoolKind:
		r.skipKind()
		return Bool(r.readBool())
	case NumberKind:
		r.skipKind()
		return r.readNumber()
	case StringKind:
		r.skipKind()
		return String(r.readString())
	case ListKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			return newList(r.readMetaSequence(k, level))
		}

		return newList(r.readListLeafSequence())
	case MapKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			return newMap(r.readMetaSequence(k, level))
		}

		return newMap(r.readMapLeafSequence())
	case RefKind:
		return r.readRef()
	case SetKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			return newSet(r.readMetaSequence(k, level))
		}

		return newSet(r.readSetLeafSequence())
	case StructKind:
		return r.readStruct()
	case TypeKind:
		r.skipKind()
		return r.readType()
	case CycleKind, UnionKind, ValueKind:
		d.Chk.Fail(fmt.Sprintf("A value instance can never have type %s", k))
	}

	panic("not reachable")
}

func (r *valueDecoder) skipValue() {
	k := r.peekKind()
	switch k {
	case BlobKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			r.skipMetaSequence(k, level)
		} else {
			r.skipBlobLeafSequence()
		}
	case BoolKind:
		r.skipKind()
		r.skipBool()
	case NumberKind:
		r.skipKind()
		r.skipNumber()
	case StringKind:
		r.skipKind()
		r.skipString()
	case ListKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			r.skipMetaSequence(k, level)
		} else {
			r.skipListLeafSequence()
		}
	case MapKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			r.skipMetaSequence(k, level)
		} else {
			r.skipMapLeafSequence()
		}
	case RefKind:
		r.skipRef()
	case SetKind:
		r.skipKind()
		level := r.readCount()
		if level > 0 {
			r.skipMetaSequence(k, level)
		} else {
			r.skipSetLeafSequence()
		}
	case StructKind:
		r.skipStruct()
	case TypeKind:
		r.skipKind()
		r.skipType()
	case CycleKind, UnionKind, ValueKind:
		d.Chk.Fail(fmt.Sprintf("A value instance can never have type %s", k))
	default:
		panic("not reachable")
	}
}

func (r *valueDecoder) copyValue(enc *valueEncoder) {
	if enc.canWriteRaw(r) {
		start := r.pos()
		r.skipValue()
		end := r.pos()
		enc.writeRaw(r.slice(start, end))
	} else {
		enc.writeValue(r.readValue())
	}
}

func (r *valueDecoder) readStruct() Value {
	return readStruct(r)
}

func (r *valueDecoder) skipStruct() {
	skipStruct(r)
}

func boolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func (r *valueDecoder) readStructType(seenStructs map[string]*Type) *Type {
	name := r.readString()
	count := r.readCount()
	fields := make(structTypeFields, count)

	t := newType(StructDesc{name, fields})
	seenStructs[name] = t

	for i := uint64(0); i < count; i++ {
		t.Desc.(StructDesc).fields[i] = StructField{
			Name: r.readString(),
		}
	}
	for i := uint64(0); i < count; i++ {
		t.Desc.(StructDesc).fields[i].Type = r.readTypeInner(seenStructs)
	}
	for i := uint64(0); i < count; i++ {
		t.Desc.(StructDesc).fields[i].Optional = r.readBool()
	}

	return t
}

func (r *valueDecoder) skipStructType() {
	r.skipString() // name
	count := r.readCount()

	for i := uint64(0); i < count; i++ {
		r.skipString() // name
	}
	for i := uint64(0); i < count; i++ {
		r.skipTypeInner()
	}
	for i := uint64(0); i < count; i++ {
		r.skipBool() // optional
	}
}

func (r *valueDecoder) readUnionType(seenStructs map[string]*Type) *Type {
	l := r.readCount()
	ts := make(typeSlice, l)
	for i := uint64(0); i < l; i++ {
		ts[i] = r.readTypeInner(seenStructs)
	}
	return makeCompoundType(UnionKind, ts...)
}

func (r *valueDecoder) skipUnionType() {
	l := r.readCount()
	for i := uint64(0); i < l; i++ {
		r.skipTypeInner()
	}
}
