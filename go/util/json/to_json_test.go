// Copyright 2019 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"bytes"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/suite"
)

func TestToJSONSuite(t *testing.T) {
	suite.Run(t, &ToJSONSuite{})
}

type ToJSONSuite struct {
	suite.Suite
	vs *types.ValueStore
}

func (suite *ToJSONSuite) SetupTest() {
	st := &chunks.TestStorage{}
	suite.vs = types.NewValueStore(st.NewView())
}

func (suite *ToJSONSuite) TearDownTest() {
	suite.vs.Close()
}

func (suite *ToJSONSuite) TestToJSON() {
	tc := []struct {
		desc     string
		in       types.Value
		opts     ToOptions
		exp      string
		expError string
	}{
		{"true", types.Bool(true), ToOptions{}, "true", ""},
		{"false", types.Bool(false), ToOptions{}, "false", ""},
		{"42", types.Number(42), ToOptions{}, "42", ""},
		{"88.8", types.Number(88.8), ToOptions{}, "88.8", ""},
		{"empty string", types.String(""), ToOptions{}, `""`, ""},
		{"foobar", types.String("foobar"), ToOptions{}, `"foobar"`, ""},
		{"strings with newlines", types.String(`"\nmonkey`), ToOptions{}, `"\"\\nmonkey"`, ""},
		{"structs when not enabled", types.NewStruct("", types.StructData{}), ToOptions{}, "", "Struct marshaling not enabled"},
		{"named struct", types.NewStruct("Person", types.StructData{}), ToOptions{Structs: true}, "", "Named struct marshaling not supported"},
		{"struct nested errors", types.NewStruct("", types.StructData{"foo": types.NewList(suite.vs)}), ToOptions{Structs: true}, "", "List marshaling not enabled"},
		{"empty struct", types.NewStruct("", types.StructData{}), ToOptions{Structs: true}, "{}", ""},
		{"non-empty struct", types.NewStruct("", types.StructData{"str": types.String("bar"), "num": types.Number(42)}), ToOptions{Structs: true}, `{"num":42,"str":"bar"}`, ""},
		{"list when not enabled", types.NewList(suite.vs), ToOptions{}, "", "List marshaling not enabled"},
		{"list nested errors", types.NewList(suite.vs, types.NewSet(suite.vs)), ToOptions{Lists: true}, "", "Set marshaling not enabled"},
		{"empty list", types.NewList(suite.vs), ToOptions{Lists: true}, "[]", ""},
		{"non-empty list", types.NewList(suite.vs, types.Number(42), types.String("foo")), ToOptions{Lists: true}, `[42,"foo"]`, ""},
		{"sets when not enabled", types.NewSet(suite.vs), ToOptions{}, "", "Set marshaling not enabled"},
		{"set nested errors", types.NewSet(suite.vs, types.NewList(suite.vs)), ToOptions{Sets: true}, "", "List marshaling not enabled"},
		{"empty set", types.NewSet(suite.vs), ToOptions{Sets: true}, "[]", ""},
		{"non-empty set", types.NewSet(suite.vs, types.Number(42), types.String("foo")), ToOptions{Sets: true}, `[42,"foo"]`, ""},
		{"maps when not enabled", types.NewMap(suite.vs), ToOptions{}, "", "Map marshaling not enabled"},
		{"map nested errors", types.NewMap(suite.vs, types.String("foo"), types.NewSet(suite.vs)), ToOptions{Maps: true}, "", "Set marshaling not enabled"},
		{"map non-string key", types.NewMap(suite.vs, types.Number(42), types.Number(42)), ToOptions{Maps: true}, "", "Map key kind Number not supported"},
		{"empty map", types.NewMap(suite.vs), ToOptions{Maps: true}, "{}", ""},
		{"non-empty map", types.NewMap(suite.vs, types.String("foo"), types.String("bar"), types.String("baz"), types.Number(42)), ToOptions{Maps: true}, `{"baz":42,"foo":"bar"}`, ""},
		{"complex value", types.NewStruct("", types.StructData{
			"list": types.NewList(suite.vs,
				types.NewSet(suite.vs,
					types.NewMap(suite.vs, types.String("foo"), types.String("bar"), types.String("hot"), types.Number(42))))}), ToOptions{Structs: true, Lists: true, Sets: true, Maps: true}, `{"list":[[{"foo":"bar","hot":42}]]}`, ""},
	}

	for _, t := range tc {
		buf := &bytes.Buffer{}
		err := ToJSON(t.in, buf, t.opts)
		if t.expError != "" {
			suite.EqualError(err, t.expError, t.desc)
			suite.Equal("", string(buf.Bytes()), t.desc)
		} else {
			suite.NoError(err)
			suite.Equal(t.exp+"\n", string(buf.Bytes()), t.desc)
		}
	}
}
