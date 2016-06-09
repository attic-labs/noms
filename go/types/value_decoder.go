// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"fmt"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type valueDecoder struct {
	nomsReader
	vr ValueReader
}

func newValueDecoder(nr nomsReader, vr ValueReader) *valueDecoder {
	return &valueDecoder{nr, vr}
}

func (r *valueDecoder) readKind() NomsKind {
	return NomsKind(r.readUint8())
}

func (r *valueDecoder) readRef(t *Type) Ref {
	h := r.readHash()
	height := r.readUint64()
	return constructRef(t, h, height)
}

func (r *valueDecoder) scanType() {
	k := r.readKind()
	switch k {
	case ListKind:
		r.scanType() // elem
	case MapKind:
		r.scanType() // key
		r.scanType() // value
	case RefKind:
		r.scanType() // referenced
	case SetKind:
		r.scanType() // value
	case StructKind:
		r.scanStructType()
	case UnionKind:
		count := r.readUint32()
		for i := uint32(0); i < count; i++ {
			r.scanType()
		}
	case CycleKind:
		r.readUint32()
	}
}

var tc = newTypeCache(1024, 1024)

func (r *valueDecoder) readType(parentStructTypes []*Type) (t *Type) {
	// Scan forward to lookup type
	startIdx := r.pos()
	r.scanType()
	typeSlice := r.sliceFrom(startIdx)
	if t, ok := tc.get(typeSlice); ok {
		return t // Cache hit
	}

	// Cache miss
	r.seek(startIdx)

	k := r.readKind()
	switch k {
	case ListKind:
		t = MakeListType(r.readType(parentStructTypes))
	case MapKind:
		t = MakeMapType(r.readType(parentStructTypes), r.readType(parentStructTypes))
	case RefKind:
		t = MakeRefType(r.readType(parentStructTypes))
	case SetKind:
		t = MakeSetType(r.readType(parentStructTypes))
	case StructKind:
		t = r.readStructType(parentStructTypes)
	case UnionKind:
		count := r.readUint32()
		elemTypes := make([]*Type, count)
		for i := uint32(0); i < count; i++ {
			elemTypes[i] = r.readType(parentStructTypes)
		}
		t = MakeUnionType(elemTypes...)
	case CycleKind:
		i := r.readUint32()
		d.Chk.True(i < uint32(len(parentStructTypes)))
		t = parentStructTypes[len(parentStructTypes)-1-int(i)]
	default:
		d.Chk.True(IsPrimitiveKind(k))
		t = MakePrimitiveType(k)
	}

	// Insert in cache
	tc.set(typeSlice, t)
	return t
}

func (r *valueDecoder) readBlobLeafSequence() indexedSequence {
	b := r.readBytes()
	return newBlobLeafSequence(r.vr, b)
}

func (r *valueDecoder) readValueSequence() ValueSlice {
	count := r.readUint32()

	data := ValueSlice{}
	for i := uint32(0); i < count; i++ {
		v := r.readValue()
		data = append(data, v)
	}

	return data
}

func (r *valueDecoder) readListLeafSequence(t *Type) indexedSequence {
	data := r.readValueSequence()
	return listLeafSequence{data, t, r.vr}
}

func (r *valueDecoder) readSetLeafSequence(t *Type) orderedSequence {
	data := r.readValueSequence()
	return setLeafSequence{data, t, r.vr}
}

func (r *valueDecoder) readMapLeafSequence(t *Type) orderedSequence {
	count := r.readUint32()
	data := []mapEntry{}
	for i := uint32(0); i < count; i++ {
		k := r.readValue()
		v := r.readValue()
		data = append(data, mapEntry{k, v})
	}

	return mapLeafSequence{data, t, r.vr}
}

func (r *valueDecoder) readMetaSequence() metaSequenceData {
	count := r.readUint32()

	data := metaSequenceData{}
	for i := uint32(0); i < count; i++ {
		ref := r.readValue().(Ref)
		value := r.readValue()
		numLeaves := r.readUint64()
		data = append(data, newMetaTuple(ref, value, numLeaves, nil))
	}

	return data
}

func (r *valueDecoder) readIndexedMetaSequence(t *Type) indexedMetaSequence {
	return newIndexedMetaSequence(r.readMetaSequence(), t, r.vr)
}

func (r *valueDecoder) readOrderedMetaSequence(t *Type) orderedMetaSequence {
	return newOrderedMetaSequence(r.readMetaSequence(), t, r.vr)
}

func (r *valueDecoder) readValue() Value {
	t := r.readType(nil)
	switch t.Kind() {
	case BlobKind:
		isMeta := r.readBool()
		if isMeta {
			return newBlob(r.readIndexedMetaSequence(t))
		}

		return newBlob(r.readBlobLeafSequence())
	case BoolKind:
		return Bool(r.readBool())
	case NumberKind:
		return Number(r.readFloat64())
	case StringKind:
		return NewString(r.readString())
	case ListKind:
		isMeta := r.readBool()
		if isMeta {
			return newList(r.readIndexedMetaSequence(t))
		}

		return newList(r.readListLeafSequence(t))
	case MapKind:
		isMeta := r.readBool()
		if isMeta {
			return newMap(r.readOrderedMetaSequence(t))
		}

		return newMap(r.readMapLeafSequence(t))
	case RefKind:
		return r.readRef(t)
	case SetKind:
		isMeta := r.readBool()
		if isMeta {
			return newSet(r.readOrderedMetaSequence(t))
		}

		return newSet(r.readSetLeafSequence(t))
	case StructKind:
		return r.readStruct(t)
	case TypeKind:
		return r.readType(nil)
	case CycleKind, UnionKind, ValueKind:
		d.Chk.Fail(fmt.Sprintf("A value instance can never have type %s", KindToString[t.Kind()]))
	}

	panic("not reachable")
}

func (r *valueDecoder) readStruct(t *Type) Value {
	// We've read `[StructKind, name, fields, unions` at this point
	desc := t.Desc.(StructDesc)
	count := desc.Len()
	values := make([]Value, count)
	for i := 0; i < count; i++ {
		values[i] = r.readValue()
	}

	return Struct{values, t, &hash.Hash{}}
}

func (r *valueDecoder) scanStructType() {
	r.scanString() // struct name
	count := r.readUint32()

	for i := uint32(0); i < count; i++ {
		r.scanString() // field name
		r.scanType()
	}
}

func (r *valueDecoder) readStructType(parentStructTypes []*Type) *Type {
	name := r.readString()
	count := r.readUint32()

	fields := make(fieldSlice, count)
	desc := StructDesc{name, fields}
	st := buildType(desc)
	parentStructTypes = append(parentStructTypes, st)

	for i := uint32(0); i < count; i++ {
		fields[i] = field{name: r.readString(), t: r.readType(parentStructTypes)}
	}
	st.Desc = desc
	return st
}
