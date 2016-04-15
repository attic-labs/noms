package value

import (
	"bytes"
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

func assertWriteEqual(t *testing.T, expected string, v types.Value) {
	assert := assert.New(t)
	var buf bytes.Buffer
	w := &valueWriter{w: &buf}
	w.Write(v)
	assert.Equal(expected, buf.String())
}

func assertWriteTaggedEqual(t *testing.T, expected string, v types.Value) {
	assert := assert.New(t)
	var buf bytes.Buffer
	w := &valueWriter{w: &buf}
	w.WriteTagged(v)
	assert.Equal(expected, buf.String())
}

func TestWritePrimitiveValues(t *testing.T) {
	assertWriteEqual(t, "true", types.Bool(true))
	assertWriteEqual(t, "false", types.Bool(false))

	assertWriteEqual(t, "0", types.Uint8(0))
	assertWriteEqual(t, "0", types.Uint16(0))
	assertWriteEqual(t, "0", types.Uint32(0))
	assertWriteEqual(t, "0", types.Uint64(0))
	assertWriteEqual(t, "0", types.Int8(0))
	assertWriteEqual(t, "0", types.Int16(0))
	assertWriteEqual(t, "0", types.Int32(0))
	assertWriteEqual(t, "0", types.Int64(0))
	assertWriteEqual(t, "0", types.Float32(0))
	assertWriteEqual(t, "0", types.Float64(0))

	assertWriteEqual(t, "42", types.Uint8(42))
	assertWriteEqual(t, "42", types.Uint16(42))
	assertWriteEqual(t, "42", types.Uint32(42))
	assertWriteEqual(t, "42", types.Uint64(42))
	assertWriteEqual(t, "42", types.Int8(42))
	assertWriteEqual(t, "42", types.Int16(42))
	assertWriteEqual(t, "42", types.Int32(42))
	assertWriteEqual(t, "42", types.Int64(42))
	assertWriteEqual(t, "42", types.Float32(42))
	assertWriteEqual(t, "42", types.Float64(42))

	assertWriteEqual(t, "-42", types.Int8(-42))
	assertWriteEqual(t, "-42", types.Int16(-42))
	assertWriteEqual(t, "-42", types.Int32(-42))
	assertWriteEqual(t, "-42", types.Int64(-42))
	assertWriteEqual(t, "-42", types.Float32(-42))
	assertWriteEqual(t, "-42", types.Float64(-42))

	assertWriteEqual(t, "3.1415927", types.Float32(3.1415926535))
	assertWriteEqual(t, "3.1415926535", types.Float64(3.1415926535))

	assertWriteEqual(t, "314159.25", types.Float32(3.1415926535e5))
	assertWriteEqual(t, "314159.26535", types.Float64(3.1415926535e5))

	assertWriteEqual(t, "3.1415925e+20", types.Float32(3.1415926535e20))
	assertWriteEqual(t, "3.1415926535e+20", types.Float64(3.1415926535e20))

	assertWriteEqual(t, `"abc"`, types.NewString("abc"))
	assertWriteEqual(t, `" "`, types.NewString(" "))
	assertWriteEqual(t, `"\t"`, types.NewString("\t"))
	assertWriteEqual(t, `"\t"`, types.NewString("	"))
	assertWriteEqual(t, `"\n"`, types.NewString("\n"))
	assertWriteEqual(t, `"\n"`, types.NewString(`
`))
	assertWriteEqual(t, `"\r"`, types.NewString("\r"))
	assertWriteEqual(t, `"\r\n"`, types.NewString("\r\n"))
	assertWriteEqual(t, `"\xff"`, types.NewString("\xff"))
	assertWriteEqual(t, `"💩"`, types.NewString("\xf0\x9f\x92\xa9"))
	assertWriteEqual(t, `"💩"`, types.NewString("💩"))
	assertWriteEqual(t, `"\a"`, types.NewString("\007"))
	assertWriteEqual(t, `"☺"`, types.NewString("\u263a"))
}

func TestWriteRef(t *testing.T) {
	cs := chunks.NewTestStore()
	ds := datas.NewDataStore(cs)

	x := types.Int32(42)
	rv := ds.WriteValue(x)
	assertWriteEqual(t, "sha1-c56efb6071a71743b826f2e10df26761549df9c2", rv)
	assertWriteTaggedEqual(t, "Ref<Int32>(sha1-c56efb6071a71743b826f2e10df26761549df9c2)", rv)
}

func TestWriteCollections(t *testing.T) {
	lt := types.MakeListType(types.Float64Type)
	l := types.NewTypedList(lt, types.Float64(0), types.Float64(1), types.Float64(2), types.Float64(3))
	assertWriteEqual(t, `[0, 1, 2, 3]`, l)
	assertWriteTaggedEqual(t, `List<Float64>([0, 1, 2, 3])`, l)

	st := types.MakeSetType(types.Int8Type)
	s := types.NewTypedSet(st, types.Int8(0), types.Int8(1), types.Int8(2), types.Int8(3))
	assertWriteEqual(t, `{0, 1, 2, 3}`, s)
	assertWriteTaggedEqual(t, `Set<Int8>({0, 1, 2, 3})`, s)

	mt := types.MakeMapType(types.Int32Type, types.BoolType)
	m := types.NewTypedMap(mt, types.Int32(0), types.Bool(false), types.Int32(1), types.Bool(true))
	assertWriteEqual(t, `{0: false, 1: true}`, m)
	assertWriteTaggedEqual(t, `Map<Int32, Bool>({0: false, 1: true})`, m)
}

func TestWriteNested(t *testing.T) {
	lt := types.MakeListType(types.Float64Type)
	l := types.NewTypedList(lt, types.Float64(0), types.Float64(1))
	l2 := types.NewTypedList(lt, types.Float64(2), types.Float64(3))

	st := types.MakeSetType(types.StringType)
	s := types.NewTypedSet(st, types.NewString("a"), types.NewString("b"))
	s2 := types.NewTypedSet(st, types.NewString("c"), types.NewString("d"))

	mt := types.MakeMapType(st, lt)
	m := types.NewTypedMap(mt, s, l, s2, l2)
	assertWriteEqual(t, `{{"c", "d"}: [2, 3], {"a", "b"}: [0, 1]}`, m)
	assertWriteTaggedEqual(t, `Map<Set<String>, List<Float64>>({{"c", "d"}: [2, 3], {"a", "b"}: [0, 1]})`, m)

}

func TestWriteStruct(t *testing.T) {
	pkg := types.NewPackage([]types.Type{
		types.MakeStructType("S1", []types.Field{
			types.Field{Name: "x", T: types.Int32Type, Optional: false},
			types.Field{Name: "y", T: types.Int32Type, Optional: true},
		}, types.Choices{}),
	}, []ref.Ref{})
	typeDef := pkg.Types()[0]
	types.RegisterPackage(&pkg)
	typ := types.MakeType(pkg.Ref(), 0)

	str := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(1),
	})
	assertWriteEqual(t, `S1 {x: 1}`, str)
	assertWriteTaggedEqual(t, `Struct<S1, sha1-bdd35d6fe5b89487d71d0ec27c1a6c79a0261baa, 0>({x: 1})`, str)

	str2 := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(2),
		"y": types.Int32(3),
	})
	assertWriteEqual(t, `S1 {x: 2, y: 3}`, str2)
	assertWriteTaggedEqual(t, `Struct<S1, sha1-bdd35d6fe5b89487d71d0ec27c1a6c79a0261baa, 0>({x: 2, y: 3})`, str2)
}

func TestWriteStructWithUnion(t *testing.T) {
	pkg := types.NewPackage([]types.Type{
		types.MakeStructType("S2", []types.Field{}, types.Choices{
			types.Field{Name: "x", T: types.Int32Type, Optional: false},
			types.Field{Name: "y", T: types.Int32Type, Optional: false},
		}),
	}, []ref.Ref{})
	typeDef := pkg.Types()[0]
	types.RegisterPackage(&pkg)
	typ := types.MakeType(pkg.Ref(), 0)

	str := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(1),
	})
	assertWriteEqual(t, `S2 {x: 1}`, str)
	assertWriteTaggedEqual(t, `Struct<S2, sha1-13e3f926c03c637bc474442a10af9023b24010f8, 0>({x: 1})`, str)

	str2 := types.NewStruct(typ, typeDef, map[string]types.Value{
		"y": types.Int32(2),
	})
	assertWriteEqual(t, `S2 {y: 2}`, str2)
	assertWriteTaggedEqual(t, `Struct<S2, sha1-13e3f926c03c637bc474442a10af9023b24010f8, 0>({y: 2})`, str2)
}

func TestWriteListOfStruct(t *testing.T) {
	pkg := types.NewPackage([]types.Type{
		types.MakeStructType("S3", []types.Field{}, types.Choices{
			types.Field{Name: "x", T: types.Int32Type, Optional: false},
		}),
	}, []ref.Ref{})
	typeDef := pkg.Types()[0]
	types.RegisterPackage(&pkg)
	typ := types.MakeType(pkg.Ref(), 0)

	str1 := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(1),
	})
	str2 := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(2),
	})
	str3 := types.NewStruct(typ, typeDef, map[string]types.Value{
		"x": types.Int32(3),
	})
	lt := types.MakeListType(typ)
	l := types.NewTypedList(lt, str1, str2, str3)
	assertWriteEqual(t, `[S3 {x: 1}, S3 {x: 2}, S3 {x: 3}]`, l)
	assertWriteTaggedEqual(t, `List<Struct<S3, sha1-543f7124883ace7da7fccaed6d5cfc31598020f1, 0>>([S3 {x: 1}, S3 {x: 2}, S3 {x: 3}])`, l)
}

func TestWriteEnum(t *testing.T) {
	pkg := types.NewPackage([]types.Type{
		types.MakeEnumType("Color", "red", "green", "blue"),
	}, []ref.Ref{})
	types.RegisterPackage(&pkg)
	typ := types.MakeType(pkg.Ref(), 0)

	assertWriteEqual(t, "red", types.NewEnum(0, typ))
	assertWriteTaggedEqual(t, "Enum<Color, sha1-51b66eaa0827d76d1618c8d4e7e42215d00d6642, 0>(red)", types.NewEnum(0, typ))
	assertWriteEqual(t, "green", types.NewEnum(1, typ))
	assertWriteTaggedEqual(t, "Enum<Color, sha1-51b66eaa0827d76d1618c8d4e7e42215d00d6642, 0>(green)", types.NewEnum(1, typ))
	assertWriteEqual(t, "blue", types.NewEnum(2, typ))
	assertWriteTaggedEqual(t, "Enum<Color, sha1-51b66eaa0827d76d1618c8d4e7e42215d00d6642, 0>(blue)", types.NewEnum(2, typ))
}

func TestWriteBlob(t *testing.T) {
	assertWriteEqual(t, "", types.NewEmptyBlob())
	assertWriteTaggedEqual(t, "Blob()", types.NewEmptyBlob())

	b1 := types.NewBlob(bytes.NewBuffer([]byte{0x01}))
	assertWriteEqual(t, "AQ==", b1)
	assertWriteTaggedEqual(t, "Blob(AQ==)", b1)

	b2 := types.NewBlob(bytes.NewBuffer([]byte{0x01, 0x02}))
	assertWriteEqual(t, "AQI=", b2)
	assertWriteTaggedEqual(t, "Blob(AQI=)", b2)

	b3 := types.NewBlob(bytes.NewBuffer([]byte{0x01, 0x02, 0x03}))
	assertWriteEqual(t, "AQID", b3)
	assertWriteTaggedEqual(t, "Blob(AQID)", b3)

	bs := make([]byte, 256)
	for i := range bs {
		bs[i] = byte(i)
	}
	b4 := types.NewBlob(bytes.NewBuffer(bs))
	assertWriteEqual(t, "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8gISIjJCUmJygpKissLS4vMDEyMzQ1Njc4OTo7PD0+P0BBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWltcXV5fYGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6e3x9fn+AgYKDhIWGh4iJiouMjY6PkJGSk5SVlpeYmZqbnJ2en6ChoqOkpaanqKmqq6ytrq+wsbKztLW2t7i5uru8vb6/wMHCw8TFxsfIycrLzM3Oz9DR0tPU1dbX2Nna29zd3t/g4eLj5OXm5+jp6uvs7e7v8PHy8/T19vf4+fr7/P3+/w==", b4)
	assertWriteTaggedEqual(t, "Blob(AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8gISIjJCUmJygpKissLS4vMDEyMzQ1Njc4OTo7PD0+P0BBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWltcXV5fYGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6e3x9fn+AgYKDhIWGh4iJiouMjY6PkJGSk5SVlpeYmZqbnJ2en6ChoqOkpaanqKmqq6ytrq+wsbKztLW2t7i5uru8vb6/wMHCw8TFxsfIycrLzM3Oz9DR0tPU1dbX2Nna29zd3t/g4eLj5OXm5+jp6uvs7e7v8PHy8/T19vf4+fr7/P3+/w==)", b4)
}

func TestWriteListOfBlob(t *testing.T) {
	lt := types.MakeListType(types.BlobType)
	b1 := types.NewBlob(bytes.NewBuffer([]byte{0x01}))
	b2 := types.NewBlob(bytes.NewBuffer([]byte{0x02}))
	l := types.NewTypedList(lt, b1, types.NewEmptyBlob(), b2)
	assertWriteEqual(t, "[AQ==, , Ag==]", l)
	assertWriteTaggedEqual(t, "List<Blob>([AQ==, , Ag==])", l)
}

func TestWriteListOfEnum(t *testing.T) {
	pkg := types.NewPackage([]types.Type{
		types.MakeEnumType("Color", "red", "green", "blue"),
	}, []ref.Ref{})
	types.RegisterPackage(&pkg)
	typ := types.MakeType(pkg.Ref(), 0)
	lt := types.MakeListType(typ)
	l := types.NewTypedList(lt, types.NewEnum(0, typ), types.NewEnum(1, typ), types.NewEnum(2, typ))
	assertWriteEqual(t, "[red, green, blue]", l)
	assertWriteTaggedEqual(t, "List<Enum<Color, sha1-51b66eaa0827d76d1618c8d4e7e42215d00d6642, 0>>([red, green, blue])", l)
}

func TestWriteType(t *testing.T) {
	assertWriteEqual(t, "Bool", types.BoolType)
	assertWriteEqual(t, "Blob", types.BlobType)
	assertWriteEqual(t, "String", types.StringType)

	assertWriteEqual(t, "Int8", types.Int8Type)
	assertWriteEqual(t, "Int16", types.Int16Type)
	assertWriteEqual(t, "Int32", types.Int32Type)
	assertWriteEqual(t, "Int64", types.Int64Type)
	assertWriteEqual(t, "Uint8", types.Uint8Type)
	assertWriteEqual(t, "Uint16", types.Uint16Type)
	assertWriteEqual(t, "Uint32", types.Uint32Type)
	assertWriteEqual(t, "Uint64", types.Uint64Type)
	assertWriteEqual(t, "Float32", types.Float32Type)
	assertWriteEqual(t, "Float64", types.Float64Type)

	assertWriteEqual(t, "List<Int8>", types.MakeListType(types.Int8Type))
	assertWriteEqual(t, "Set<Int16>", types.MakeSetType(types.Int16Type))
	assertWriteEqual(t, "Ref<Int32>", types.MakeRefType(types.Int32Type))
	assertWriteEqual(t, "Map<Int64, String>", types.MakeMapType(types.Int64Type, types.StringType))

	pkg := types.NewPackage([]types.Type{
		types.MakeEnumType("Color", "red", "green", "blue"),
		types.MakeStructType("Str", []types.Field{
			types.Field{Name: "c", T: types.MakeType(ref.Ref{}, 0), Optional: false},
			types.Field{Name: "o", T: types.StringType, Optional: true},
		}, types.Choices{
			types.Field{Name: "x", T: types.MakeType(ref.Ref{}, 1), Optional: false},
			types.Field{Name: "y", T: types.BoolType, Optional: false},
		}),
	}, []ref.Ref{})
	types.RegisterPackage(&pkg)
	et := types.MakeType(pkg.Ref(), 0)
	st := types.MakeType(pkg.Ref(), 1)

	assertWriteEqual(t, "Enum<Color, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 0>", et)
	assertWriteTaggedEqual(t, "Type(Enum<Color, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 0>)", et)
	assertWriteEqual(t, "Struct<Str, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 1>", st)
	assertWriteTaggedEqual(t, "Type(Struct<Str, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 1>)", st)

	eTypeDef := pkg.Types()[0]
	assertWriteEqual(t, "enum Color {red green blue}", eTypeDef)
	assertWriteTaggedEqual(t, "Type(enum Color {red green blue})", eTypeDef)

	sTypeDef := pkg.Types()[1]
	assertWriteEqual(t, "struct Str {c: Enum<Color, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 0> o: optional String union {x: Struct<Str, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 1> y: Bool}}", sTypeDef)
	assertWriteTaggedEqual(t, "Type(struct Str {c: Enum<Color, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 0> o: optional String union {x: Struct<Str, sha1-9323c4c8d8a5745550b914fb01c8641ab42f121a, 1> y: Bool}})", sTypeDef)
}

func TestWriteTaggedPrimitiveValues(t *testing.T) {
	assertWriteEqual(t, "true", types.Bool(true))
	assertWriteEqual(t, "false", types.Bool(false))

	assertWriteTaggedEqual(t, "Uint8(0)", types.Uint8(0))
	assertWriteTaggedEqual(t, "Uint16(0)", types.Uint16(0))
	assertWriteTaggedEqual(t, "Uint32(0)", types.Uint32(0))
	assertWriteTaggedEqual(t, "Uint64(0)", types.Uint64(0))
	assertWriteTaggedEqual(t, "Int8(0)", types.Int8(0))
	assertWriteTaggedEqual(t, "Int16(0)", types.Int16(0))
	assertWriteTaggedEqual(t, "Int32(0)", types.Int32(0))
	assertWriteTaggedEqual(t, "Int64(0)", types.Int64(0))
	assertWriteTaggedEqual(t, "Float32(0)", types.Float32(0))
	assertWriteTaggedEqual(t, "Float64(0)", types.Float64(0))

	assertWriteTaggedEqual(t, "Uint8(42)", types.Uint8(42))
	assertWriteTaggedEqual(t, "Uint16(42)", types.Uint16(42))
	assertWriteTaggedEqual(t, "Uint32(42)", types.Uint32(42))
	assertWriteTaggedEqual(t, "Uint64(42)", types.Uint64(42))
	assertWriteTaggedEqual(t, "Int8(42)", types.Int8(42))
	assertWriteTaggedEqual(t, "Int16(42)", types.Int16(42))
	assertWriteTaggedEqual(t, "Int32(42)", types.Int32(42))
	assertWriteTaggedEqual(t, "Int64(42)", types.Int64(42))
	assertWriteTaggedEqual(t, "Float32(42)", types.Float32(42))
	assertWriteTaggedEqual(t, "Float64(42)", types.Float64(42))

	assertWriteTaggedEqual(t, "Int8(-42)", types.Int8(-42))
	assertWriteTaggedEqual(t, "Int16(-42)", types.Int16(-42))
	assertWriteTaggedEqual(t, "Int32(-42)", types.Int32(-42))
	assertWriteTaggedEqual(t, "Int64(-42)", types.Int64(-42))
	assertWriteTaggedEqual(t, "Float32(-42)", types.Float32(-42))
	assertWriteTaggedEqual(t, "Float64(-42)", types.Float64(-42))

	assertWriteTaggedEqual(t, "Float32(3.1415927)", types.Float32(3.1415926535))
	assertWriteTaggedEqual(t, "Float64(3.1415926535)", types.Float64(3.1415926535))

	assertWriteTaggedEqual(t, "Float32(314159.25)", types.Float32(3.1415926535e5))
	assertWriteTaggedEqual(t, "Float64(314159.26535)", types.Float64(3.1415926535e5))

	assertWriteTaggedEqual(t, "Float32(3.1415925e+20)", types.Float32(3.1415926535e20))
	assertWriteTaggedEqual(t, "Float64(3.1415926535e+20)", types.Float64(3.1415926535e20))

	assertWriteTaggedEqual(t, `"abc"`, types.NewString("abc"))
	assertWriteTaggedEqual(t, `" "`, types.NewString(" "))
	assertWriteTaggedEqual(t, `"\t"`, types.NewString("\t"))
	assertWriteTaggedEqual(t, `"\t"`, types.NewString("	"))
	assertWriteTaggedEqual(t, `"\n"`, types.NewString("\n"))
	assertWriteTaggedEqual(t, `"\n"`, types.NewString(`
`))
	assertWriteTaggedEqual(t, `"\r"`, types.NewString("\r"))
	assertWriteTaggedEqual(t, `"\r\n"`, types.NewString("\r\n"))
	assertWriteTaggedEqual(t, `"\xff"`, types.NewString("\xff"))
	assertWriteTaggedEqual(t, `"💩"`, types.NewString("\xf0\x9f\x92\xa9"))
	assertWriteTaggedEqual(t, `"💩"`, types.NewString("💩"))
	assertWriteTaggedEqual(t, `"\a"`, types.NewString("\007"))
	assertWriteTaggedEqual(t, `"☺"`, types.NewString("\u263a"))
}

func TestWriteTaggedType(t *testing.T) {
	assertWriteTaggedEqual(t, "Type(Bool)", types.BoolType)
	assertWriteTaggedEqual(t, "Type(Blob)", types.BlobType)
	assertWriteTaggedEqual(t, "Type(String)", types.StringType)

	assertWriteTaggedEqual(t, "Type(Int8)", types.Int8Type)
	assertWriteTaggedEqual(t, "Type(Int16)", types.Int16Type)
	assertWriteTaggedEqual(t, "Type(Int32)", types.Int32Type)
	assertWriteTaggedEqual(t, "Type(Int64)", types.Int64Type)
	assertWriteTaggedEqual(t, "Type(Uint8)", types.Uint8Type)
	assertWriteTaggedEqual(t, "Type(Uint16)", types.Uint16Type)
	assertWriteTaggedEqual(t, "Type(Uint32)", types.Uint32Type)
	assertWriteTaggedEqual(t, "Type(Uint64)", types.Uint64Type)
	assertWriteTaggedEqual(t, "Type(Float32)", types.Float32Type)
	assertWriteTaggedEqual(t, "Type(Float64)", types.Float64Type)

	assertWriteTaggedEqual(t, "Type(List<Int8>)", types.MakeListType(types.Int8Type))
	assertWriteTaggedEqual(t, "Type(Set<Int16>)", types.MakeSetType(types.Int16Type))
	assertWriteTaggedEqual(t, "Type(Ref<Int32>)", types.MakeRefType(types.Int32Type))
	assertWriteTaggedEqual(t, "Type(Map<Int64, String>)", types.MakeMapType(types.Int64Type, types.StringType))

}
