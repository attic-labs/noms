// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"testing"

	"github.com/attic-labs/testify/suite"
)

type testValue struct {
	value       Value
	expectedRef string
	description string
}

type testSuite struct {
	suite.Suite
	testValues []*testValue
}

// please update Go and JS to keep them in sync - see js/src//xp-test.js
func newTestSuite() *testSuite {
	testValues := []*testValue{
		&testValue{Bool(true), "3753t1pev51ajbmitxl6ykqeryx72q128b0mm9fqk9t89wcvu6", "bool - true"},
		&testValue{Bool(false), "2cxf1z91qo0h4e3irlm7y9bp0sd4vsbvizisup3g1tzt2xc4au", "bool - false"},
		&testValue{Number(-1), "5xqj6rxt33yper4t7epvl5e80f7uz8q2v1j93tkyt9jwpcleon", "num - -1"},
		&testValue{Number(0), "06flsmqayz48yh4p47383sbgpgqo7t8senfogzv7s6swcn9qx8", "num - 0"},
		&testValue{Number(1), "5ea6wxe8o8vk067otqo8cmom9d8rgpjs5mi4mdr8alldtdv77u", "num - 1"},
		&testValue{String(""), "5rdmp2c6fd6mjlcouryxs8g9yqdl1320uw9j70rw9n05h41hbk", "str - empty"},
		&testValue{String("0"), "3xnohwqoivqfjev41d13tozncju9y7h6u50v71kp9tzpq8bx0k", "str - 0"},
		&testValue{String("false"), "0e5l07lb03e6n46h5u12kilyg43o0ev9l6f4pib6s2w0elfrfq", "str - false"},
	}

	// TODO: add these types too
	/*
		BlobKind
		ValueKind
		ListKind
		MapKind
		RefKind
		SetKind
		StructKind
		TypeKind
		CycleKind // Only used in encoding/decoding.
		UnionKind
	*/

	return &testSuite{testValues: testValues}
}

// write a value, read that value back out
// assert the values are equal and
// verify the digest is what we expect
func (suite *testSuite) roundTripDigestTest(t *testValue) {
	vs := NewTestValueStore()
	r := vs.WriteValue(t.value)
	v2 := vs.ReadValue(r.TargetHash())

	suite.True(v2.Equals(t.value), t.description)
	suite.True(t.value.Equals(v2), t.description)
	suite.Equal(t.expectedRef, r.TargetHash().String(), t.description)
}

// Called from testify suite.Run()
func (suite *testSuite) TestTypes() {
	for i := range suite.testValues {
		suite.roundTripDigestTest(suite.testValues[i])
	}
}

// Called from "go test"
func TestSuite(t *testing.T) {
	suite.Run(t, newTestSuite())
}
