// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

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
	vs *ValueStore
}

func (suite *WalkAllTestSuite) SetupTest() {
	suite.vs = NewTestValueStore()
}

func (suite *WalkAllTestSuite) walkWorker(r Ref, expected int) {
	actual := 0
	AllP(r, suite.vs, func(c Value, r *Ref) {
		actual++
	}, 1)
	suite.Equal(expected, actual)
}

func (suite *WalkAllTestSuite) TestWalkPrimitives() {
	suite.walkWorker(suite.vs.WriteValue(Number(0.0)), 2)
	suite.walkWorker(suite.vs.WriteValue(String("hello")), 2)
}

func (suite *WalkAllTestSuite) TestWalkComposites() {
	suite.walkWorker(suite.NewList(), 2)
	suite.walkWorker(suite.NewList(Bool(false), Number(8)), 4)
	suite.walkWorker(suite.NewSet(), 2)
	suite.walkWorker(suite.NewSet(Bool(false), Number(8)), 4)
	suite.walkWorker(suite.NewMap(), 2)
	suite.walkWorker(suite.NewMap(Number(8), Bool(true), Number(0), Bool(false)), 6)
}

func (suite *WalkAllTestSuite) NewList(vs ...Value) Ref {
	v := NewList(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) NewMap(vs ...Value) Ref {
	v := NewMap(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) NewSet(vs ...Value) Ref {
	v := NewSet(vs...)
	return suite.vs.WriteValue(v)
}

func (suite *WalkAllTestSuite) TestWalkNestedComposites() {
	suite.walkWorker(suite.NewList(suite.NewSet(), Number(8)), 5)
	suite.walkWorker(suite.NewSet(suite.NewList(), suite.NewSet()), 6)
	// {"string": "string",
	//  "list": [false true],
	//  "map": {"nested": "string"}
	//  "mtlist": []
	//  "set": [5 7 8]
	//  []: "wow"
	// }
	nested := suite.NewMap(
		String("string"), String("string"),
		String("list"), suite.NewList(Bool(false), Bool(true)),
		String("map"), suite.NewMap(String("nested"), String("string")),
		String("mtlist"), suite.NewList(),
		String("set"), suite.NewSet(Number(5), Number(7), Number(8)),
		suite.NewList(), String("wow"), // note that the dupe list chunk is skipped
	)
	suite.walkWorker(nested, 25)
}

type WalkTestSuite struct {
	WalkAllTestSuite
	shouldSeeItem Value
	shouldSee     Value
	mustSkip      Value
	deadValue     Value
}

func (suite *WalkTestSuite) SetupTest() {
	suite.vs = NewTestValueStore()
	suite.shouldSeeItem = String("zzz")
	suite.shouldSee = NewList(suite.shouldSeeItem)
	suite.deadValue = Number(0xDEADBEEF)
	suite.mustSkip = NewList(suite.deadValue)
}

func (suite *WalkTestSuite) TestStopWalkImmediately() {
	actual := 0
	SomeP(NewList(NewSet(), NewList()), suite.vs, func(v Value, r *Ref) bool {
		actual++
		return true
	}, 1)
	suite.Equal(1, actual)
}

func (suite *WalkTestSuite) skipWorker(composite Value) (reached []Value) {
	SomeP(composite, suite.vs, func(v Value, r *Ref) bool {
		suite.False(v.Equals(suite.deadValue), "Should never have reached %+v", suite.deadValue)
		reached = append(reached, v)
		return v.Equals(suite.mustSkip)
	}, 1)
	return
}

// Skipping a sub-tree must allow other items in the list to be processed.
func (suite *WalkTestSuite) SkipTestSkipListElement() {
	wholeList := NewList(suite.mustSkip, suite.shouldSee, suite.shouldSee)
	reached := suite.skipWorker(wholeList)
	for _, v := range []Value{wholeList, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 6)
}

func (suite *WalkTestSuite) SkipTestSkipSetElement() {
	wholeSet := NewSet(suite.mustSkip, suite.shouldSee).Insert(suite.shouldSee)
	reached := suite.skipWorker(wholeSet)
	for _, v := range []Value{wholeSet, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 4)
}

func (suite *WalkTestSuite) SkipTestSkipMapValue() {
	shouldAlsoSeeItem := String("Also good")
	shouldAlsoSee := NewSet(shouldAlsoSeeItem)
	wholeMap := NewMap(suite.shouldSee, suite.mustSkip, shouldAlsoSee, suite.shouldSee)
	reached := suite.skipWorker(wholeMap)
	for _, v := range []Value{wholeMap, suite.shouldSee, suite.shouldSeeItem, suite.mustSkip, shouldAlsoSee, shouldAlsoSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 8)
}

func (suite *WalkTestSuite) SkipTestSkipMapKey() {
	wholeMap := NewMap(suite.mustSkip, suite.shouldSee, suite.shouldSee, suite.shouldSee)
	reached := suite.skipWorker(wholeMap)
	for _, v := range []Value{wholeMap, suite.mustSkip, suite.shouldSee, suite.shouldSeeItem} {
		suite.Contains(reached, v, "Doesn't contain %+v", v)
	}
	suite.Len(reached, 8)
}
