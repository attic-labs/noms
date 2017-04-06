// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

func makePrimitiveType(k NomsKind) *Type {
	return newType(PrimitiveDesc(k))
}

var BoolType = makePrimitiveType(BoolKind)
var NumberType = makePrimitiveType(NumberKind)
var StringType = makePrimitiveType(StringKind)
var BlobType = makePrimitiveType(BlobKind)
var TypeType = makePrimitiveType(TypeKind)
var ValueType = makePrimitiveType(ValueKind)

func makeCompoundType(kind NomsKind, elemTypes ...*Type) *Type {
	return newType(CompoundDesc{kind, elemTypes})
}

func makeStructTypeQuickly(name string, fields structTypeFields, checkKind checkKindType) *Type {
	t := newType(StructDesc{name, fields})
	if t.HasUnresolvedCycle() {
		t, _ = toUnresolvedType(t, map[string]*Type{})
		resolveStructCycles(t, map[string]*Type{})
		if !t.HasUnresolvedCycle() {
			checkStructType(t, checkKind)
		}
	}
	return t
}

func makeStructType(name string, fields structTypeFields) *Type {
	verifyStructName(name)
	verifyFields(fields)
	return makeStructTypeQuickly(name, fields, checkKindNormalize)
}

func indexOfType(t *Type, tl []*Type) (uint32, bool) {
	for i, tt := range tl {
		if tt == t {
			return uint32(i), true
		}
	}
	return 0, false
}

// Returns a new type where cyclic pointer references are replaced with Cycle<Name> types.
func toUnresolvedType(t *Type, seenStructs map[string]*Type) (*Type, bool) {
	switch desc := t.Desc.(type) {
	case CompoundDesc:
		ts := make(typeSlice, len(desc.ElemTypes))
		didChange := false
		for i, et := range desc.ElemTypes {
			st, changed := toUnresolvedType(et, seenStructs)
			ts[i] = st
			didChange = didChange || changed
		}

		if !didChange {
			return t, false
		}

		return newType(CompoundDesc{t.TargetKind(), ts}), true
	case StructDesc:
		name := desc.Name
		if name != "" {
			if _, ok := seenStructs[name]; ok {
				return newType(CycleDesc(name)), true
			}
		}

		nt := newType(StructDesc{Name: name})
		if name != "" {
			seenStructs[name] = nt
		}

		fs := make(structTypeFields, len(desc.fields))
		didChange := false
		for i, f := range desc.fields {
			st, changed := toUnresolvedType(f.Type, seenStructs)
			fs[i] = StructField{f.Name, st, f.Optional}
			didChange = didChange || changed
		}

		desc.fields = fs
		nt.Desc = desc
		return nt, true
	case CycleDesc:
		cycleName := string(desc)
		_, ok := seenStructs[cycleName]
		return t, ok // Only cycles which can be resolved in the current struct.
	}

	return t, false
}

// ToUnresolvedType replaces cycles (by pointer comparison) in types to Cycle types.
func ToUnresolvedType(t *Type) *Type {
	t2, _ := toUnresolvedType(t, map[string]*Type{})
	return t2
}

// Drops cycles and replaces them with pointers to parent structs
func resolveStructCycles(t *Type, seenStructs map[string]*Type) *Type {
	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for i, et := range desc.ElemTypes {
			desc.ElemTypes[i] = resolveStructCycles(et, seenStructs)
		}

	case StructDesc:
		name := desc.Name
		if name != "" {
			seenStructs[name] = t
		}
		for i, f := range desc.fields {
			desc.fields[i].Type = resolveStructCycles(f.Type, seenStructs)
		}

	case CycleDesc:
		name := string(desc)
		if nt, ok := seenStructs[name]; ok {
			return nt
		}
	}

	return t
}

// We normalize structs during their construction iff they have no unresolved
// cycles. Normalizing applies a canonical ordering to the composite types of a
// union and serializes all types under the struct. To ensure a consistent
// ordering of the composite types of a union, we generate a unique "order id"
// or OID for each of those types. The OID is the hash of a unique type
// encoding that is independent of the extant order of types within any
// subordinate unions. This encoding for most types is a straightforward
// serialization of its components; for unions the encoding is a bytewise XOR
// of the hashes of each of its composite type encodings.
//
// We require a consistent order of types within a union to ensure that
// equivalent types have a single persistent encoding and, therefore, a single
// hash. The method described above fails for "unrolled" cycles whereby two
// equivalent, but uniquely described structures, would have different OIDs.
// Consider for example the following two types that, while equivalent, do not
// yield the same OID:
//
//   Struct A { a: Cycle<0> }
//   Struct A { a: Struct A { a: Cycle<1> } }
//
// We explicitly disallow this sort of redundantly expressed type. If a
// non-Byzantine use of such a construction arises, we can attempt to simplify
// the expansive type or find another means of comparison.

type checkKindType uint8

const (
	checkKindNormalize checkKindType = iota
	checkKindNoValidate
	checkKindValidate
)

func checkStructType(t *Type, checkKind checkKindType) {
	if checkKind == checkKindNoValidate {
		return
	}

	switch checkKind {
	case checkKindNormalize:
		walkType(t, nil, sortUnions)
	case checkKindValidate:
		walkType(t, nil, validateTypes)
	default:
		panic("unreachable")
	}
}

func sortUnions(t *Type, _ []*Type) {
	if t.TargetKind() == UnionKind {
		sort.Sort(t.Desc.(CompoundDesc).ElemTypes)
	}
}

func validateTypes(t *Type, _ []*Type) {
	switch t.TargetKind() {
	case UnionKind:
		elemTypes := t.Desc.(CompoundDesc).ElemTypes
		if len(elemTypes) == 1 {
			panic("Invalid union type")
		}
		for i := 1; i < len(elemTypes); i++ {
			if !unionLess(elemTypes[i-1], elemTypes[i]) {
				panic("Invalid union order")
			}
		}
	case StructKind:
		desc := t.Desc.(StructDesc)
		verifyStructName(desc.Name)
		verifyFields(desc.fields)
	}
}

func walkType(t *Type, parentStructTypes []*Type, cb func(*Type, []*Type)) {
	if t.TargetKind() == StructKind {
		if _, found := indexOfType(t, parentStructTypes); found {
			return
		}
	}

	cb(t, parentStructTypes)

	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for _, tt := range desc.ElemTypes {
			walkType(tt, parentStructTypes, cb)
		}
	case StructDesc:
		for _, f := range desc.fields {
			walkType(f.Type, append(parentStructTypes, t), cb)
		}
	}
}

// MakeUnionType creates a new union type unless the elemTypes can be folded into a single non union type.
func makeUnionType(elemTypes ...*Type) *Type {
	return makeSimplifiedType(false, makeCompoundType(UnionKind, elemTypes...))
}

func MakeListType(elemType *Type) *Type {
	return makeSimplifiedType(false, makeCompoundType(ListKind, elemType))
}

func MakeSetType(elemType *Type) *Type {
	return makeSimplifiedType(false, makeCompoundType(SetKind, elemType))
}

func MakeRefType(elemType *Type) *Type {
	return makeSimplifiedType(false, makeCompoundType(RefKind, elemType))
}

func MakeMapType(keyType, valType *Type) *Type {
	return makeSimplifiedType(false, makeCompoundType(MapKind, keyType, valType))
}

type FieldMap map[string]*Type

func MakeStructTypeFromFields(name string, fields FieldMap) *Type {
	fs := make(structTypeFields, len(fields))
	i := 0
	for k, v := range fields {
		fs[i] = StructField{k, v, false}
		i++
	}
	sort.Sort(&fs)
	return makeStructType(name, fs)
}

// StructField describes a field in a struct type.
type StructField struct {
	Name     string
	Type     *Type
	Optional bool
}

type structTypeFields []StructField

func (s structTypeFields) Len() int           { return len(s) }
func (s structTypeFields) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s structTypeFields) Less(i, j int) bool { return s[i].Name < s[j].Name }

func MakeStructType(name string, fields ...StructField) *Type {
	fs := structTypeFields(fields)
	sort.Sort(&fs)

	return makeStructType(name, fs)
}

func MakeUnionType(elemTypes ...*Type) *Type {
	return makeUnionType(elemTypes...)
}

// MakeUnionTypeIntersectStructs is a bit of strange function. It creates a
// simplified union type except for structs, where it creates interesection
// types.
// This function will go away so do not use it!
func MakeUnionTypeIntersectStructs(elemTypes ...*Type) *Type {
	return makeSimplifiedType(true, makeCompoundType(UnionKind, elemTypes...))
}

func MakeCycleType(name string) *Type {
	if name == "" {
		d.Panic("Cycle type must have a non empty name")
	}
	return newType(CycleDesc(name))
}
