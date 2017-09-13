// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"fmt"
	"math"

	"github.com/attic-labs/noms/go/d"
)

type valueEncoder struct {
	nomsWriter
}

func newValueEncoder(w nomsWriter) *valueEncoder {
	return &valueEncoder{w}
}

func (w *valueEncoder) writeKind(kind NomsKind) {
	w.writeUint8(uint8(kind))
}

func (w *valueEncoder) writeValueSlice(values ValueSlice) {
	count := uint32(len(values))
	w.writeCount(uint64(count))

	for i := uint32(0); i < count; i++ {
		w.writeValue(values[i])
	}
}

type writeToEncoder interface {
	writeTo(*valueEncoder)
}

func (w *valueEncoder) writeCollection(c Collection) {
	c.sequence().(writeToEncoder).writeTo(w)
}

func (w *valueEncoder) writeValue(v Value) {
	k := v.Kind()

	switch k {
	case BlobKind, ListKind, MapKind, SetKind:
		w.writeCollection(v.(Collection))
	case BoolKind:
		w.writeKind(k)
		w.writeBool(bool(v.(Bool)))
	case NumberKind:
		w.writeKind(k)
		n := v.(Number)
		f := float64(n)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			d.Panic("%f is not a supported number", f)
		}
		w.writeNumber(n)
	case RefKind, StructKind:
		v.(writeToEncoder).writeTo(w)
	case StringKind:
		w.writeKind(k)
		w.writeString(string(v.(String)))
	case TypeKind:
		w.writeKind(k)
		v.(*Type).writeTo(w, map[string]*Type{})
	case CycleKind, UnionKind, ValueKind:
		d.Chk.Fail(fmt.Sprintf("A value instance can never have type %s", k))
	default:
		d.Chk.Fail("Unknown NomsKind")
	}
}

func (w *valueEncoder) writeOrderedKey(key orderedKey) {
	v := key.v
	if !key.isOrderedByValue {
		// See https://github.com/attic-labs/noms/issues/1688#issuecomment-227528987
		d.PanicIfTrue(key.h.IsEmpty())
		v = constructRef(key.h, BoolType, 0)
	}
	w.writeValue(v)
}
