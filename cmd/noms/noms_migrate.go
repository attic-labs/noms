// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	v7spec "github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	v7types "github.com/attic-labs/noms/go/types"
	flag "github.com/juju/gnuflag"
)

var nomsMigrate = &util.Command{
	Run:       runMigrate,
	Flags:     setupMigrateFlags,
	UsageLine: "migrate [options] <source-object> <dest-dataset>",
	Short:     "Migrates between versions of Noms",
	Long:      "",
	Nargs:     2,
}

func setupMigrateFlags() *flag.FlagSet {
	return flag.NewFlagSet("migrate", flag.ExitOnError)
}

func runMigrate(args []string) int {
	// TODO: verify source store is expected version
	// TODO: tests
	// TODO: support multiple source versions
	// TODO: support cyclic types
	// TODO: parallelize
	// TODO: incrementalize

	sourceStore, sourceObj, err := v7spec.GetPath(args[0])
	d.CheckError(err)
	defer sourceStore.Close()

	if sourceObj == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", args[0]))
	}

	sinkDataset, err := spec.GetDataset(args[1])
	d.CheckError(err)
	defer sinkDataset.Database().Close()

	sinkObj, err := migrateValue(sourceObj, sourceStore, sinkDataset.Database())
	d.CheckError(err)

	_, err = sinkDataset.CommitValue(sinkObj)
	d.CheckError(err)

	return 0
}

func migrateValue(source v7types.Value, sourceStore v7types.ValueReadWriter, sinkStore types.ValueReadWriter) (types.Value, error) {
	switch source := source.(type) {
	case v7types.Bool:
		return types.Bool(bool(source)), nil
	case v7types.Number:
		return types.Number(float64(source)), nil
	case v7types.String:
		return types.String(string(source)), nil
	case v7types.Blob:
		return types.NewStreamingBlob(source.Reader(), sourceStore), nil
	case v7types.List:
		vc := make(chan types.Value, 1024)
		lc := types.NewStreamingList(sinkStore, vc)
		for i := uint64(0); i < source.Len(); i++ {
			nv, err := migrateValue(source.Get(i), sourceStore, sinkStore)
			if err != nil {
				return nil, err
			}
			vc <- nv
		}
		close(vc)
		return <-lc, nil
	case v7types.Map:
		kvc := make(chan types.Value, 1024)
		mc := types.NewStreamingMap(sinkStore, kvc)
		var err error
		var nk, nv types.Value
		source.Iter(func(k, v v7types.Value) (stop bool) {
			nk, err = migrateValue(k, sourceStore, sinkStore)
			if err == nil {
				nv, err = migrateValue(v, sourceStore, sinkStore)
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
		return <-mc, nil
	case v7types.Set:
		vc := make(chan types.Value, 1024)
		sc := types.NewStreamingSet(sinkStore, vc)
		source.Iter(func(v v7types.Value) (stop bool) {
			nv, err := migrateValue(v, sourceStore, sinkStore)
			if err != nil {
				stop = true
			} else {
				vc <- nv
			}
			return
		})
		return <-sc, nil
	case v7types.Struct:
		t := migrateType(source.Type())
		sd := source.Type().Desc.(v7types.StructDesc)
		fields := make([]types.Value, 0, sd.Len())
		var err error
		sd.IterFields(func(name string, _ *v7types.Type) {
			if err == nil {
				var fv types.Value
				fv, err = migrateValue(source.Get(name), sourceStore, sinkStore)
				fields = append(fields, fv)
			}
		})
		if err != nil {
			return nil, err
		}
		return types.NewStructWithType(t, fields), nil
	}
	d.Chk.Fail("Unexpected type: %+v", source)
	return nil, nil
}

func migrateType(source *v7types.Type) *types.Type {
	migrateChildTypes := func() []*types.Type {
		sc := source.Desc.(v7types.CompoundDesc).ElemTypes
		dest := make([]*types.Type, 0, len(sc))
		for i := 0; i < len(sc); i++ {
			dest = append(dest, migrateType(sc[i]))
		}
		return dest
	}

	switch source.Kind() {
	case v7types.BoolKind:
		return types.MakePrimitiveType(types.BoolKind)
	case v7types.NumberKind:
		return types.MakePrimitiveType(types.NumberKind)
	case v7types.StringKind:
		return types.MakePrimitiveType(types.StringKind)
	case v7types.BlobKind:
		return types.MakePrimitiveType(types.BlobKind)
	case v7types.ValueKind:
		return types.MakePrimitiveType(types.ValueKind)
	case v7types.ListKind:
		return types.MakeListType(migrateChildTypes()[0])
	case v7types.MapKind:
		ct := migrateChildTypes()
		d.Chk.Equal(2, len(ct))
		return types.MakeMapType(ct[0], ct[1])
	case v7types.SetKind:
		return types.MakeSetType(migrateChildTypes()[0])
	case v7types.RefKind:
		return types.MakeRefType(migrateChildTypes()[0])
	case v7types.UnionKind:
		return types.MakeUnionType(migrateChildTypes()...)
	case v7types.TypeKind:
		return migrateType(source)
	case v7types.StructKind:
		sd := source.Desc.(v7types.StructDesc)
		names := make([]string, 0, sd.Len())
		typs := make([]*types.Type, 0, sd.Len())
		sd.IterFields(func(name string, t *v7types.Type) {
			names = append(names, name)
			typs = append(typs, migrateType(t))
		})
		return types.MakeStructType(sd.Name, names, typs)
	}
	d.Chk.Fail("notreached")
	return nil
}
