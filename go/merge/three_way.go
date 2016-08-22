// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"fmt"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

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

// ThreeWay attempts a three-way merge between two candidates and a common ancestor. It considers the three of them recursively, applying some simple rules to identify conflicts:
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
//    - If the same object is both removed and inserted wrt parent: conflict
//
// All other modifications are allowed.
// Currently, ThreeWay() only works on types.Map.
func ThreeWay(a, b, parent types.Value, vwr types.ValueReadWriter) (merged types.Value, err error) {
	if a == nil && b == nil {
		return parent, nil
	} else if a == nil {
		return parent, newMergeConflict("Cannot merge nil Value with %s.", b.Type().Describe())
	} else if b == nil {
		return parent, newMergeConflict("Cannot merge %s with nil value.", a.Type().Describe())
	} else if unmergeable(a, b) {
		return parent, newMergeConflict("Cannot merge %s with %s.", a.Type().Describe(), b.Type().Describe())
	}

	return threeWayMerge(a, b, parent, vwr)
}

// a and b cannot be merged if they are of different NomsKind, or if at least one of the two is nil, or if either is a Noms primitive.
func unmergeable(a, b types.Value) bool {
	if a != nil && b != nil {
		aKind, bKind := a.Type().Kind(), b.Type().Kind()
		return aKind != bKind || types.IsPrimitiveKind(aKind) || types.IsPrimitiveKind(bKind)
	}
	return true
}

func threeWayMerge(a, b, parent types.Value, vwr types.ValueReadWriter) (merged types.Value, err error) {
	d.PanicIfTrue(a == nil || b == nil, "Merge candidates cannont be nil: a = %v, b = %v", a, b)
	newTypeConflict := func() *ErrMergeConflict {
		pDescription := "<nil>"
		if parent != nil {
			pDescription = parent.Type().Describe()
		}
		return newMergeConflict("Cannot merge %s and %s on top of %s.", a.Type().Describe(), b.Type().Describe(), pDescription)
	}

	switch a.Type().Kind() {
	case types.ListKind:
		// TODO: Come up with a plan for List (BUG 148)
		return parent, newMergeConflict("Cannot merge %s.", a.Type().Describe())

	case types.MapKind:
		if aMap, bMap, pMap, ok := mapAssert(a, b, parent); ok {
			return threeWayMapMerge(aMap, bMap, pMap, vwr)
		}

	case types.RefKind:
		if aValue, bValue, pValue, ok := refAssert(a, b, parent, vwr); ok {
			merged, err := threeWayMerge(aValue, bValue, pValue, vwr)
			if err != nil {
				return parent, err
			}
			return vwr.WriteValue(merged), nil
		}

	case types.SetKind:
		// TODO: Implement plan from BUG148
		return parent, newMergeConflict("Cannot merge %s.", a.Type().Describe())

	case types.StructKind:
		if aStruct, bStruct, pStruct, ok := structAssert(a, b, parent); ok {
			return threeWayStructMerge(aStruct, bStruct, pStruct, vwr)
		}

	default:
		return parent, newMergeConflict("Cannot merge %s.", a.Type().Describe())

	}
	return parent, newTypeConflict()
}

func threeWayMapMerge(a, b, parent types.Map, vwr types.ValueReadWriter) (merged types.Value, err error) {
	aChangeChan, bChangeChan := make(chan types.ValueChanged), make(chan types.ValueChanged)
	aStopChan, bStopChan := make(chan struct{}, 1), make(chan struct{}, 1)

	go func() {
		a.DiffLeftRight(parent, aChangeChan, aStopChan)
		close(aChangeChan)
	}()
	go func() {
		b.DiffLeftRight(parent, bChangeChan, bStopChan)
		close(bChangeChan)
	}()

	defer stopAndDrain(aStopChan, aChangeChan)
	defer stopAndDrain(bStopChan, bChangeChan)

	apply := func(target types.Value, change types.ValueChanged, newVal types.Value) types.Value {
		switch change.ChangeType {
		case types.DiffChangeAdded, types.DiffChangeModified:
			return target.(types.Map).Set(change.V, newVal)
		case types.DiffChangeRemoved:
			return target.(types.Map).Remove(change.V)
		default:
			panic("Not Reached")
		}
	}
	return threeWayOrderedSequenceMerge(parent, aChangeChan, bChangeChan, a.Get, b.Get, parent.Get, apply, vwr)
}

func threeWayStructMerge(a, b, parent types.Struct, vwr types.ValueReadWriter) (merged types.Value, err error) {
	aChangeChan, bChangeChan := make(chan types.ValueChanged), make(chan types.ValueChanged)
	aStopChan, bStopChan := make(chan struct{}, 1), make(chan struct{}, 1)

	go func() {
		a.Diff(parent, aChangeChan, aStopChan)
		close(aChangeChan)
	}()
	go func() {
		b.Diff(parent, bChangeChan, bStopChan)
		close(bChangeChan)
	}()

	defer stopAndDrain(aStopChan, aChangeChan)
	defer stopAndDrain(bStopChan, bChangeChan)

	makeGetFunc := func(s types.Struct) getFunc {
		return func(key types.Value) types.Value {
			if field, ok := key.(types.String); ok {
				if val, present := s.MaybeGet(string(field)); present {
					return val
				}
				return nil
			}
			panic(fmt.Errorf("Bad key type in diff: %s", key.Type().Describe()))
		}
	}

	apply := func(target types.Value, change types.ValueChanged, newVal types.Value) types.Value {
		// Right now, this always iterates over all fields to create a new Struct, because there's no API for adding/removing a field from an existing struct type.
		if f, ok := change.V.(types.String); ok {
			field := string(f)
			// fmt.Println("Applying change", describeChange(change))
			data := types.StructData{}
			desc := target.Type().Desc.(types.StructDesc)
			desc.IterFields(func(name string, t *types.Type) {
				if name != field {
					data[name] = target.(types.Struct).Get(name)
				}
			})
			if change.ChangeType == types.DiffChangeAdded || change.ChangeType == types.DiffChangeModified {
				data[field] = newVal
			}
			return types.NewStruct(desc.Name, data)
		}
		panic(fmt.Errorf("Bad key type in diff: %s", change.V.Type().Describe()))
	}
	return threeWayOrderedSequenceMerge(parent, aChangeChan, bChangeChan, makeGetFunc(a), makeGetFunc(b), makeGetFunc(parent), apply, vwr)
}

func stopAndDrain(stop chan<- struct{}, drain <-chan types.ValueChanged) {
	close(stop)
	for range drain {
	}
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
	ok = aOk && bOk && pOk
	return
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
