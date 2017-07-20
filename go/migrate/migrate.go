// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package migrate

import (
	"fmt"
	"io"

	v712 "gopkg.in/attic-labs/noms.v7/go/types"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// Conv migrates a Noms value of format version 7.12 to the current version.
func Conv(source v712.Value, sourceStore v712.ValueReadWriter, sinkStore types.ValueReadWriter) (dest types.Value, err error) {
	switch source := source.(type) {
	case v712.Bool:
		return types.Bool(bool(source)), nil
	case v712.Number:
		return types.Number(float64(source)), nil
	case v712.String:
		return types.String(string(source)), nil
	case v712.Blob:
		preader, pwriter := io.Pipe()
		go func() {
			source.Copy(pwriter)
			pwriter.Close()
		}()
		return types.NewStreamingBlob(sinkStore, preader), nil
	case v712.List:
		vc := make(chan types.Value, 1024)
		lc := types.NewStreamingList(sinkStore, vc)
		for i := uint64(0); i < source.Len(); i++ {
			var nv types.Value
			nv, err = Conv(source.Get(i), sourceStore, sinkStore)
			if err != nil {
				break
			}
			vc <- nv
		}
		close(vc)
		dest = <-lc
		return
	case v712.Map:
		kvc := make(chan types.Value, 1024)
		mc := types.NewStreamingMap(sinkStore, kvc)
		source.Iter(func(k, v v712.Value) (stop bool) {
			var nk, nv types.Value
			nk, err = Conv(k, sourceStore, sinkStore)
			if err == nil {
				nv, err = Conv(v, sourceStore, sinkStore)
			}
			if err != nil {
				stop = true
			} else {
				kvc <- nk
				kvc <- nv
			}
			return
		})
		close(kvc)
		dest = <-mc
		return
	case v712.Set:
		vc := make(chan types.Value, 1024)
		sc := types.NewStreamingSet(sinkStore, vc)
		source.Iter(func(v v712.Value) (stop bool) {
			var nv types.Value
			nv, err = Conv(v, sourceStore, sinkStore)
			if err != nil {
				stop = true
			} else {
				vc <- nv
			}
			return
		})
		close(vc)
		dest = <-sc
		return
	case v712.Struct:
		fields := make(map[string]types.Value, source.Len())
		source.IterFields(func(name string, v v712.Value) {
			var nv types.Value
			nv, err = Conv(v, sourceStore, sinkStore)
			if err == nil {
				fields[name] = nv
			}
		})
		if err == nil {
			dest = types.NewStruct(source.Name(), fields)
		}
		return
	case *v712.Type:
		return migrateType(source, nil), nil
	case v712.Ref:
		var val types.Value
		v7val := source.TargetValue(sourceStore)
		val, err = Conv(v7val, sourceStore, sinkStore)
		if err == nil {
			dest = sinkStore.WriteValue(val)
		}
		// Special case: you're allowed to explicitly create
		// Ref<Value>. This is the only place in entire API
		// right now where type isn't completely derived.
		if source.TargetType().Equals(v712.ValueType) {
			dest = types.ToRefOfValue(dest.(types.Ref))
		}
		return
	}

	panic(fmt.Sprintf("unreachable type: %T", source))
}

func migrateType(source *v712.Type, seen []*v712.Type) *types.Type {
	if source.TargetKind() == v712.StructKind {
		for _, st := range seen {
			if source.Equals(st) {
				return types.MakeCycleType(source.Desc.(v712.StructDesc).Name)
			}
		}
	}
	seen = append(seen, source)
	migrateChildTypes := func() []*types.Type {
		sc := source.Desc.(v712.CompoundDesc).ElemTypes
		dest := make([]*types.Type, 0, len(sc))
		for i := 0; i < len(sc); i++ {
			dest = append(dest, migrateType(sc[i], seen))
		}
		return dest
	}

	switch source.TargetKind() {
	case v712.BoolKind:
		return types.BoolType
	case v712.NumberKind:
		return types.NumberType
	case v712.StringKind:
		return types.StringType
	case v712.BlobKind:
		return types.BlobType
	case v712.ValueKind:
		return types.ValueType
	case v712.ListKind:
		return types.MakeListType(migrateChildTypes()[0])
	case v712.MapKind:
		ct := migrateChildTypes()
		d.Chk.Equal(2, len(ct))
		return types.MakeMapType(ct[0], ct[1])
	case v712.SetKind:
		return types.MakeSetType(migrateChildTypes()[0])
	case v712.RefKind:
		return types.MakeRefType(migrateChildTypes()[0])
	case v712.UnionKind:
		return types.MakeUnionType(migrateChildTypes()...)
	case v712.TypeKind:
		return types.TypeType
	case v712.StructKind:
		sd := source.Desc.(v712.StructDesc)
		fields := make([]types.StructField, 0, sd.Len())
		sd.IterFields(func(name string, t *v712.Type, optional bool) {
			d.PanicIfTrue(optional) // v7 did not have optional fields.
			fields = append(fields, types.StructField{
				Name: name,
				Type: migrateType(t, seen),
			})
		})
		return types.MakeStructType(sd.Name, fields...)
	case v712.CycleKind:
		// falls through to unreachable below.
		// we should be catching this case with the seen check at top of function.
	}

	panic(fmt.Sprintf("unreachable kind: %d", source.TargetKind()))
}
