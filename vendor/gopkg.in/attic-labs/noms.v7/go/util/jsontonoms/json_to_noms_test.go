// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package jsontonoms

import (
	"testing"

	"gopkg.in/attic-labs/noms.v7/go/types"
	"github.com/attic-labs/testify/suite"
)

func TestLibTestSuite(t *testing.T) {
	suite.Run(t, &LibTestSuite{})
}

type LibTestSuite struct {
	suite.Suite
}

func (suite *LibTestSuite) TestPrimitiveTypes() {
	suite.EqualValues(types.String("expected"), NomsValueFromDecodedJSON("expected", false))
	suite.EqualValues(types.Bool(false), NomsValueFromDecodedJSON(false, false))
	suite.EqualValues(types.Number(1.7), NomsValueFromDecodedJSON(1.7, false))
	suite.False(NomsValueFromDecodedJSON(1.7, false).Equals(types.Bool(true)))
}

func (suite *LibTestSuite) TestCompositeTypes() {
	// [false true]
	suite.EqualValues(
		types.NewList().Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(nil),
		NomsValueFromDecodedJSON([]interface{}{false, true}, false))

	// [[false true]]
	suite.EqualValues(
		types.NewList().Edit().Append(
			types.NewList().Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(nil)).List(nil),
		NomsValueFromDecodedJSON([]interface{}{[]interface{}{false, true}}, false))

	// {"string": "string",
	//  "list": [false true],
	//  "map": {"nested": "string"}
	// }
	m := types.NewMap(
		types.String("string"),
		types.String("string"),
		types.String("list"),
		types.NewList().Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(nil),
		types.String("map"),
		types.NewMap(
			types.String("nested"),
			types.String("string")))
	o := NomsValueFromDecodedJSON(map[string]interface{}{
		"string": "string",
		"list":   []interface{}{false, true},
		"map":    map[string]interface{}{"nested": "string"},
	}, false)

	suite.True(m.Equals(o))
}

func (suite *LibTestSuite) TestCompositeTypeWithStruct() {
	// {"string": "string",
	//  "list": [false true],
	//  "struct": {"nested": "string"}
	// }
	tstruct := types.NewStruct("", types.StructData{
		"string": types.String("string"),
		"list":   types.NewList().Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(nil),
		"struct": types.NewStruct("", types.StructData{
			"nested": types.String("string"),
		}),
	})
	o := NomsValueFromDecodedJSON(map[string]interface{}{
		"string": "string",
		"list":   []interface{}{false, true},
		"struct": map[string]interface{}{"nested": "string"},
	}, true)

	suite.True(tstruct.Equals(o))
}

func (suite *LibTestSuite) TestCompositeTypeWithNamedStruct() {
	// {
	//  "_name": "TStruct1",
	//  "string": "string",
	//  "list": [false true],
	//  "id": {"_name", "Id", "owner": "string", "value": "string"}
	// }
	tstruct := types.NewStruct("TStruct1", types.StructData{
		"string": types.String("string"),
		"list":   types.NewList().Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(nil),
		"struct": types.NewStruct("Id", types.StructData{
			"owner": types.String("string"),
			"value": types.String("string"),
		}),
	})
	o := NomsValueUsingNamedStructsFromDecodedJSON(map[string]interface{}{
		"_name":  "TStruct1",
		"string": "string",
		"list":   []interface{}{false, true},
		"struct": map[string]interface{}{"_name": "Id", "owner": "string", "value": "string"},
	})

	suite.True(tstruct.Equals(o))
}

func (suite *LibTestSuite) TestPanicOnUnsupportedType() {
	suite.Panics(func() { NomsValueFromDecodedJSON(map[int]string{1: "one"}, false) }, "Should panic on map[int]string!")
}
