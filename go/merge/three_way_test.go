// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

var (
	aa1      = []interface{}{"a1", "a-one", "a2", "a-two", "a3", "a-three", "a4", "a-four"}
	aa1a     = []interface{}{"a1", "a-one", "a2", "a-two", "a3", "a-three-diff", "a4", "a-four", "a6", "a-six"}
	aa1b     = []interface{}{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five"}
	aaMerged = []interface{}{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five", "a6", "a-six"}

	mm1       = []interface{}{}
	mm1a      = []interface{}{"k1", []interface{}{"a", 0}}
	mm1b      = []interface{}{"k1", []interface{}{"b", 1}}
	mm1Merged = []interface{}{"k1", []interface{}{"a", 0, "b", 1}}

	mm2       = []interface{}{"k2", aa1, "k3", "k-three"}
	mm2a      = []interface{}{"k1", []interface{}{"a", 0}, "k2", aa1a, "k3", "k-three", "k4", "k-four"}
	mm2b      = []interface{}{"k1", []interface{}{"b", 1}, "k2", aa1b}
	mm2Merged = []interface{}{"k1", []interface{}{"a", 0, "b", 1}, "k2", aaMerged, "k4", "k-four"}
)

func tryThreeWayMerge(t *testing.T, a, b, p, expected types.Value, vs types.ValueReadWriter) {
	merged, err := ThreeWay(a, b, p, vs)
	if assert.NoError(t, err) {
		assert.True(t, expected.Equals(merged), "%s != %s", types.EncodedValue(expected), types.EncodedValue(merged))
	}
}

func tryThreeWayConflict(t *testing.T, a, b, p types.Value, contained string) {
	_, err := ThreeWay(a, b, p, nil)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), contained)
	}
}

func TestThreeWayMergeMap_DoNothing(t *testing.T) {
	tryThreeWayMerge(t, nil, nil, cm(aa1), cm(aa1), nil)
}

func TestThreeWayMergeMap_NoRecursion(t *testing.T) {
	tryThreeWayMerge(t, cm(aa1a), cm(aa1b), cm(aa1), cm(aaMerged), nil)
	tryThreeWayMerge(t, cm(aa1b), cm(aa1a), cm(aa1), cm(aaMerged), nil)
}

func TestThreeWayMergeMap_RecursiveCreate(t *testing.T) {
	tryThreeWayMerge(t, cm(mm1a), cm(mm1b), cm(mm1), cm(mm1Merged), nil)
	tryThreeWayMerge(t, cm(mm1b), cm(mm1a), cm(mm1), cm(mm1Merged), nil)
}

func TestThreeWayMergeMap_RecursiveCreateNil(t *testing.T) {
	tryThreeWayMerge(t, cm(mm1a), cm(mm1b), nil, cm(mm1Merged), nil)
	tryThreeWayMerge(t, cm(mm1b), cm(mm1a), nil, cm(mm1Merged), nil)
}

func TestThreeWayMergeMap_RecursiveMerge(t *testing.T) {
	tryThreeWayMerge(t, cm(mm2a), cm(mm2b), cm(mm2), cm(mm2Merged), nil)
	tryThreeWayMerge(t, cm(mm2b), cm(mm2a), cm(mm2), cm(mm2Merged), nil)
}

func TestThreeWayMergeMap_RefMerge(t *testing.T) {
	vs := types.NewTestValueStore()

	strRef := vs.WriteValue(types.NewStruct("Foo", types.StructData{"life": types.Number(42)}))

	create := func(kv ...interface{}) types.Map {
		return cm(kv)
	}
	m := create("r2", vs.WriteValue(cm(aa1)))
	ma := create("r1", strRef, "r2", vs.WriteValue(cm(aa1a)))
	mb := create("r1", strRef, "r2", vs.WriteValue(cm(aa1b)))
	mMerged := create("r1", strRef, "r2", vs.WriteValue(cm(aaMerged)))
	vs.Flush()

	tryThreeWayMerge(t, ma, mb, m, mMerged, vs)
	tryThreeWayMerge(t, mb, ma, m, mMerged, vs)
}

func TestThreeWayMergeMap_RecursiveMultiLevelMerge(t *testing.T) {
	vs := types.NewTestValueStore()

	create := func(kv ...interface{}) types.Map {
		return cm(kv)
	}
	m := create("mm1", cm(mm1), "mm2", vs.WriteValue(cm(mm2)))
	ma := create("mm1", cm(mm1a), "mm2", vs.WriteValue(cm(mm2a)))
	mb := create("mm1", cm(mm1b), "mm2", vs.WriteValue(cm(mm2b)))
	mMerged := create("mm1", cm(mm1Merged), "mm2", vs.WriteValue(cm(mm2Merged)))
	vs.Flush()

	tryThreeWayMerge(t, ma, mb, m, mMerged, vs)
	tryThreeWayMerge(t, mb, ma, m, mMerged, vs)
}

func TestThreeWayMergeMap_NilConflict(t *testing.T) {
	tryThreeWayConflict(t, nil, cm(mm2b), cm(mm2), "Cannot merge nil Value with")
	tryThreeWayConflict(t, cm(mm2a), nil, cm(mm2), "with nil value.")
}

func TestThreeWayMergeMap_ImmediateConflict(t *testing.T) {
	tryThreeWayConflict(t, types.NewSet(), cm(mm2b), cm(mm2), "Cannot merge Set<> with Map")
	tryThreeWayConflict(t, cm(mm2b), types.NewSet(), cm(mm2), "Cannot merge Map")
}

func TestThreeWayMergeMap_NestedConflict(t *testing.T) {
	tryThreeWayConflict(t, cm(mm2a).Set(types.String("k2"), types.NewSet()), cm(mm2b), cm(mm2), types.EncodedValue(types.NewSet()))
	tryThreeWayConflict(t, cm(mm2a).Set(types.String("k2"), types.NewSet()), cm(mm2b), cm(mm2), types.EncodedValue(cm(aa1b)))
}

func TestThreeWayMergeMap_NestedConflictingOperation(t *testing.T) {
	key := types.String("k2")
	tryThreeWayConflict(t, cm(mm2a).Remove(key), cm(mm2b), cm(mm2), "removed "+types.EncodedValue(key))
	tryThreeWayConflict(t, cm(mm2a).Remove(key), cm(mm2b), cm(mm2), "modded "+types.EncodedValue(key))
}

func TestThreeWayMergeStruct_DoNothing(t *testing.T) {
	tryThreeWayMerge(t, nil, nil, cs(aa1), cs(aa1), nil)
}

func TestThreeWayMergeStruct_NoRecursion(t *testing.T) {
	tryThreeWayMerge(t, cs(aa1a), cs(aa1b), cs(aa1), cs(aaMerged), nil)
	tryThreeWayMerge(t, cs(aa1b), cs(aa1a), cs(aa1), cs(aaMerged), nil)
}

func TestThreeWayMergeStruct_RecursiveCreate(t *testing.T) {
	tryThreeWayMerge(t, cs(mm1a), cs(mm1b), cs(mm1), cs(mm1Merged), nil)
	tryThreeWayMerge(t, cs(mm1b), cs(mm1a), cs(mm1), cs(mm1Merged), nil)
}

func TestThreeWayMergeStruct_RecursiveCreateNil(t *testing.T) {
	tryThreeWayMerge(t, cs(mm1a), cs(mm1b), nil, cs(mm1Merged), nil)
	tryThreeWayMerge(t, cs(mm1b), cs(mm1a), nil, cs(mm1Merged), nil)
}

func TestThreeWayMergeStruct_RecursiveMerge(t *testing.T) {
	tryThreeWayMerge(t, cs(mm2a), cs(mm2b), cs(mm2), cs(mm2Merged), nil)
	tryThreeWayMerge(t, cs(mm2b), cs(mm2a), cs(mm2), cs(mm2Merged), nil)
}

func TestThreeWayMergeStruct_RefMerge(t *testing.T) {
	vs := types.NewTestValueStore()

	strRef := vs.WriteValue(types.NewStruct("Foo", types.StructData{"life": types.Number(42)}))

	create := func(kv ...interface{}) types.Struct {
		return cs(kv)
	}
	m := create("r2", vs.WriteValue(cs(aa1)))
	ma := create("r1", strRef, "r2", vs.WriteValue(cs(aa1a)))
	mb := create("r1", strRef, "r2", vs.WriteValue(cs(aa1b)))
	mMerged := create("r1", strRef, "r2", vs.WriteValue(cs(aaMerged)))
	vs.Flush()

	tryThreeWayMerge(t, ma, mb, m, mMerged, vs)
	tryThreeWayMerge(t, mb, ma, m, mMerged, vs)
}

func TestThreeWayMergeStruct_RecursiveMultiLevelMerge(t *testing.T) {
	vs := types.NewTestValueStore()

	create := func(kv ...interface{}) types.Struct {
		return cs(kv)
	}
	m := create("mm1", cs(mm1), "mm2", vs.WriteValue(cs(mm2)))
	ma := create("mm1", cs(mm1a), "mm2", vs.WriteValue(cs(mm2a)))
	mb := create("mm1", cs(mm1b), "mm2", vs.WriteValue(cs(mm2b)))
	mMerged := create("mm1", cs(mm1Merged), "mm2", vs.WriteValue(cs(mm2Merged)))
	vs.Flush()

	tryThreeWayMerge(t, ma, mb, m, mMerged, vs)
	tryThreeWayMerge(t, mb, ma, m, mMerged, vs)
}

func TestThreeWayMergeStruct_NilConflict(t *testing.T) {
	tryThreeWayConflict(t, nil, cs(mm2b), cs(mm2), "Cannot merge nil Value with")
	tryThreeWayConflict(t, cs(mm2a), nil, cs(mm2), "with nil value.")
}

func TestThreeWayMergeStruct_ImmediateConflict(t *testing.T) {
	tryThreeWayConflict(t, types.NewSet(), cs(mm2b), cs(mm2), "Cannot merge Set<> with struct")
	tryThreeWayConflict(t, cs(mm2b), types.NewSet(), cs(mm2), "Cannot merge struct")
}

func TestThreeWayMergeStruct_NestedConflict(t *testing.T) {
	a := cs([]interface{}{"k2", "not-a-struct"})
	tryThreeWayConflict(t, a, cs(mm2b), cs(mm2), "not-a-struct")
	tryThreeWayConflict(t, a, cs(mm2b), cs(mm2), types.EncodedValue(cs(aa1b)))
}

func TestThreeWayMergeStruct_NestedConflictingOperation(t *testing.T) {
	key := "k2"
	a := cs([]interface{}{"k3", "k-three"})
	tryThreeWayConflict(t, a, cs(mm2b), cs(mm2), "removed "+types.EncodedValue(types.String(key)))
	tryThreeWayConflict(t, a, cs(mm2b), cs(mm2), "modded "+types.EncodedValue(types.String(key)))
}

func cm(kv []interface{}) types.Map {
	keyValues := valsToTypesValues(func(kv []interface{}) types.Value { return cm(kv) }, kv...)
	return types.NewMap(keyValues...)
}

func cs(kv []interface{}) types.Struct {
	fields := types.StructData{}
	recurse := func(kv []interface{}) types.Value { return cs(kv) }
	for i := 0; i < len(kv); i += 2 {
		fields[kv[i].(string)] = valToTypesValue(recurse, kv[i+1])
	}
	return types.NewStruct("TestStruct", fields)
}

func valsToTypesValues(f func([]interface{}) types.Value, kv ...interface{}) []types.Value {
	keyValues := []types.Value{}
	for _, e := range kv {
		v := valToTypesValue(f, e)
		keyValues = append(keyValues, v)
	}
	return keyValues
}

func valToTypesValue(f func([]interface{}) types.Value, v interface{}) types.Value {
	var v1 types.Value
	switch t := v.(type) {
	case string:
		v1 = types.String(t)
	case int:
		v1 = types.Number(t)
	case []interface{}:
		v1 = f(t)
	case types.Value:
		v1 = t
	}
	return v1
}
