// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package walk

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/suite"
)

func TestWalkTestSuite(t *testing.T) {
	suite.Run(t, &WalkTestSuite{})
}

func TestWalkAllTestSuite(t *testing.T) {
	suite.Run(t, &WalkAllTestSuite{})
}

type WalkAllTestSuite struct {
	suite.Suite
	vs *types.ValueStore
}

func (suite *WalkAllTestSuite) SetupTest() {
	suite.vs = types.NewTestValueStore()
}

func (suite *WalkAllTestSuite) walkWorker(v types.Value, expected int, deep bool) {
	actual := 0
	WalkValues(v, suite.vs, func(c types.Value) {
		actual++
	}, deep)
	suite.Equal(expected, actual)
}

func (suite *WalkAllTestSuite) walkWorkerDeep(v types.Value, expected int) {
	suite.walkWorker(v, expected, true)
}

func (suite *WalkAllTestSuite) walkWorkerShallow(v types.Value, expected int) {
	suite.walkWorker(v, expected, false)
}

func (suite *WalkAllTestSuite) TestWalkValuesDuplicates() {

	dup := suite.NewList(types.Number(9), types.Number(10), types.Number(11), types.Number(12), types.Number(13))
	l := suite.NewList(types.Number(8), dup, dup)

	suite.walkWorkerDeep(l, 11)
	suite.walkWorkerShallow(types.NewList(types.Number(8), types.Number(9), types.Number(10)), 4)
}

func (suite *WalkAllTestSuite) TestWalkPrimitives() {
	suite.walkWorkerDeep(suite.vs.WriteValue(types.Number(0.0)), 2)
	suite.walkWorkerDeep(suite.vs.WriteValue(types.String("hello")), 2)
}

func (suite *WalkAllTestSuite) TestWalkComposites() {
	suite.walkWorkerDeep(suite.NewList(), 2)
	suite.walkWorkerDeep(suite.NewList(types.Bool(false), types.Number(8)), 4)
	suite.walkWorkerDeep(suite.NewSet(), 2)
	suite.walkWorkerDeep(suite.NewSet(types.Bool(false), types.Number(8)), 4)
	suite.walkWorkerDeep(suite.NewMap(), 2)
	suite.walkWorkerDeep(suite.NewMap(types.Number(8), types.Bool(true), types.Number(0), types.Bool(false)), 6)
}

func (suite *WalkAllTestSuite) NewList(vs ...types.Value) types.Ref {
	v := types.NewList(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) NewMap(vs ...types.Value) types.Ref {
	v := types.NewMap(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) NewSet(vs ...types.Value) types.Ref {
	v := types.NewSet(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) TestWalkNestedComposites() {
	suite.walkWorkerDeep(suite.NewList(suite.NewSet(), types.Number(8)), 5)
	suite.walkWorkerDeep(suite.NewSet(suite.NewList(), suite.NewSet()), 6)
	// {"string": "string",
	//  "list": [false true],
	//  "map": {"nested": "string"}
	//  "mtlist": []
	//  "set": [5 7 8]
	//  []: "wow"
	// }
	nested := suite.NewMap(
		types.String("string"), types.String("string"),
		types.String("list"), suite.NewList(types.Bool(false), types.Bool(true)),
		types.String("map"), suite.NewMap(types.String("nested"), types.String("string")),
		types.String("mtlist"), suite.NewList(),
		types.String("set"), suite.NewSet(types.Number(5), types.Number(7), types.Number(8)),
		suite.NewList(), types.String("wow"), // note that the dupe list chunk is skipped
	)
	suite.walkWorkerDeep(nested, 25)
}

type WalkTestSuite struct {
	WalkAllTestSuite
	shouldSeeItem types.Value
	shouldSee     types.Value
	mustSkip      types.Value
	deadValue     types.Value
}

func (suite *WalkTestSuite) SetupTest() {
	suite.vs = types.NewTestValueStore()
	suite.shouldSeeItem = types.String("zzz")
	suite.shouldSee = types.NewList(suite.shouldSeeItem)
	suite.deadValue = types.Number(0xDEADBEEF)
	suite.mustSkip = types.NewList(suite.deadValue)
}
