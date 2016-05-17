package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validateType(t *Type, v Value) {
	assertType(t, v)
}

func assertInvalid(tt *testing.T, t *Type, v Value) {
	assert := assert.New(tt)
	assert.Panics(func() {
		assertType(t, v)
	})
}

func assertAll(tt *testing.T, t *Type, v Value) {
	allTypes := []*Type{
		BoolType,
		NumberType,
		StringType,
		BlobType,
		TypeType,
		ValueType,
	}

	for _, at := range allTypes {
		if at == ValueType || t.Equals(at) {
			validateType(at, v)
		} else {
			assertInvalid(tt, at, v)
		}
	}
}

func TestAssertTypePrimitives(t *testing.T) {
	validateType(BoolType, Bool(true))
	validateType(BoolType, Bool(false))
	validateType(NumberType, Number(42))
	validateType(StringType, NewString("abc"))

	assertInvalid(t, BoolType, Number(1))
	assertInvalid(t, BoolType, NewString("abc"))
	assertInvalid(t, NumberType, Bool(true))
	assertInvalid(t, StringType, Number(42))
}

func TestAssertTypeValue(t *testing.T) {
	validateType(ValueType, Bool(true))
	validateType(ValueType, Number(1))
	validateType(ValueType, NewString("abc"))
	l := NewList(Number(0), Number(1), Number(2), Number(3))
	validateType(ValueType, l)
}

func TestAssertTypeBlob(t *testing.T) {
	blob := NewBlob(bytes.NewBuffer([]byte{0x00, 0x01}))
	assertAll(t, BlobType, blob)
}

func TestAssertTypeList(tt *testing.T) {
	listOfNumberType := MakeListType(NumberType)
	l := NewList(Number(0), Number(1), Number(2), Number(3))
	validateType(listOfNumberType, l)
	assertAll(tt, listOfNumberType, l)
	validateType(MakeListType(ValueType), l)
}

func TestAssertTypeMap(tt *testing.T) {
	mapOfNumberToStringType := MakeMapType(NumberType, StringType)
	m := NewMap(Number(0), NewString("a"), Number(2), NewString("b"))
	validateType(mapOfNumberToStringType, m)
	assertAll(tt, mapOfNumberToStringType, m)
	validateType(MakeMapType(ValueType, ValueType), m)
}

func TestAssertTypeSet(tt *testing.T) {
	setOfNumberType := MakeSetType(NumberType)
	s := NewSet(Number(0), Number(1), Number(2), Number(3))
	validateType(setOfNumberType, s)
	assertAll(tt, setOfNumberType, s)
	validateType(MakeSetType(ValueType), s)
}

func TestAssertTypeType(tt *testing.T) {
	t := MakeSetType(NumberType)
	validateType(TypeType, t)
	assertAll(tt, TypeType, t)
	validateType(ValueType, t)
}

func TestAssertTypeStruct(tt *testing.T) {
	t := MakeStructType("Struct", TypeMap{
		"x": BoolType,
	})

	v := NewStruct("Struct", structData{"x": Bool(true)})
	validateType(t, v)
	assertAll(tt, t, v)
	validateType(ValueType, v)
}

func TestAssertTypeUnion(tt *testing.T) {
	validateType(MakeUnionType(NumberType), Number(42))
	validateType(MakeUnionType(NumberType, StringType), Number(42))
	validateType(MakeUnionType(NumberType, StringType), NewString("hi"))
	validateType(MakeUnionType(NumberType, StringType, BoolType), Number(555))
	validateType(MakeUnionType(NumberType, StringType, BoolType), NewString("hi"))
	validateType(MakeUnionType(NumberType, StringType, BoolType), Bool(true))

	lt := MakeListType(MakeUnionType(NumberType, StringType))
	validateType(lt, NewList(Number(1), NewString("hi"), Number(2), NewString("bye")))

	st := MakeSetType(StringType)
	validateType(MakeUnionType(st, NumberType), Number(42))
	validateType(MakeUnionType(st, NumberType), NewSet(NewString("a"), NewString("b")))

	assertInvalid(tt, MakeUnionType(), Number(42))
	assertInvalid(tt, MakeUnionType(StringType), Number(42))
	assertInvalid(tt, MakeUnionType(StringType, BoolType), Number(42))
	assertInvalid(tt, MakeUnionType(st, StringType), Number(42))
	assertInvalid(tt, MakeUnionType(st, NumberType), NewSet(Number(1), Number(2)))
}

func TestAssertTypeEmptyListUnion(tt *testing.T) {
	lt := MakeListType(MakeUnionType())
	validateType(lt, NewList())
}

func TestAssertTypeEmptyList(tt *testing.T) {
	lt := MakeListType(NumberType)
	validateType(lt, NewList())

	// List<> not a subtype of List<Number>
	assertInvalid(tt, MakeListType(MakeUnionType()), NewList(Number(1)))
}

func TestAssertTypeEmptySet(tt *testing.T) {
	st := MakeSetType(NumberType)
	validateType(st, NewSet())

	// Set<> not a subtype of Set<Number>
	assertInvalid(tt, MakeSetType(MakeUnionType()), NewSet(Number(1)))
}

func TestAssertTypeEmptyMap(tt *testing.T) {
	mt := MakeMapType(NumberType, StringType)
	validateType(mt, NewMap())

	// Map<> not a subtype of Map<Number, Number>
	assertInvalid(tt, MakeMapType(MakeUnionType(), MakeUnionType()), NewMap(Number(1), Number(2)))
}

func TestAssertTypeStructSubtypeByName(tt *testing.T) {
	namedT := MakeStructType("Name", TypeMap{"x": NumberType})
	anonT := MakeStructType("", TypeMap{"x": NumberType})
	namedV := NewStruct("Name", structData{"x": Number(42)})
	name2V := NewStruct("foo", structData{"x": Number(42)})
	anonV := NewStruct("", structData{"x": Number(42)})

	validateType(namedT, namedV)
	assertInvalid(tt, namedT, name2V)
	assertInvalid(tt, namedT, anonV)

	validateType(anonT, namedV)
	validateType(anonT, name2V)
	validateType(anonT, anonV)
}

func TestAssertTypeStructSubtypeExtraFields(tt *testing.T) {
	at := MakeStructType("", TypeMap{})
	bt := MakeStructType("", TypeMap{"x": NumberType})
	ct := MakeStructType("", TypeMap{"x": NumberType, "s": StringType})
	av := NewStruct("", structData{})
	bv := NewStruct("", structData{"x": Number(1)})
	cv := NewStruct("", structData{"x": Number(2), "s": NewString("hi")})

	validateType(at, av)
	assertInvalid(tt, bt, av)
	assertInvalid(tt, ct, av)

	validateType(at, bv)
	validateType(bt, bv)
	assertInvalid(tt, ct, bv)

	validateType(at, cv)
	validateType(bt, cv)
	validateType(ct, cv)
}

func TestAssertTypeStructSubtype(tt *testing.T) {
	c1 := NewStruct("Commit", structData{
		"value":   Number(1),
		"parents": NewSet(),
	})
	t1 := MakeStructType("Commit", TypeMap{
		"value":   NumberType,
		"parents": MakeSetType(MakeUnionType()),
	})
	validateType(t1, c1)

	t11 := MakeStructType("Commit", TypeMap{
		"value":   NumberType,
		"parents": MakeSetType(MakeRefType(NumberType /* placeholder */)),
	})
	t11.Desc.(StructDesc).Fields["parents"].Desc.(CompoundDesc).ElemTypes[0].Desc.(CompoundDesc).ElemTypes[0] = t11
	validateType(t11, c1)

	c2 := NewStruct("Commit", structData{
		"value":   Number(2),
		"parents": NewSet(NewRef(c1)),
	})
	validateType(t11, c2)

	// t3 :=
}
