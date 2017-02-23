package types

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestIntersectImpl(t *testing.T) {
	cases := []struct {
		a, b *Type
		out  bool
	}{
		// bool & any -> true
		{ValueType, StringType, true},
		// ref<bool> & ref<bool> -> true
		{MakeRefType(BoolType), MakeRefType(BoolType), true},
		// ref<number> & ref<string> -> false
		{MakeRefType(NumberType), MakeRefType(StringType), false},
		// set<bool> & set<bool> -> true
		{MakeSetType(BoolType), MakeSetType(BoolType), true},
		// set<bool> & set<string> -> false
		{MakeSetType(BoolType), MakeSetType(StringType), false},
		// list<blob> & list<blob> -> true
		{MakeListType(BlobType), MakeListType(BlobType), true},
		// list<blob> & list<string> -> false
		{MakeListType(BlobType), MakeListType(StringType), false},

		// map<bool,bool> & map<bool,bool> -> true
		{MakeMapType(BoolType, BoolType), MakeMapType(BoolType, BoolType), true},
		// map<bool,bool> & map<bool,string> -> false
		{MakeMapType(BoolType, BoolType), MakeMapType(BoolType, StringType), false},
		// map<bool,bool> & map<string,bool> -> false
		{MakeMapType(BoolType, BoolType), MakeMapType(StringType, BoolType), false},

		// bool & string|bool|blob -> true
		{BoolType, MakeUnionType(StringType, BoolType, BlobType), true},
		// string|bool|blob & blob -> true
		{MakeUnionType(StringType, BoolType, BlobType), BlobType, true},
		// string|bool|blob & number|blob|string -> true
		{MakeUnionType(StringType, BoolType, BlobType), MakeUnionType(NumberType, BlobType, StringType), true},

		// struct{foo:bool} & struct{foo:bool} -> true
		{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
			MakeStructTypeFromFields("", FieldMap{"foo": BoolType}), true},
		// struct{foo:bool} & struct{foo:number} -> false
		{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
			MakeStructTypeFromFields("", FieldMap{"foo": StringType}), false},
		// struct{foo:bool} & struct{foo:bool,bar:number} -> true
		{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
			MakeStructTypeFromFields("", FieldMap{"foo": BoolType, "bar": NumberType}), true},
		// struct{foo:ref<bool>} & struct{foo:ref<number>} -> false
		{MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(BoolType)}),
			MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(NumberType)}), false},
		// struct{foo:ref<bool>} & struct{foo:ref<number|bool>} -> true
		{MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(BoolType)}),
			MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(MakeUnionType(NumberType, BoolType))}), true},
		// struct A{foo:bool} & struct A{foo:bool, baz:string} -> true
		{MakeStructTypeFromFields("A", FieldMap{"foo": BoolType}),
			MakeStructTypeFromFields("A", FieldMap{"foo": BoolType, "baz": StringType}), true},
		// struct A{foo:bool} & struct B{foo:bool} -> false
		{MakeStructTypeFromFields("A", FieldMap{"foo": BoolType}),
			MakeStructTypeFromFields("B", FieldMap{"foo": BoolType}), false},
		// map<string, struct A{foo:string}> & map<string, struct A{foo:string, bar:bool}> -> true
		{MakeMapType(StringType, MakeStructTypeFromFields("A", FieldMap{"foo": StringType})),
			MakeMapType(StringType, MakeStructTypeFromFields("A", FieldMap{"foo": StringType, "bar": BoolType})), true},

		// map<struct{x:number, y:number}, struct A{foo:string}> & map<struct{x:number, y:number}, struct A{foo:string, bar:bool}> -> true
		{
			MakeMapType(
				MakeStructTypeFromFields("", FieldMap{"x": NumberType, "y": NumberType}),
				MakeStructTypeFromFields("A", FieldMap{"foo": StringType})),
			MakeMapType(
				MakeStructTypeFromFields("", FieldMap{"x": NumberType, "y": NumberType}),
				MakeStructTypeFromFields("A", FieldMap{"foo": StringType, "bar": BoolType})),
			true,
		},

		// struct A{self:A} & struct A{self:A, foo:Number} -> true
		{MakeStructTypeFromFields("A", FieldMap{"self": MakeCycleType(0)}),
			MakeStructTypeFromFields("A", FieldMap{"self": MakeCycleType(0), "foo": NumberType}), true},
	}

	for i, c := range cases {
		act := TypesIntersect(c.a, c.b)
		assert.Equal(t, c.out, act, "Test case at position %d; \n\ta:%s\n\tb:%s", i, c.a.Describe(), c.b.Describe())
	}
}
