// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"fmt"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

type ResolveFunc func(aChange, bChange types.ValueChanged, a, b types.Value, path types.Path) (change types.ValueChanged, merged types.Value, ok bool)

// ErrMergeConflict indicates that a merge attempt failed and must be resolved manually for the provided reason.
type ErrMergeConflict struct {
	msg string
}

func (e *ErrMergeConflict) Error() string {
	return e.msg
}

func newMergeConflict(format string, args ...interface{}) *ErrMergeConflict {
	return &ErrMergeConflict{fmt.Sprintf(format, args...)}
}

// ThreeWay attempts a three-way merge between two candidates and a common ancestor.
// It considers the three of them recursively, applying some simple rules to identify conflicts:
//  - If any of the three nodes are different NomsKinds: conflict
//  - If we are dealing with a map:
//    - If the same key is both removed and inserted wrt parent: conflict
//    - If the same key is inserted wrt parent, but with different values: conflict
//  - If we are dealing with a struct:
//    - If the same field is both removed and inserted wrt parent: conflict
//    - If the same field is inserted wrt parent, but with different values: conflict
//  - If we are dealing with a list:
//    - If the same index is both removed and inserted wrt parent: conflict
//    - If the same index is inserted wrt parent, but with different values: conflict
//  - If we are dealing with a set:
//    - `merged` is essentially union(a, b, parent)
//
// All other modifications are allowed.
// ThreeWay() works on types.Map, types.Set, and types.Struct.
func ThreeWay(a, b, parent types.Value, vwr types.ValueReadWriter, resolve ResolveFunc, progress chan struct{}) (merged types.Value, err error) {
	if a == nil && b == nil {
		return parent, nil
	} else if a == nil {
		return parent, newMergeConflict("Cannot merge nil Value with %s.", b.Type().Describe())
	} else if b == nil {
		return parent, newMergeConflict("Cannot merge %s with nil value.", a.Type().Describe())
	} else if unmergeable(a, b) {
		return parent, newMergeConflict("Cannot merge %s with %s.", a.Type().Describe(), b.Type().Describe())
	}

	if resolve == nil {
		resolve = defaultResolve
	}
	m := &merger{vwr, resolve, progress}
	return m.threeWay(a, b, parent, types.Path{})
}

// a and b cannot be merged if they are of different NomsKind, or if at least one of the two is nil, or if either is a Noms primitive.
func unmergeable(a, b types.Value) bool {
	if a != nil && b != nil {
		aKind, bKind := a.Type().Kind(), b.Type().Kind()
		return aKind != bKind || types.IsPrimitiveKind(aKind) || types.IsPrimitiveKind(bKind)
	}
	return true
}

type merger struct {
	vwr      types.ValueReadWriter
	resolve  ResolveFunc
	progress chan<- struct{}
}

func defaultResolve(aChange, bChange types.ValueChanged, a, b types.Value, p types.Path) (change types.ValueChanged, merged types.Value, ok bool) {
	return
}

func updateProgress(progress chan<- struct{}) {
	// TODO: Eventually we'll want more information than a single bit :).
	if progress != nil {
		progress <- struct{}{}
	}
}

func (m *merger) threeWay(a, b, parent types.Value, path types.Path) (merged types.Value, err error) {
	defer updateProgress(m.progress)
	d.PanicIfTrue(a == nil || b == nil, "Merge candidates cannont be nil: a = %v, b = %v", a, b)

	switch a.Type().Kind() {
	case types.ListKind:
		if aList, bList, pList, ok := listAssert(a, b, parent); ok {
			return threeWayListMerge(aList, bList, pList)
		}

	case types.MapKind:
		if aMap, bMap, pMap, ok := mapAssert(a, b, parent); ok {
			return m.threeWayMapMerge(aMap, bMap, pMap, path)
		}

	case types.RefKind:
		if aValue, bValue, pValue, ok := refAssert(a, b, parent, m.vwr); ok {
			merged, err := m.threeWay(aValue, bValue, pValue, path)
			if err != nil {
				return parent, err
			}
			return m.vwr.WriteValue(merged), nil
		}

	case types.SetKind:
		if aSet, bSet, pSet, ok := setAssert(a, b, parent); ok {
			return m.threeWaySetMerge(aSet, bSet, pSet, path)
		}

	case types.StructKind:
		if aStruct, bStruct, pStruct, ok := structAssert(a, b, parent); ok {
			return m.threeWayStructMerge(aStruct, bStruct, pStruct, path)
		}
	}

	pDescription := "<nil>"
	if parent != nil {
		pDescription = parent.Type().Describe()
	}
	return parent, newMergeConflict("Cannot merge %s and %s on top of %s.", a.Type().Describe(), b.Type().Describe(), pDescription)
}

func (m *merger) threeWayMapMerge(a, b, parent types.Map, path types.Path) (merged types.Value, err error) {
	type mapLike interface {
		Set(k, v types.Value) types.Map
		Remove(k types.Value) types.Map
	}
	apply := func(target types.Value, change types.ValueChanged, newVal types.Value) types.Value {
		defer updateProgress(m.progress)
		switch change.ChangeType {
		case types.DiffChangeAdded, types.DiffChangeModified:
			return target.(mapLike).Set(change.V, newVal)
		case types.DiffChangeRemoved:
			return target.(mapLike).Remove(change.V)
		default:
			panic("Not Reached")
		}
	}
	return m.threeWayOrderedSequenceMerge(mapCandidate{a}, mapCandidate{b}, mapCandidate{parent}, apply, path)
}

func (m *merger) threeWaySetMerge(a, b, parent types.Set, path types.Path) (merged types.Value, err error) {
	type setLike interface {
		Insert(values ...types.Value) types.Set
		Remove(valus ...types.Value) types.Set
	}
	apply := func(target types.Value, change types.ValueChanged, ignored types.Value) types.Value {
		defer updateProgress(m.progress)
		switch change.ChangeType {
		case types.DiffChangeAdded, types.DiffChangeModified:
			return target.(setLike).Insert(change.V)
		case types.DiffChangeRemoved:
			return target.(setLike).Remove(change.V)
		default:
			panic("Not Reached")
		}
	}
	return m.threeWayOrderedSequenceMerge(setCandidate{a}, setCandidate{b}, setCandidate{parent}, apply, path)
}

func (m *merger) threeWayStructMerge(a, b, parent types.Struct, path types.Path) (merged types.Value, err error) {
	type structLike interface {
		Get(string) types.Value
	}
	apply := func(target types.Value, change types.ValueChanged, newVal types.Value) types.Value {
		defer updateProgress(m.progress)
		// Right now, this always iterates over all fields to create a new Struct, because there's no API for adding/removing a field from an existing struct type.
		if f, ok := change.V.(types.String); ok {
			field := string(f)
			data := types.StructData{}
			desc := target.Type().Desc.(types.StructDesc)
			desc.IterFields(func(name string, t *types.Type) {
				if name != field {
					data[name] = target.(structLike).Get(name)
				}
			})
			if change.ChangeType == types.DiffChangeAdded || change.ChangeType == types.DiffChangeModified {
				data[field] = newVal
			}
			return types.NewStruct(desc.Name, data)
		}
		panic(fmt.Errorf("Bad key type in diff: %s", change.V.Type().Describe()))
	}
	return m.threeWayOrderedSequenceMerge(structCandidate{a}, structCandidate{b}, structCandidate{parent}, apply, path)
}

type candidate interface {
	types.Value
	diff(parent types.Value, change chan<- types.ValueChanged, stop <-chan struct{})
	get(k types.Value) types.Value
	pathConcat(change types.ValueChanged, path types.Path) (out types.Path)
}

type mapCandidate struct {
	types.Map
}

func (mc mapCandidate) diff(p types.Value, change chan<- types.ValueChanged, stop <-chan struct{}) {
	mc.Diff(p.(mapCandidate).Map, change, stop)
}

func (mc mapCandidate) get(k types.Value) types.Value {
	return mc.Get(k)
}

func (mc mapCandidate) pathConcat(change types.ValueChanged, path types.Path) (out types.Path) {
	out = append(out, path...)
	if kind := change.V.Type().Kind(); kind == types.BoolKind || kind == types.StringKind || kind == types.NumberKind {
		out = append(out, types.NewIndexPath(change.V))
	} else {
		out = append(out, types.NewHashIndexPath(change.V.Hash()))
	}
	return
}

type setCandidate struct {
	types.Set
}

func (sc setCandidate) diff(p types.Value, change chan<- types.ValueChanged, stop <-chan struct{}) {
	sc.Diff(p.(setCandidate).Set, change, stop)
}

func (sc setCandidate) get(k types.Value) types.Value {
	return k
}

func (sc setCandidate) pathConcat(change types.ValueChanged, path types.Path) (out types.Path) {
	out = append(out, path...)
	if kind := change.V.Type().Kind(); kind == types.BoolKind || kind == types.StringKind || kind == types.NumberKind {
		out = append(out, types.NewIndexPath(change.V))
	} else {
		out = append(out, types.NewHashIndexPath(change.V.Hash()))
	}
	return
}

type structCandidate struct {
	types.Struct
}

func (sc structCandidate) diff(p types.Value, change chan<- types.ValueChanged, stop <-chan struct{}) {
	sc.Diff(p.(structCandidate).Struct, change, stop)
}

func (sc structCandidate) get(key types.Value) types.Value {
	if field, ok := key.(types.String); ok {
		val, _ := sc.MaybeGet(string(field))
		return val
	}
	panic(fmt.Errorf("Bad key type in diff: %s", key.Type().Describe()))
}

func (sc structCandidate) pathConcat(change types.ValueChanged, path types.Path) (out types.Path) {
	out = append(out, path...)
	str, ok := change.V.(types.String)
	d.PanicIfTrue(!ok, "Field names must be strings, not %s", change.V.Type().Describe())
	return append(out, types.NewFieldPath(string(str)))
}

func listAssert(a, b, parent types.Value) (aList, bList, pList types.List, ok bool) {
	var aOk, bOk, pOk bool
	aList, aOk = a.(types.List)
	bList, bOk = b.(types.List)
	if parent != nil {
		pList, pOk = parent.(types.List)
	} else {
		pList, pOk = types.NewList(), true
	}
	return aList, bList, pList, aOk && bOk && pOk
}

func mapAssert(a, b, parent types.Value) (aMap, bMap, pMap types.Map, ok bool) {
	var aOk, bOk, pOk bool
	aMap, aOk = a.(types.Map)
	bMap, bOk = b.(types.Map)
	if parent != nil {
		pMap, pOk = parent.(types.Map)
	} else {
		pMap, pOk = types.NewMap(), true
	}
	return aMap, bMap, pMap, aOk && bOk && pOk
}

func refAssert(a, b, parent types.Value, vwr types.ValueReadWriter) (aValue, bValue, pValue types.Value, ok bool) {
	var aOk, bOk, pOk bool
	var aRef, bRef, pRef types.Ref
	aRef, aOk = a.(types.Ref)
	bRef, bOk = b.(types.Ref)
	if !aOk || !bOk {
		return
	}

	aValue = aRef.TargetValue(vwr)
	bValue = bRef.TargetValue(vwr)
	if parent != nil {
		if pRef, pOk = parent.(types.Ref); pOk {
			pValue = pRef.TargetValue(vwr)
		}
	} else {
		pOk = true // parent == nil is still OK. It just leaves pValue as nil.
	}
	return aValue, bValue, pValue, aOk && bOk && pOk
}

func setAssert(a, b, parent types.Value) (aSet, bSet, pSet types.Set, ok bool) {
	var aOk, bOk, pOk bool
	aSet, aOk = a.(types.Set)
	bSet, bOk = b.(types.Set)
	if parent != nil {
		pSet, pOk = parent.(types.Set)
	} else {
		pSet, pOk = types.NewSet(), true
	}
	return aSet, bSet, pSet, aOk && bOk && pOk
}

func structAssert(a, b, parent types.Value) (aStruct, bStruct, pStruct types.Struct, ok bool) {
	var aOk, bOk, pOk bool
	aStruct, aOk = a.(types.Struct)
	bStruct, bOk = b.(types.Struct)
	if aOk && bOk {
		aDesc, bDesc := a.Type().Desc.(types.StructDesc), b.Type().Desc.(types.StructDesc)
		if aDesc.Name == bDesc.Name {
			if parent != nil {
				pStruct, pOk = parent.(types.Struct)
			} else {
				pStruct, pOk = types.NewStruct(aDesc.Name, nil), true
			}
			return aStruct, bStruct, pStruct, pOk
		}
	}
	return
}
