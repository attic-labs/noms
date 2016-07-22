// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

const (
	ParentsField = "parents"
	ValueField   = "value"
	MetaField    = "meta"
)

var valueCommitType = makeCommitType(types.ValueType)

// NewCommit creates a new commit object. The type of Commit is computed based on the type of the value and the type of the parents.
// It also includes a Meta field whose type is always the empty struct
//
// For the first commit we get:
//
// ```
// struct Commit {
//   meta: struct {},
//   parents: Set<Ref<Cycle<0>>>,
//   value: T,
// }
// ```
//
// As long as we continue to commit values with type T that type stays the same.
//
// When we later commits a value of type U we get:
//
// ```
// struct Commit {
//   meta: struct {},
//   parents: Set<Ref<struct Commit {
//     meta: struct {},
//     parents: Set<Ref<Cycle<0>>>,
//     value: T | U
//   }>>,
//   value: U,
// }
// ```
//
// The new type gets combined as a union type for the value of the inner commit struct.

func NewCommit(value types.Value, parents types.Set, meta types.Struct) types.Struct {
	t := makeCommitType(value.Type(), valueTypesFromParents(parents)...)
	return types.NewStructWithType(t, types.ValueSlice{meta, parents, value})
}

func makeCommitType(valueType *types.Type, parentsValueTypes ...*types.Type) *types.Type {
	tmp := make([]*types.Type, len(parentsValueTypes)+1)
	copy(tmp, parentsValueTypes)
	tmp[len(tmp)-1] = valueType
	parentsValueUnionType := types.MakeUnionType(tmp...)
	fieldNames := []string{MetaField, ParentsField, ValueField}
	var parentsType *types.Type
	if parentsValueUnionType.Equals(valueType) {
		parentsType = types.MakeSetType(types.MakeRefType(types.MakeCycleType(0)))
	} else {
		parentsType = types.MakeSetType(types.MakeRefType(
			types.MakeStructType("Commit", fieldNames, []*types.Type{
				types.EmptyStructType,
				types.MakeSetType(types.MakeRefType(types.MakeCycleType(0))),
				parentsValueUnionType,
			})))
	}
	fieldTypes := []*types.Type{
		types.EmptyStructType,
		parentsType,
		valueType,
	}

	return types.MakeStructType("Commit", fieldNames, fieldTypes)
}

func valueTypesFromParents(parents types.Set) []*types.Type {
	elemType := getSetElementType(parents.Type())
	switch elemType.Kind() {
	case types.UnionKind:
		ts := []*types.Type{}
		for _, rt := range elemType.Desc.(types.CompoundDesc).ElemTypes {
			ts = append(ts, valueFromRefOfCommit(rt))
		}
		return ts
	default:
		return []*types.Type{valueFromRefOfCommit(elemType)}
	}
}

func getSetElementType(t *types.Type) *types.Type {
	d.Chk.True(t.Kind() == types.SetKind)
	return t.Desc.(types.CompoundDesc).ElemTypes[0]
}

func valueFromRefOfCommit(t *types.Type) *types.Type {
	return valueTypeFromCommit(getRefElementType(t))
}

func getRefElementType(t *types.Type) *types.Type {
	d.Chk.True(t.Kind() == types.RefKind)
	return t.Desc.(types.CompoundDesc).ElemTypes[0]
}

func valueTypeFromCommit(t *types.Type) *types.Type {
	d.Chk.True(t.Kind() == types.StructKind && t.Name() == "Commit")
	return t.Desc.(types.StructDesc).Field(ValueField)
}

func IsCommitType(t *types.Type) bool {
	return types.IsSubtype(valueCommitType, t)
}

func isRefOfCommitType(t *types.Type) bool {
	return t.Kind() == types.RefKind && IsCommitType(getRefElementType(t))
}
