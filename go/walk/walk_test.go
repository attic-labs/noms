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

func (suite *WalkAllTestSuite) walkWorker(r types.Ref, expected int) {
	actual := 0
	AllP(r, suite.vs, func(c types.Value, r *types.Ref) {
		actual++
	})
	suite.Equal(expected, actual)
}

func (suite *WalkAllTestSuite) walkValuesWorker(v types.Value, expected int, deep bool) {
	actual := 0
	WalkValues(v, suite.vs, func(c types.Value) {
		actual++
	}, deep)
	suite.Equal(expected, actual)
}

func (suite *WalkAllTestSuite) walkRefsWorker(v types.Value, expected int, deep bool) {
	actual := 0
	WalkRefs(v, suite.vs, func(r types.Ref) {
		actual++
	}, deep)
	suite.Equal(expected, actual)
}

func (suite *WalkAllTestSuite) TestWalkValues() {

	dup := suite.NewList(types.Number(9), types.Number(10), types.Number(11), types.Number(12), types.Number(13))
	l := suite.NewList(types.Number(8), dup, dup)

	suite.walkValuesWorker(l, 10, true)
	suite.walkValuesWorker(types.NewList(types.Number(8), types.Number(9), types.Number(10)), 3, false)
}

func (suite *WalkAllTestSuite) TestWalkRefs() {
	dup := suite.NewList(types.Number(9), types.Number(10), types.Number(11), types.Number(12), types.Number(13))
	l := suite.NewList(types.Number(8), dup, dup)

	suite.walkRefsWorker(l, 2, true)
	suite.walkRefsWorker(suite.NewList(types.Number(8), types.Number(9), types.Number(10)), 1, false)
}

func (suite *WalkAllTestSuite) TestWalkPrimitives() {
	suite.walkWorker(suite.vs.WriteValue(types.Number(0.0)), 2)
	suite.walkWorker(suite.vs.WriteValue(types.String("hello")), 2)
}

func (suite *WalkAllTestSuite) TestWalkComposites() {
	suite.walkWorker(suite.NewList(), 2)
	suite.walkWorker(suite.NewList(types.Bool(false), types.Number(8)), 4)
	suite.walkWorker(suite.NewSet(), 2)
	suite.walkWorker(suite.NewSet(types.Bool(false), types.Number(8)), 4)
	suite.walkWorker(suite.NewMap(), 2)
	suite.walkWorker(suite.NewMap(types.Number(8), types.Bool(true), types.Number(0), types.Bool(false)), 6)
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
	suite.walkWorker(suite.NewList(suite.NewSet(), types.Number(8)), 5)
	suite.walkWorker(suite.NewSet(suite.NewList(), suite.NewSet()), 6)
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
	suite.walkWorker(nested, 25)
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

func (suite *WalkTestSuite) TestStopWalkImmediately() {
	actual := 0
	SomeP(types.NewList(types.NewSet(), types.NewList()), suite.vs, func(v types.Value, r *types.Ref) bool {
		actual++
		return true
	})
	suite.Equal(1, actual)
}

func (suite *WalkTestSuite) skipWorker(composite types.Value) (reached []types.Value) {
	SomeP(composite, suite.vs, func(v types.Value, r *types.Ref) bool {
		suite.False(v.Equals(suite.deadValue), "Should never have reached %+v", suite.deadValue)
		reached = append(reached, v)
		return v.Equals(suite.mustSkip)
	})
	return
}

// Skipping a sub-tree must allow other items in the list to be processed.
func (suite *WalkTestSuite) SkipTestSkipListElement() {
	wholeList := types.NewList(suite.mustSkip, suite.shouldSee, suite.shouldSee)
	reached := suite.skipWorker(wholeList)
	for _, v := range []types.Value{wholeList, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 6)
}

func (suite *WalkTestSuite) SkipTestSkipSetElement() {
	wholeSet := types.NewSet(suite.mustSkip, suite.shouldSee).Insert(suite.shouldSee)
	reached := suite.skipWorker(wholeSet)
	for _, v := range []types.Value{wholeSet, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 4)
}

func (suite *WalkTestSuite) SkipTestSkipMapValue() {
	shouldAlsoSeeItem := types.String("Also good")
	shouldAlsoSee := types.NewSet(shouldAlsoSeeItem)
	wholeMap := types.NewMap(suite.shouldSee, suite.mustSkip, shouldAlsoSee, suite.shouldSee)
	reached := suite.skipWorker(wholeMap)
	for _, v := range []types.Value{wholeMap, suite.shouldSee, suite.shouldSeeItem, suite.mustSkip, shouldAlsoSee, shouldAlsoSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 8)
}

func (suite *WalkTestSuite) SkipTestSkipMapKey() {
	wholeMap := types.NewMap(suite.mustSkip, suite.shouldSee, suite.shouldSee, suite.shouldSee)
	reached := suite.skipWorker(wholeMap)
	for _, v := range []types.Value{wholeMap, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 8)
}
