package types

import (
	"testing"

	"github.com/attic-labs/testify/assert"
)

// testing strategy
// - test simplifying each kind in isolation, both shallow and deep
// - test makeSupertype
//   - pass one type only
//   - test that instances are properly deduplicated
//   - test union flattening
//   - test grouping of the various kinds
//   - test cycles

func simplifyRefs(ts typeset, intersectStructs bool) *Type {
	return simplifyContainers(RefKind, ts, intersectStructs)
}
func simplifySets(ts typeset, intersectStructs bool) *Type {
	return simplifyContainers(SetKind, ts, intersectStructs)
}
func simplifyLists(ts typeset, intersectStructs bool) *Type {
	return simplifyContainers(ListKind, ts, intersectStructs)
}

func TestSimplifyType(t *testing.T) {

	for _, intersectStruct := range []bool{false, true} {
		cases := []struct {
			in  []*Type
			out *Type
		}{
			// Ref<Bool> -> Ref<Bool>
			{
				[]*Type{MakeRefType(BoolType)},
				MakeRefType(BoolType),
			},
			// Ref<Number>|Ref<String>|Ref<blob> -> Ref<Number|String|blob>
			{
				[]*Type{MakeRefType(NumberType), MakeRefType(StringType), MakeRefType(BlobType)},
				MakeRefType(MakeUnionType(NumberType, StringType, BlobType)),
			},
			// Ref<Set<Bool>>|Ref<Set<String>> -> Ref<Set<Bool|String>>
			{
				[]*Type{MakeRefType(MakeSetType(BoolType)), MakeRefType(MakeSetType(StringType))},
				MakeRefType(MakeSetType(MakeUnionType(BoolType, StringType))),
			},
			// Ref<Set<Bool>|Ref<Set<String>>|Ref<Number> -> Ref<Set<Bool|String>|Number>
			{
				[]*Type{MakeRefType(MakeSetType(BoolType)), MakeRefType(MakeSetType(StringType)), MakeRefType(NumberType)},
				MakeRefType(MakeUnionType(MakeSetType(MakeUnionType(BoolType, StringType)), NumberType)),
			},

			// Set<Bool> -> Set<Bool>
			{
				[]*Type{MakeSetType(BoolType)},
				MakeSetType(BoolType),
			},
			// Set<Number>|Set<String>|Set<blob> -> Set<Number|String|blob>
			{
				[]*Type{MakeSetType(NumberType), MakeSetType(StringType), MakeSetType(BlobType)},
				MakeSetType(MakeUnionType(NumberType, StringType, BlobType)),
			},
			// Set<Set<Bool>>|Set<Set<String>> -> Set<Set<Bool|String>>
			{
				[]*Type{MakeSetType(MakeSetType(BoolType)), MakeSetType(MakeSetType(StringType))},
				MakeSetType(MakeSetType(MakeUnionType(BoolType, StringType))),
			},
			// Set<Set<Bool>|Set<Set<String>>|Set<Number> -> Set<Set<Bool|String>|Number>
			{
				[]*Type{MakeSetType(MakeSetType(BoolType)), MakeSetType(MakeSetType(StringType)), MakeSetType(NumberType)},
				MakeSetType(MakeUnionType(MakeSetType(MakeUnionType(BoolType, StringType)), NumberType)),
			},

			// List<Bool> -> List<Bool>
			{
				[]*Type{MakeListType(BoolType)},
				MakeListType(BoolType),
			},
			// List<Number>|List<String>|List<blob> -> List<Number|String|blob>
			{
				[]*Type{MakeListType(NumberType), MakeListType(StringType), MakeListType(BlobType)},
				MakeListType(MakeUnionType(NumberType, StringType, BlobType)),
			},
			// List<Set<Bool>>|List<Set<String>> -> List<Set<Bool|String>>
			{
				[]*Type{MakeListType(MakeListType(BoolType)), MakeListType(MakeListType(StringType))},
				MakeListType(MakeListType(MakeUnionType(BoolType, StringType))),
			},
			// List<Set<Bool>|List<Set<String>>|List<Number> -> List<Set<Bool|String>|Number>
			{
				[]*Type{MakeListType(MakeListType(BoolType)), MakeListType(MakeListType(StringType)), MakeListType(NumberType)},
				MakeListType(MakeUnionType(MakeListType(MakeUnionType(BoolType, StringType)), NumberType)),
			},

			// Map<Bool,bool> -> Map<Bool,bool>
			{
				[]*Type{MakeMapType(BoolType, BoolType)},
				MakeMapType(BoolType, BoolType),
			},
			// Map<Bool,bool>|Map<Bool,string> -> Map<Bool,bool|String>
			{
				[]*Type{MakeMapType(BoolType, BoolType), MakeMapType(BoolType, StringType)},
				MakeMapType(BoolType, MakeUnionType(BoolType, StringType)),
			},
			// Map<Bool,bool>|Map<String,bool> -> Map<Bool|String,bool>
			{
				[]*Type{MakeMapType(BoolType, BoolType), MakeMapType(StringType, BoolType)},
				MakeMapType(MakeUnionType(BoolType, StringType), BoolType),
			},
			// Map<Bool,bool>|Map<String,string> -> Map<Bool|String,bool|String>
			{
				[]*Type{MakeMapType(BoolType, BoolType), MakeMapType(StringType, StringType)},
				MakeMapType(MakeUnionType(BoolType, StringType), MakeUnionType(BoolType, StringType)),
			},
			// Map<Set<Bool>,bool>|Map<Set<String>,string> -> Map<Set<Bool|String>,bool|String>
			{
				[]*Type{MakeMapType(MakeSetType(BoolType), BoolType), MakeMapType(MakeSetType(StringType), StringType)},
				MakeMapType(MakeSetType(MakeUnionType(BoolType, StringType)), MakeUnionType(BoolType, StringType)),
			},

			// struct{foo:Bool} -> struct{foo:Bool}
			{
				[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": BoolType})},
				MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
			},
			// struct{foo:Bool}|struct{foo:Number} -> struct{foo:Bool|Number}
			{
				[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
					MakeStructTypeFromFields("", FieldMap{"foo": StringType})},
				MakeStructTypeFromFields("", FieldMap{"foo": MakeUnionType(BoolType, StringType)}),
			},
			// struct{foo:Bool}|struct{foo:Bool,bar:Number} -> struct{foo:Bool,bar?:Number}
			{
				[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
					MakeStructTypeFromFields("", FieldMap{"foo": BoolType, "bar": NumberType})},
				MakeStructType("",
					StructField{"bar", NumberType, !intersectStruct},
					StructField{"foo", BoolType, false},
				),
			},
			// struct{foo:Bool}|struct{bar:Number} -> struct{foo?:Bool,bar?:Number}
			{
				[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": BoolType}),
					MakeStructTypeFromFields("", FieldMap{"bar": NumberType})},
				MakeStructType("",
					StructField{"bar", NumberType, !intersectStruct},
					StructField{"foo", BoolType, !intersectStruct},
				),
			},
			// struct{foo:Ref<Bool>}|struct{foo:Ref<Number>} -> struct{foo:Ref<Bool|Number>}
			{
				[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(BoolType)}),
					MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(NumberType)})},
				MakeStructTypeFromFields("", FieldMap{"foo": MakeRefType(MakeUnionType(BoolType, NumberType))}),
			},

			// struct A{foo:Bool}|struct A{foo:String} -> struct A{foo:Bool|String}
			{
				[]*Type{MakeStructTypeFromFields("A", FieldMap{"foo": BoolType}),
					MakeStructTypeFromFields("A", FieldMap{"foo": StringType})},
				MakeStructTypeFromFields("A", FieldMap{"foo": MakeUnionType(BoolType, StringType)}),
			},

			// struct A { b: struct B { a: Cycle<1> } } ->
			// struct A { b: struct B { a: Cycle<1> } }
			{
				[]*Type{
					MakeStructType("A",
						StructField{"b", MakeStructType("B",
							StructField{"a", MakeCycleType(1), false},
						), false},
					),
				},
				MakeStructType("A",
					StructField{"b", MakeStructType("B",
						StructField{"a", MakeCycleType(1), false},
					), false},
				),
			},

			// struct A { b: struct B { a: Cycle<1> } } | struct A { c: Number } ->
			// struct A { b?: struct B { a: Cycle<1> }, c?: Number }| struct A { c: Number }
			{
				[]*Type{
					MakeStructType("A",
						StructField{"b", MakeStructType("B",
							StructField{"a", MakeCycleType(1), false},
						), false},
					),
					MakeStructType("A",
						StructField{"c", NumberType, false},
					),
				},
				MakeStructType("A",
					StructField{"b", MakeStructType("B",
						StructField{"a", MakeCycleType(1), false},
					), !intersectStruct},
					StructField{"c", NumberType, !intersectStruct},
				),
			},

			// struct {a: struct {b: String}} -> struct {a: struct {b: String}}
			{
				[]*Type{
					MakeStructType("",
						StructField{"a", MakeStructType("",
							StructField{"b", StringType, false},
						), false},
					),
				},
				MakeStructType("",
					StructField{"a", MakeStructType("",
						StructField{"b", StringType, false},
					), false},
				),
			},
		}

		for i, c := range cases {
			act := makeSimplifiedType(intersectStruct, makeCompoundType(UnionKind, c.in...))
			assert.True(t, c.out.Equals(act), "Test case as position %d - got %s, wanted %s", i, act.Describe(), c.out.Describe())
		}
	}
}

func TestMakeSimplifiedUnion(t *testing.T) {
	cycleType := MakeStructTypeFromFields("", FieldMap{"self": MakeCycleType(0)})

	for _, intersectStruct := range []bool{false, true} {

		cases := []struct {
			in  []*Type
			out *Type
		}{
			// {} -> <empty-union>
			{[]*Type{},
				MakeUnionType()},
			// {bool} -> bool
			{[]*Type{BoolType},
				BoolType},
			// {bool,bool} -> bool
			{[]*Type{BoolType, BoolType},
				BoolType},
			// {bool,Number} -> bool|Number
			{[]*Type{BoolType, NumberType},
				MakeUnionType(BoolType, NumberType)},
			// {bool,Number|(string|blob|Number)} -> bool|Number|String|blob
			{[]*Type{BoolType, MakeUnionType(NumberType, MakeUnionType(StringType, BlobType, NumberType))},
				MakeUnionType(BoolType, NumberType, StringType, BlobType)},

			// {Ref<Number>} -> Ref<Number>
			{[]*Type{MakeRefType(NumberType)},
				MakeRefType(NumberType)},
			// {Ref<Number>,Ref<String>} -> Ref<Number|String>
			{[]*Type{MakeRefType(NumberType), MakeRefType(StringType)},
				MakeRefType(MakeUnionType(NumberType, StringType))},

			// {Set<Number>} -> Set<Number>
			{[]*Type{MakeSetType(NumberType)},
				MakeSetType(NumberType)},
			// {Set<Number>,Set<String>} -> Set<Number|String>
			{[]*Type{MakeSetType(NumberType), MakeSetType(StringType)},
				MakeSetType(MakeUnionType(NumberType, StringType))},

			// {List<Number>} -> List<Number>
			{[]*Type{MakeListType(NumberType)},
				MakeListType(NumberType)},
			// {List<Number>,List<String>} -> List<Number|String>
			{[]*Type{MakeListType(NumberType), MakeListType(StringType)},
				MakeListType(MakeUnionType(NumberType, StringType))},

			// {Map<Number,Number>} -> Map<Number,Number>
			{[]*Type{MakeMapType(NumberType, NumberType)},
				MakeMapType(NumberType, NumberType)},
			// {Map<Number,Number>,Map<String,string>} -> Map<Number|String,Number|String>
			{[]*Type{MakeMapType(NumberType, NumberType), MakeMapType(StringType, StringType)},
				MakeMapType(MakeUnionType(NumberType, StringType), MakeUnionType(NumberType, StringType))},

			// {struct{foo:Number}} -> struct{foo:Number}
			{[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": NumberType})},
				MakeStructTypeFromFields("", FieldMap{"foo": NumberType})},
			// {struct{foo:Number}, struct{foo:String}} -> struct{foo:Number|String}
			{[]*Type{MakeStructTypeFromFields("", FieldMap{"foo": NumberType}),
				MakeStructTypeFromFields("", FieldMap{"foo": StringType})},
				MakeStructTypeFromFields("", FieldMap{"foo": MakeUnionType(NumberType, StringType)})},

			// {Bool,String,Ref<Bool>,Ref<String>,Ref<Set<String>>,Ref<Set<Bool>>,
			//    struct{foo:Number},struct{bar:String},struct A{foo:Number}} ->
			// Bool|String|Ref<Bool|String|Set<String|Bool>>|struct{foo?:Number,bar?:String}|struct A{foo:Number}
			{
				[]*Type{
					BoolType, StringType,
					MakeRefType(BoolType), MakeRefType(StringType),
					MakeRefType(MakeSetType(BoolType)), MakeRefType(MakeSetType(StringType)),
					MakeStructTypeFromFields("", FieldMap{"foo": NumberType}),
					MakeStructTypeFromFields("", FieldMap{"bar": StringType}),
					MakeStructTypeFromFields("A", FieldMap{"foo": StringType}),
				},
				MakeUnionType(
					BoolType, StringType,
					MakeRefType(MakeUnionType(BoolType, StringType,
						MakeSetType(MakeUnionType(BoolType, StringType)))),
					MakeStructType("",
						StructField{"foo", NumberType, !intersectStruct},
						StructField{"bar", StringType, !intersectStruct},
					),
					MakeStructTypeFromFields("A", FieldMap{"foo": StringType}),
				),
			},

			{[]*Type{cycleType}, cycleType},

			{[]*Type{cycleType, NumberType, StringType},
				MakeUnionType(cycleType, NumberType, StringType)},
		}

		for i, c := range cases {
			act := makeSimplifiedType(intersectStruct, makeCompoundType(UnionKind, c.in...))
			assert.True(t, c.out.Equals(act), "Test case as position %d - got %s, expected %s", i, act.Describe(), c.out.Describe())
		}

	}
}

func TestSimplifyStructFields(t *testing.T) {
	assert := assert.New(t)

	test := func(in []structTypeFields, exp structTypeFields) {
		act := simplifyStructFields(in, false)
		assert.Equal(act, exp)
	}

	test([]structTypeFields{
		structTypeFields{
			StructField{"a", BoolType, false},
		},
		structTypeFields{
			StructField{"a", BoolType, false},
		},
	},
		structTypeFields{
			StructField{"a", BoolType, false},
		},
	)

	test([]structTypeFields{
		structTypeFields{
			StructField{"a", BoolType, false},
		},
		structTypeFields{
			StructField{"b", BoolType, false},
		},
	},
		structTypeFields{
			StructField{"a", BoolType, true},
			StructField{"b", BoolType, true},
		},
	)

	test([]structTypeFields{
		structTypeFields{
			StructField{"a", BoolType, false},
		},
		structTypeFields{
			StructField{"a", BoolType, true},
		},
	},
		structTypeFields{
			StructField{"a", BoolType, true},
		},
	)
}
