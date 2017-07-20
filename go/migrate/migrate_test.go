// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package migrate

import (
	"bytes"
	"testing"

	oldchunks "gopkg.in/attic-labs/noms.v7/go/chunks"
	olddatas "gopkg.in/attic-labs/noms.v7/go/datas"
	oldtypes "gopkg.in/attic-labs/noms.v7/go/types"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

func TestConv(t *testing.T) {
	of := oldchunks.NewMemoryStoreFactory()
	nf := chunks.NewMemoryStoreFactory()
	sourceStore := olddatas.NewDatabase(of.CreateStore(""))
	sinkStore := datas.NewDatabase(nf.CreateStore(""))

	test := func(expected types.Value, source oldtypes.Value) {
		actual, err := Conv(source, sourceStore, sinkStore)
		assert.NoError(t, err)
		assert.True(t, actual.Equals(expected))
	}

	test(types.Bool(true), oldtypes.Bool(true))
	test(types.Bool(false), oldtypes.Bool(false))

	test(types.Number(-42), oldtypes.Number(-42))
	test(types.Number(-1.23456789), oldtypes.Number(-1.23456789))
	test(types.Number(0), oldtypes.Number(0))
	test(types.Number(1.23456789), oldtypes.Number(1.23456789))
	test(types.Number(42), oldtypes.Number(42))

	test(types.String(""), oldtypes.String(""))
	test(types.String("Hello World"), oldtypes.String("Hello World"))
	test(types.String("💩"), oldtypes.String("💩"))

	test(types.NewBlob(bytes.NewBuffer([]byte{})), oldtypes.NewBlob(bytes.NewBuffer([]byte{})))
	test(types.NewBlob(bytes.NewBufferString("hello")), oldtypes.NewBlob(bytes.NewBufferString("hello")))

	test(types.NewList(), oldtypes.NewList())
	test(types.NewList(types.Bool(true)), oldtypes.NewList(oldtypes.Bool(true)))
	test(types.NewList(types.Bool(true), types.String("hi")), oldtypes.NewList(oldtypes.Bool(true), oldtypes.String("hi")))

	test(types.NewSet(), oldtypes.NewSet())
	test(types.NewSet(types.Bool(true)), oldtypes.NewSet(oldtypes.Bool(true)))
	test(types.NewSet(types.Bool(true), types.String("hi")), oldtypes.NewSet(oldtypes.Bool(true), oldtypes.String("hi")))

	test(types.NewMap(), oldtypes.NewMap())
	test(types.NewMap(types.Bool(true), types.String("hi")), oldtypes.NewMap(oldtypes.Bool(true), oldtypes.String("hi")))

	test(types.NewStruct("", types.StructData{}), oldtypes.NewStruct("", oldtypes.StructData{}))
	test(types.NewStruct("xyz", types.StructData{}), oldtypes.NewStruct("xyz", oldtypes.StructData{}))
	test(types.NewStruct("T", types.StructData{}), oldtypes.NewStruct("T", oldtypes.StructData{}))

	test(types.NewStruct("T", types.StructData{
		"x": types.Number(42),
		"s": types.String("hi"),
		"b": types.Bool(false),
	}), oldtypes.NewStruct("T", oldtypes.StructData{
		"x": oldtypes.Number(42),
		"s": oldtypes.String("hi"),
		"b": oldtypes.Bool(false),
	}))

	test(
		types.NewStruct("", types.StructData{
			"a": types.Number(42),
		}),
		oldtypes.NewStruct("", oldtypes.StructData{
			"a": oldtypes.Number(42),
		}),
	)

	test(
		types.NewStruct("", types.StructData{
			"a": types.NewList(),
		}),
		oldtypes.NewStruct("", oldtypes.StructData{
			"a": oldtypes.NewList(),
		}),
	)

	r := sourceStore.WriteValue(oldtypes.Number(123))
	test(types.NewRef(types.Number(123)), r)
	v := sinkStore.ReadValue(types.Number(123).Hash())
	assert.True(t, types.Number(123).Equals(v))

	// Types
	test(types.BoolType, oldtypes.BoolType)
	test(types.NumberType, oldtypes.NumberType)
	test(types.StringType, oldtypes.StringType)
	test(types.BlobType, oldtypes.BlobType)
	test(types.TypeType, oldtypes.TypeType)
	test(types.ValueType, oldtypes.ValueType)
	test(types.MakeListType(types.NumberType), oldtypes.MakeListType(oldtypes.NumberType))
	test(types.TypeOf(types.MakeListType(types.NumberType)), oldtypes.TypeOf(oldtypes.MakeListType(oldtypes.NumberType)))

	test(types.MakeListType(types.NumberType), oldtypes.MakeListType(oldtypes.NumberType))
	test(types.MakeSetType(types.NumberType), oldtypes.MakeSetType(oldtypes.NumberType))
	test(types.MakeRefType(types.NumberType), oldtypes.MakeRefType(oldtypes.NumberType))
	test(types.MakeMapType(types.NumberType, types.StringType), oldtypes.MakeMapType(oldtypes.NumberType, oldtypes.StringType))
	test(types.MakeUnionType(), oldtypes.MakeUnionType())
	test(types.MakeUnionType(types.StringType, types.BoolType), oldtypes.MakeUnionType(oldtypes.StringType, oldtypes.BoolType))

	commit := types.MakeStructType("Commit",
		types.StructField{
			Name: "parents",
			Type: types.MakeSetType(types.MakeRefType(types.MakeStructType("Commit",
				types.StructField{
					Name: "parents",
					Type: types.MakeSetType(types.MakeRefType(types.MakeCycleType("Commit"))),
				},
				types.StructField{
					Name: "value",
					Type: types.MakeUnionType(types.NumberType, types.StringType),
				},
			))),
		},
		types.StructField{
			Name: "value",
			Type: types.StringType,
		},
	)

	commit7 := oldtypes.MakeStructType("Commit",
		oldtypes.StructField{
			Name: "parents",
			Type: oldtypes.MakeSetType(oldtypes.MakeRefType(oldtypes.MakeStructType("Commit",
				oldtypes.StructField{
					Name: "parents",
					Type: oldtypes.MakeSetType(oldtypes.MakeRefType(oldtypes.MakeCycleType("Commit"))),
				},
				oldtypes.StructField{
					Name: "value",
					Type: oldtypes.MakeUnionType(oldtypes.NumberType, oldtypes.StringType),
				},
			))),
		},
		oldtypes.StructField{
			Name: "value",
			Type: oldtypes.StringType,
		},
	)
	test(commit, commit7)
}
