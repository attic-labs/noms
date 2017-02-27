// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nomdl

import (
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/suite"
)

type ParserSuite struct {
	suite.Suite
}

func TestParser(t *testing.T) {
	suite.Run(t, &ParserSuite{})
}

func (suite *ParserSuite) initParser(src string) *Parser {
	return New(strings.NewReader(src), ParserOptions{
		Filename: "example",
	})
}

func (suite *ParserSuite) assertParseType(code string, expected *types.Type) {
	actual, err := ParseType(code)
	suite.NoError(err)
	suite.True(expected.Equals(actual), "Expected: %s, Actual: %s", expected.Describe(), actual.Describe())
}

func (suite *ParserSuite) assertParseError(code, msg string) {
	p := New(strings.NewReader(code), ParserOptions{
		Filename: "example",
	})
	err := catchSyntaxError(func() {
		t := p.parseType()
		suite.Nil(t)
	})
	suite.Error(err)
	suite.Equal(msg, err.Error())
}

func (suite *ParserSuite) TestSimpleTypes() {
	suite.assertParseType("Blob", types.BlobType)
	suite.assertParseType("Bool", types.BoolType)
	suite.assertParseType("Number", types.NumberType)
	suite.assertParseType("String", types.StringType)
	suite.assertParseType("Value", types.ValueType)
	suite.assertParseType("Type", types.TypeType)
}

func (suite *ParserSuite) TestWhitespace() {
	for _, r := range " \t\n\r" {
		suite.assertParseType(string(r)+"Blob", types.BlobType)
		suite.assertParseType("Blob"+string(r), types.BlobType)
	}
}

func (suite *ParserSuite) TestComments() {
	suite.assertParseType("/* */Blob", types.BlobType)
	suite.assertParseType("Blob/* */", types.BlobType)
	suite.assertParseType("Blob//", types.BlobType)
	suite.assertParseType("//\nBlob", types.BlobType)
}

func (suite *ParserSuite) TestCompoundTypes() {
	suite.assertParseType("List<>", types.MakeListType(types.MakeUnionType()))
	suite.assertParseType("List<Bool>", types.MakeListType(types.BoolType))
	suite.assertParseError("List<Bool, Number>", `Unexpected token ",", expected ">", example:1:11`)

	suite.assertParseType("Set<>", types.MakeSetType(types.MakeUnionType()))
	suite.assertParseType("Set<Bool>", types.MakeSetType(types.BoolType))
	suite.assertParseError("Set<Bool, Number>", `Unexpected token ",", expected ">", example:1:10`)

	suite.assertParseError("Ref<>", `Unexpected token ">", example:1:6`)
	suite.assertParseType("Ref<Bool>", types.MakeRefType(types.BoolType))
	suite.assertParseError("Ref<Number, Bool>", `Unexpected token ",", expected ">", example:1:12`)

	suite.assertParseType("Cycle<42>", types.MakeCycleType(42))
	suite.assertParseError("Cycle<-123>", `Unexpected token "-", expected Int, example:1:8`)
	suite.assertParseError("Cycle<12.3>", `Unexpected token Float, expected Int, example:1:11`)

	suite.assertParseType("Map<>", types.MakeMapType(types.MakeUnionType(), types.MakeUnionType()))
	suite.assertParseType("Map<Bool, String>", types.MakeMapType(types.BoolType, types.StringType))
	suite.assertParseError("Map<Bool,>", `Unexpected token ">", example:1:11`)
	suite.assertParseError("Map<,Bool>", `Unexpected token ",", example:1:6`)
	suite.assertParseError("Map<,>", `Unexpected token ",", example:1:6`)
}

func (suite *ParserSuite) TestStructTypes() {
	suite.assertParseType("struct {}", types.MakeStructTypeFromFields("", types.FieldMap{}))
	suite.assertParseType("struct S {}", types.MakeStructTypeFromFields("S", types.FieldMap{}))

	suite.assertParseType(`struct S {
                x: Number
                }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.NumberType,
	}))

	suite.assertParseType(`struct S {
                x: Number,
        }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.NumberType,
	}))

	suite.assertParseType(`struct S {
                x: Number,
                y: String
        }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.NumberType,
		"y": types.StringType,
	}))

	suite.assertParseType(`struct S {
                x: Number,
                y: String,
        }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.NumberType,
		"y": types.StringType,
	}))

	suite.assertParseError(`struct S {
                x: Number
                y: String
        }`, `Unexpected token Ident, expected ",", example:3:18`)
}

func (suite *ParserSuite) TestUnionTypes() {
	suite.assertParseType("Blob | Bool", types.MakeUnionType(types.BlobType, types.BoolType))
	suite.assertParseType("Bool | Number | String", types.MakeUnionType(types.BoolType, types.NumberType, types.StringType))
	suite.assertParseType("List<Bool | Number>", types.MakeListType(types.MakeUnionType(types.BoolType, types.NumberType)))
	suite.assertParseType("Map<Bool | Number, Bool | Number>",
		types.MakeMapType(
			types.MakeUnionType(types.BoolType, types.NumberType),
			types.MakeUnionType(types.BoolType, types.NumberType),
		),
	)
	suite.assertParseType(`struct S {
                x: Number | Bool
                }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.MakeUnionType(types.BoolType, types.NumberType),
	}))
	suite.assertParseType(`struct S {
                x: Number | Bool,
                y: String
        }`, types.MakeStructTypeFromFields("S", types.FieldMap{
		"x": types.MakeUnionType(types.BoolType, types.NumberType),
		"y": types.StringType,
	}))

	suite.assertParseError("Bool |", "Unexpected token EOF, example:1:7")
	suite.assertParseError("Bool | Number |", "Unexpected token EOF, example:1:16")
}
