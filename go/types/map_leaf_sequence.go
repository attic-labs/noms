// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

type mapLeafSequence struct {
	leafSequence
}

type mapEntry struct {
	key   Value
	value Value
}

func (entry mapEntry) writeTo(w *valueEncoder) {
	w.writeValue(entry.key)
	w.writeValue(entry.value)
}

func readMapEntry(r *valueDecoder) mapEntry {
	return mapEntry{r.readValue(), r.readValue()}
}

func (entry mapEntry) equals(other mapEntry) bool {
	return entry.key.Equals(other.key) && entry.value.Equals(other.value)
}

type mapEntrySlice []mapEntry

func (mes mapEntrySlice) Len() int           { return len(mes) }
func (mes mapEntrySlice) Swap(i, j int)      { mes[i], mes[j] = mes[j], mes[i] }
func (mes mapEntrySlice) Less(i, j int) bool { return mes[i].key.Less(mes[j].key) }
func (mes mapEntrySlice) Equals(other mapEntrySlice) bool {
	if mes.Len() != other.Len() {
		return false
	}

	for i, v := range mes {
		if !v.equals(other[i]) {
			return false
		}
	}

	return true
}

func newMapLeafSequence(vrw ValueReadWriter, data ...mapEntry) orderedSequence {
	d.PanicIfTrue(vrw == nil)
	offsets := make([]uint32, 3+len(data))
	w := newBinaryNomsWriter()
	enc := newValueEncoder(w)
	enc.writeKind(MapKind)
	offsets = append(offsets, w.offset)
	offsets[0] = w.offset
	enc.writeCount(0) // level
	offsets[1] = w.offset
	enc.writeCount(uint64(len(data)))
	offsets[2] = w.offset
	for i, me := range data {
		me.writeTo(enc)
		offsets[i+3] = w.offset
	}
	return mapLeafSequence{leafSequence{vrw, w.data(), offsets}}
}

func (ml mapLeafSequence) writeTo(enc *valueEncoder) {
	enc.writeRaw(ml.buff)
}

// sequence interface

func (ml mapLeafSequence) getItem(idx int) sequenceItem {
	offset := ml.getItemOffset(idx)
	if offset == -1 {
		return mapEntry{}
	}
	dec := ml.decoderAtOffset(offset)
	return readMapEntry(dec)
}

func (ml mapLeafSequence) WalkRefs(cb RefCallback) {
	dec, count := ml.decoderSkipToValues()
	for i := uint64(0); i < count*2; i++ {
		dec.readValue().WalkRefs(cb)
	}
}

func (ml mapLeafSequence) entries() mapEntrySlice {
	dec, count := ml.decoderSkipToValues()
	entries := make(mapEntrySlice, count)
	for i := uint64(0); i < count; i++ {
		entries[i] = mapEntry{dec.readValue(), dec.readValue()}
	}
	return entries
}

func (ml mapLeafSequence) getCompareFn(other sequence) compareFn {
	return func(idx, otherIdx int) bool {
		return ml.getItem(idx).(mapEntry).equals(other.getItem(otherIdx).(mapEntry))
	}
}

func (ml mapLeafSequence) typeOf() *Type {
	dec, count := ml.decoderSkipToValues()
	kts := make([]*Type, count)
	vts := make([]*Type, count)
	for i := uint64(0); i < count; i++ {
		kts[i] = dec.readValue().typeOf()
		vts[i] = dec.readValue().typeOf()
	}
	return makeCompoundType(MapKind, makeCompoundType(UnionKind, kts...), makeCompoundType(UnionKind, vts...))
}

// orderedSequence interface

func (ml mapLeafSequence) decoderSkipToIndex(idx int) *valueDecoder {
	offset := ml.getItemOffset(idx)
	// if offset == -1 {
	// 	return nil
	// }
	return ml.decoderAtOffset(offset)
	//
	// dec, count := ml.decoderSkipToValues()
	// if offset == -1
	// 	return nil
	// }
	// for ; idx > 0; idx-- {
	// 	dec.skipValue()
	// 	dec.skipValue()
	// }
	// return dec
}

func (ml mapLeafSequence) getKey(idx int) orderedKey {
	dec := ml.decoderSkipToIndex(idx)
	// TODO: Out of bounds
	// if dec == nil {
	// 	return orderedKey{}
	// }

	return newOrderedKey(dec.readValue())
}

func (ml mapLeafSequence) search(key orderedKey) int {
	return sort.Search(int(ml.Len()), func(i int) bool {
		return !ml.getKey(i).Less(key)
	})
}

func (ml mapLeafSequence) getValue(idx int) Value {
	dec := ml.decoderSkipToIndex(idx)
	// if dec == nil {
	// 	return nil
	// }

	dec.skipValue()
	return dec.readValue()
}
