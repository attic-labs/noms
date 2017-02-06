// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import (
	"bytes"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/suite"
)

type QueryGraphQLSuite struct {
	suite.Suite
	vs *types.ValueStore
}

func TestQueryGraphQL(t *testing.T) {
	suite.Run(t, &QueryGraphQLSuite{})
}

func (suite *QueryGraphQLSuite) SetupTest() {
	cs := chunks.NewTestStore()
	suite.vs = types.NewValueStore(types.NewBatchStoreAdaptor(cs))
}

func (suite *QueryGraphQLSuite) assertQueryResult(v types.Value, q, expect string) {
	buff := &bytes.Buffer{}
	Query(v, q, suite.vs, buff)
	suite.Equal(expect, string(buff.Bytes()))
}

func (suite *QueryGraphQLSuite) TestScalars() {
	suite.assertQueryResult(types.String("aaa"), "{value}", `{"data":{"value":"aaa"}}`)
	// suite.assertQueryResult(types.String(""), "{value}", `{"data":{"value":""}}`)

	suite.assertQueryResult(types.Number(0), "{value}", `{"data":{"value":0}}`)
	suite.assertQueryResult(types.Number(1), "{value}", `{"data":{"value":1}}`)
	suite.assertQueryResult(types.Number(-1), "{value}", `{"data":{"value":-1}}`)
	suite.assertQueryResult(types.Number(1<<31), "{value}", `{"data":{"value":2.147483648e+09}}`)
	suite.assertQueryResult(types.Number(-(1 << 31)), "{value}", `{"data":{"value":-2.147483648e+09}}`)
	suite.assertQueryResult(types.Number(0.001), "{value}", `{"data":{"value":0.001}}`)
	suite.assertQueryResult(types.Number(0.00000001), "{value}", `{"data":{"value":1e-08}}`)

	suite.assertQueryResult(types.Bool(false), "{value}", `{"data":{"value":false}}`)
	suite.assertQueryResult(types.Bool(true), "{value}", `{"data":{"value":true}}`)
}

func (suite *QueryGraphQLSuite) TestStructBasic() {
	s1 := types.NewStruct("Foo", types.StructData{
		"a": types.String("aaa"),
		"b": types.Bool(true),
		"c": types.Number(0.1),
	})

	suite.assertQueryResult(s1, "{value{a}}", `{"data":{"value":{"a":"aaa"}}}`)
	suite.assertQueryResult(s1, "{value{a b}}", `{"data":{"value":{"a":"aaa","b":true}}}`)
	suite.assertQueryResult(s1, "{value{a b c}}", `{"data":{"value":{"a":"aaa","b":true,"c":0.1}}}`)
	suite.assertQueryResult(s1, "{value{a c}}", `{"data":{"value":{"a":"aaa","c":0.1}}}`)
}

func (suite *QueryGraphQLSuite) TestEmbeddedStruct() {
	s1 := types.NewStruct("Foo", types.StructData{
		"a": types.String("aaa"),
		"b": types.NewStruct("Bar", types.StructData{
			"c": types.Bool(true),
			"d": types.Number(0.1),
		}),
	})

	suite.assertQueryResult(s1, "{value{a}}", `{"data":{"value":{"a":"aaa"}}}`)
	suite.assertQueryResult(s1, "{value{a b {c}}}", `{"data":{"value":{"a":"aaa","b":{"c":true}}}}`)
	suite.assertQueryResult(s1, "{value{a b {c d}}}", `{"data":{"value":{"a":"aaa","b":{"c":true,"d":0.1}}}}`)
}

func (suite *QueryGraphQLSuite) TestListBasic() {
	list := types.NewList(types.String("foo"), types.String("bar"), types.String("baz"))

	suite.assertQueryResult(list, "{value}", `{"data":{"value":["foo","bar","baz"]}}`)
	suite.assertQueryResult(list, "{value(at:1,count:2)}", `{"data":{"value":["bar","baz"]}}`)

	list = types.NewList(types.Bool(true), types.Bool(false), types.Bool(false))

	suite.assertQueryResult(list, "{value}", `{"data":{"value":[true,false,false]}}`)
	suite.assertQueryResult(list, "{value(at:1,count:2)}", `{"data":{"value":[false,false]}}`)

	list = types.NewList(types.Number(1), types.Number(1.1), types.Number(-100))

	suite.assertQueryResult(list, "{value}", `{"data":{"value":[1,1.1,-100]}}`)
	suite.assertQueryResult(list, "{value(at:1,count:2)}", `{"data":{"value":[1.1,-100]}}`)
}

func (suite *QueryGraphQLSuite) TestListOfStruct() {
	list := types.NewList(
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(28),
			"b": types.String("foo"),
		}),
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(-20.102),
			"b": types.String("bar"),
		}),
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(5),
			"b": types.String("baz"),
		}),
	)

	suite.assertQueryResult(list, "{value{a b}}", `{"data":{"value":[{"a":28,"b":"foo"},{"a":-20.102,"b":"bar"},{"a":5,"b":"baz"}]}}`)

	suite.assertQueryResult(list, "{value{a}}", `{"data":{"value":[{"a":28},{"a":-20.102},{"a":5}]}}`)
}

func (suite *QueryGraphQLSuite) TestSetBasic() {
	set := types.NewSet(types.String("foo"), types.String("bar"), types.String("baz"))

	suite.assertQueryResult(set, "{value}", `{"data":{"value":["bar","baz","foo"]}}`)
	suite.assertQueryResult(set, "{value(count:2)}", `{"data":{"value":["bar","baz"]}}`)

	set = types.NewSet(types.Bool(true), types.Bool(false))

	suite.assertQueryResult(set, "{value}", `{"data":{"value":[false,true]}}`)
	suite.assertQueryResult(set, "{value(count:1)}", `{"data":{"value":[false]}}`)

	set = types.NewSet(types.Number(1), types.Number(1.1), types.Number(-100))

	suite.assertQueryResult(set, "{value}", `{"data":{"value":[-100,1,1.1]}}`)
	suite.assertQueryResult(set, "{value(count:2)}", `{"data":{"value":[-100,1]}}`)
}

func (suite *QueryGraphQLSuite) TestSetOfStruct() {
	set := types.NewSet(
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(28),
			"b": types.String("foo"),
		}),
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(-20.102),
			"b": types.String("bar"),
		}),
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(5),
			"b": types.String("baz"),
		}),
	)

	suite.assertQueryResult(set, "{value{a b}}", `{"data":{"value":[{"a":-20.102,"b":"bar"},{"a":5,"b":"baz"},{"a":28,"b":"foo"}]}}`)

	suite.assertQueryResult(set, "{value{a}}", `{"data":{"value":[{"a":-20.102},{"a":5},{"a":28}]}}`)
}

func (suite *QueryGraphQLSuite) TestMapBasic() {
	m := types.NewMap(
		types.String("foo"), types.Number(1),
		types.String("bar"), types.Number(2),
		types.String("baz"), types.Number(3),
	)

	suite.assertQueryResult(m, "{value{key value}}", `{"data":{"value":[{"key":"bar","value":2},{"key":"baz","value":3},{"key":"foo","value":1}]}}`)
	suite.assertQueryResult(m, "{value(count:2){value}}", `{"data":{"value":[{"value":2},{"value":3}]}}`)
	suite.assertQueryResult(m, "{value(count:3){key}}", `{"data":{"value":[{"key":"bar"},{"key":"baz"},{"key":"foo"}]}}`)
}

func (suite *QueryGraphQLSuite) TestMapOfStruct() {
	m := types.NewMap(
		types.String("foo"), types.NewStruct("Foo", types.StructData{
			"a": types.Number(28),
			"b": types.String("foo"),
		}),
		types.String("bar"), types.NewStruct("Foo", types.StructData{
			"a": types.Number(-20.102),
			"b": types.String("bar"),
		}),
		types.String("baz"), types.NewStruct("Foo", types.StructData{
			"a": types.Number(5),
			"b": types.String("baz"),
		}),
	)

	suite.assertQueryResult(m, "{value{key value{a}}}", `{"data":{"value":[{"key":"bar","value":{"a":-20.102}},{"key":"baz","value":{"a":5}},{"key":"foo","value":{"a":28}}]}}`)
	suite.assertQueryResult(m, "{value(count:1){value{a b}}}", `{"data":{"value":[{"value":{"a":-20.102,"b":"bar"}}]}}`)
	suite.assertQueryResult(m, "{value(count:3){key}}", `{"data":{"value":[{"key":"bar"},{"key":"baz"},{"key":"foo"}]}}`)
}

func (suite *QueryGraphQLSuite) TestRef() {
	r := suite.vs.WriteValue(types.Number(100))

	suite.assertQueryResult(r, "{value{targetValue}}", `{"data":{"value":{"targetValue":100}}}`)
	suite.assertQueryResult(r, "{value{targetHash}}", `{"data":{"value":{"targetHash":"fpbhln9asjlalp10btna9ocuc4nj9v15"}}}`)
	suite.assertQueryResult(r, "{value{targetValue targetHash}}", `{"data":{"value":{"targetHash":"fpbhln9asjlalp10btna9ocuc4nj9v15","targetValue":100}}}`)

	r = suite.vs.WriteValue(types.NewStruct("Foo", types.StructData{
		"a": types.Number(28),
		"b": types.String("foo"),
	}))

	suite.assertQueryResult(r, "{value{targetValue{a}}}", `{"data":{"value":{"targetValue":{"a":28}}}}`)
	suite.assertQueryResult(r, "{value{targetValue{a b}}}", `{"data":{"value":{"targetValue":{"a":28,"b":"foo"}}}}`)

	r = suite.vs.WriteValue(types.NewList(types.String("foo"), types.String("bar"), types.String("baz")))

	suite.assertQueryResult(r, "{value{targetValue}}", `{"data":{"value":{"targetValue":["foo","bar","baz"]}}}`)
	suite.assertQueryResult(r, "{value{targetValue(at:1,count:2)}}", `{"data":{"value":{"targetValue":["bar","baz"]}}}`)
}

func (suite *QueryGraphQLSuite) TestListOfUnionOfStructs() {
	list := types.NewList(
		types.NewStruct("Foo", types.StructData{
			"a": types.Number(28),
			"b": types.String("baz"),
		}),
		types.NewStruct("Bar", types.StructData{
			"b": types.String("bar"),
		}),
		types.NewStruct("Baz", types.StructData{
			"c": types.Bool(true),
		}),
	)

	suite.assertQueryResult(list, "{value{... on Foo{a b} ... on Bar{b} ... on Baz{c}}}", `{"data":{"value":[{"a":28,"b":"baz"},{"b":"bar"},{"c":true}]}}`)
}

func (suite *QueryGraphQLSuite) TestCyclicStructs() {
	typ := types.MakeStructTypeFromFields("A", types.FieldMap{
		"a": types.StringType,
		"b": types.MakeSetType(types.MakeCycleType(0)),
	})

	// Struct A {
	//  a: "aaa"
	//  b: Set(Struct A {
	// 	 a: "bbb"
	// 	 b: Set()
	//  })
	// }

	s1 := types.NewStructWithType(typ, types.ValueSlice{
		types.String("aaa"),
		types.NewSet(types.NewStructWithType(typ, types.ValueSlice{types.String("bbb"), types.NewSet()})),
	})

	suite.assertQueryResult(s1, "{value{a b{a}}}", `{"data":{"value":{"a":"aaa","b":[{"a":"bbb"}]}}}`)
}
