package test_util

import (
	"testing"

	"github.com/attic-labs/noms/clients/go/util"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/suite"
)

func TestLibTestSuite(t *testing.T) {
	suite.Run(t, &LibTestSuite{})
}

type LibTestSuite struct {
	suite.Suite
}

func (suite *LibTestSuite) TestPrimitiveTypes() {
	suite.EqualValues(types.NewString("expected"), util.NomsValueFromDecodedJSON("expected"))
	suite.EqualValues(types.Bool(false), util.NomsValueFromDecodedJSON(false))
	suite.EqualValues(types.Number(1.7), util.NomsValueFromDecodedJSON(1.7))
	suite.False(util.NomsValueFromDecodedJSON(1.7).Equals(types.Bool(true)))
}

func (suite *LibTestSuite) TestCompositeTypes() {
	// [false true]
	suite.EqualValues(
		types.NewList().Append(types.Bool(false)).Append(types.Bool(true)),
		util.NomsValueFromDecodedJSON([]interface{}{false, true}))

	// [[false true]]
	suite.EqualValues(
		types.NewList().Append(
			types.NewList().Append(types.Bool(false)).Append(types.Bool(true))),
		util.NomsValueFromDecodedJSON([]interface{}{[]interface{}{false, true}}))

	// {"string": "string",
	//  "list": [false true],
	//  "map": {"nested": "string"}
	// }
	m := types.NewMap(
		types.NewString("string"),
		types.NewString("string"),
		types.NewString("list"),
		types.NewList().Append(types.Bool(false)).Append(types.Bool(true)),
		types.NewString("map"),
		types.NewMap(
			types.NewString("nested"),
			types.NewString("string")))
	o := util.NomsValueFromDecodedJSON(map[string]interface{}{
		"string": "string",
		"list":   []interface{}{false, true},
		"map":    map[string]interface{}{"nested": "string"},
	})

	suite.True(m.Equals(o))
}

func (suite *LibTestSuite) TestPanicOnUnsupportedType() {
	suite.Panics(func() { util.NomsValueFromDecodedJSON(map[int]string{1: "one"}) }, "Should panic on map[int]string!")
}
