package types

import (
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

// Type structs define and describe Noms types, both custom and built-in.
// Name is optional.
// Kind and Desc collectively describe any legal Noms type. Kind captures what kind of type the instance describes, e.g. Set, Bool, Map, Struct, etc. Desc captures further information needed to flesh out the definition, such as the type of the elements in a List, or the field names and types of a Struct.
// If Kind refers to a primitive, then Desc is empty.
// If Kind refers to Set or List, then Desc is a map[String]Type{"elem": elementType}
// If Kind refers to Map, then Desc is a map[String]Type{"key": keyType, "value": valueType}
// If Kind refers to Struct, then Desc is a map[String]Type{"fieldName1": field1Type, "fieldName2": field2Type}
// TODO: Replace Kind and Desc with a Union that can hold this kind of type description information in a less ad-hoc kind of way. Also, TODO...make Unions a thing.
type Type struct {
	Name String
	Kind UInt8
	Desc Map
	ref  ref.Ref
}

func MakePrimitiveType(n string, k Kind) Type {
	return makeType(NewString(n), k, NewMap())
}

func MakeMapType(name String, keyType, valueType Type) Type {
	return makeType(name, MapKind, NewMap(NewString("key"), keyType, NewString("value"), valueType))
}

func MakeListType(name String, valueType Type) Type {
	return makeType(name, ListKind, NewMap(NewString("elem"), valueType))
}

func MakeSetType(name String, valueType Type) Type {
	return makeType(name, SetKind, NewMap(NewString("elem"), valueType))
}

func MakeStructType(name String, fields Map) Type {
	return makeType(name, StructKind, fields)
}

func makeType(name String, kind Kind, desc Map) Type {
	switch kind {
	case BoolKind, Int8Kind, Int16Kind, Int32Kind, Int64Kind, Float32Kind, Float64Kind, UInt8Kind, UInt16Kind, UInt32Kind, UInt64Kind, StringKind, ListKind, MapKind, SetKind, StructKind:
		return Type{Name: name, Kind: UInt8(kind), Desc: desc}
	default:
		d.Exp.Fail("Unrecognized Kind:", "%v", kind)
		panic("unreachable")
	}
}

func (t Type) Ref() ref.Ref {
	return ensureRef(&t.ref, t)
}

func (t Type) Equals(other Value) (res bool) {
	if other == nil {
		return false
	} else {
		return t.Ref() == other.Ref()
	}
}

func (t Type) Chunks() []Future {
	return t.Desc.Chunks()
}

type Kind uint8

const (
	BoolKind Kind = iota
	Int8Kind
	Int16Kind
	Int32Kind
	Int64Kind
	Float32Kind
	Float64Kind
	UInt8Kind
	UInt16Kind
	UInt32Kind
	UInt64Kind
	StringKind
	ListKind
	MapKind
	SetKind
	StructKind
)
