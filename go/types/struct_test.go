// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestGenericStructEquals(t *testing.T) {
	assert := assert.New(t)

	typ := MakeStructType("S1", FieldMap{
		"s": StringType,
		"x": BoolType,
	})

	s1 := NewStructWithType(typ, ValueSlice{String("hi"), Bool(true)})
	s2 := NewStructWithType(typ, ValueSlice{String("hi"), Bool(true)})

	assert.True(s1.Equals(s2))
	assert.True(s2.Equals(s1))
}

func TestGenericStructChunks(t *testing.T) {
	assert := assert.New(t)

	typ := MakeStructType("S1", FieldMap{"r": MakeRefType(BoolType)})

	b := Bool(true)

	s1 := NewStructWithType(typ, ValueSlice{NewRef(b)})

	assert.Len(s1.Chunks(), 1)
	assert.Equal(b.Hash(), s1.Chunks()[0].TargetHash())
}

func TestGenericStructNew(t *testing.T) {
	assert := assert.New(t)

	s := NewStruct("S2", StructData{"b": Bool(true), "o": String("hi")})
	assert.True(s.Get("b").Equals(Bool(true)))
	_, ok := s.MaybeGet("missing")
	assert.False(ok)

	s2 := NewStruct("S2", StructData{"b": Bool(false), "o": String("hi")})
	assert.True(s2.Get("b").Equals(Bool(false)))
	o, ok := s2.MaybeGet("o")
	assert.True(ok)
	assert.True(String("hi").Equals(o))

	typ := MakeStructType("S2", FieldMap{
		"b": BoolType,
		"o": StringType,
	})
	assert.Panics(func() { NewStructWithType(typ, nil) })
	assert.Panics(func() { NewStructWithType(typ, ValueSlice{String("hi")}) })
}

func TestGenericStructSet(t *testing.T) {
	assert := assert.New(t)

	s := NewStruct("S3", StructData{"b": Bool(true), "o": String("hi")})
	s2 := s.Set("b", Bool(false))

	assert.Panics(func() { s.Set("b", Number(1)) })
	assert.Panics(func() { s.Set("x", Number(1)) })

	s3 := s2.Set("b", Bool(true))
	assert.True(s.Equals(s3))
}

func TestStructDiff(t *testing.T) {
	assert := assert.New(t)

	assertDiff := func(expect []ValueChanged, s1, s2 Struct) {
		changes := make(chan ValueChanged)
		go func() {
			s1.Diff(s2, changes, nil)
			close(changes)
		}()
		i := 0
		for change := range changes {
			assert.Equal(expect[i], change)
			i++
		}
		assert.Equal(len(expect), i, "Wrong number of changes")
	}

	vc := func(ct DiffChangeType, fieldName string) ValueChanged {
		return ValueChanged{ChangeType: ct, V: String(fieldName)}
	}

	s1 := NewStruct("", StructData{"a": Bool(true), "b": String("hi"), "c": Number(4)})

	assertDiff([]ValueChanged{},
		s1, NewStruct("", StructData{"a": Bool(true), "b": String("hi"), "c": Number(4)}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "a"), vc(DiffChangeModified, "b")},
		s1, NewStruct("", StructData{"a": Bool(false), "b": String("bye"), "c": Number(4)}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "b"), vc(DiffChangeModified, "c")},
		s1, NewStruct("", StructData{"a": Bool(true), "b": String("bye"), "c": Number(5)}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "a"), vc(DiffChangeModified, "c")},
		s1, NewStruct("", StructData{"a": Bool(false), "b": String("hi"), "c": Number(10)}))

	assertDiff([]ValueChanged{vc(DiffChangeAdded, "a")},
		s1, NewStruct("NewType", StructData{"b": String("hi"), "c": Number(4)}))

	assertDiff([]ValueChanged{vc(DiffChangeAdded, "b")},
		s1, NewStruct("NewType", StructData{"a": Bool(true), "c": Number(4)}))

	assertDiff([]ValueChanged{vc(DiffChangeRemoved, "Z")},
		s1, NewStruct("NewType", StructData{"Z": Number(17), "a": Bool(true), "b": String("hi"), "c": Number(4)}))

	assertDiff([]ValueChanged{vc(DiffChangeAdded, "b"), vc(DiffChangeRemoved, "d")},
		s1, NewStruct("NewType", StructData{"a": Bool(true), "c": Number(4), "d": Number(5)}))

	s2 := NewStruct("", StructData{
		"a": NewList(Number(0), Number(1)),
		"b": NewMap(String("foo"), Bool(false), String("bar"), Bool(true)),
		"c": NewSet(Number(0), Number(1), String("foo")),
	})

	assertDiff([]ValueChanged{},
		s2, NewStruct("", StructData{
			"a": NewList(Number(0), Number(1)),
			"b": NewMap(String("foo"), Bool(false), String("bar"), Bool(true)),
			"c": NewSet(Number(0), Number(1), String("foo")),
		}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "a"), vc(DiffChangeModified, "b")},
		s2, NewStruct("", StructData{
			"a": NewList(Number(1), Number(1)),
			"b": NewMap(String("foo"), Bool(true), String("bar"), Bool(true)),
			"c": NewSet(Number(0), Number(1), String("foo")),
		}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "a"), vc(DiffChangeModified, "c")},
		s2, NewStruct("", StructData{
			"a": NewList(Number(0)),
			"b": NewMap(String("foo"), Bool(false), String("bar"), Bool(true)),
			"c": NewSet(Number(0), Number(2), String("foo")),
		}))

	assertDiff([]ValueChanged{vc(DiffChangeModified, "b"), vc(DiffChangeModified, "c")},
		s2, NewStruct("", StructData{
			"a": NewList(Number(0), Number(1)),
			"b": NewMap(String("boo"), Bool(false), String("bar"), Bool(true)),
			"c": NewSet(Number(0), Number(1), String("bar")),
		}))
}

func TestEscStructField(t *testing.T) {
	assert := assert.New(t)
	cases := []string{
		"a", "a",
		"AaZz19_", "AaZz19_",
		"Q", "Q51",
		"AQ1", "AQ511",
		"INSPECTIONQ20STATUS", "INSPECTIONQ5120STATUS",
		"$", "Q24",
		"_content", "Q5Fcontent",
		"Few ¢ents Short", "FewQ20QC2A2entsQ20Short",
		"💩", "QF09F92A9",
		"https://picasaweb.google.com/data", "httpsQ3AQ2FQ2FpicasawebQ2EgoogleQ2EcomQ2Fdata",
	}

	for i := 0; i < len(cases); i += 2 {
		orig, expected := cases[i], cases[i+1]
		assert.Equal(expected, EscapeStructField(orig))
	}
}

func TestCycles(t *testing.T) {
	// Success is this not recursing infinitely and blowing the stack
	fileType := MakeStructType("File", FieldMap{"data": BlobType})
	directoryType := MakeStructType("Directory", FieldMap{"entries": MakeMapType(StringType, MakeCycleType(1))})
	inodeType := MakeStructType("Inode", FieldMap{"contents": MakeUnionType(directoryType, fileType)})
	fsType := MakeStructType("Filesystem", FieldMap{"root": inodeType})

	rootDir := NewStructWithType(directoryType, ValueSlice{NewMap()})
	rootInode := NewStruct("Inode", StructData{"contents": rootDir})
	NewStructWithType(fsType, ValueSlice{rootInode})
}
