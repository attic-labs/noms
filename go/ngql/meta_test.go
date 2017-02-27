// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

func makeMetaStructWithSchema(schema types.Value) types.Struct {
	return types.NewStruct("Meta", types.StructData{
		schemaField: schema,
	})
}

// Cannot depend on datas so mock a commit.
func makeCommit(value, meta types.Value) types.Struct {
	return types.NewStruct("Commit", types.StructData{
		"value":   value,
		"parents": types.NewSet(),
		"meta":    meta,
	})
}

func TestCommitWithSchema(t *testing.T) {
	assert := assert.New(t)

	assertTypeEquals := func(e, a *types.Type) {
		assert.True(a.Equals(e), "Actual: %s\nExpected %s", a.Describe(), e.Describe())
	}

	assert.Nil(getCommitSchema(types.Number(1)))

	assert.Nil(getCommitSchema(types.NewStruct("", types.StructData{})))

	commit := makeCommit(types.Number(1), types.EmptyStruct)
	schemaType := getCommitSchema(commit)
	assert.Nil(schemaType)

	commit = makeCommit(types.Number(1), makeMetaStructWithSchema(types.String("abc")))
	schemaType = getCommitSchema(commit)
	assert.Nil(schemaType)

	commit = makeCommit(types.Number(1), makeMetaStructWithSchema(types.NumberType))
	schemaType = getCommitSchema(commit)
	assertTypeEquals(types.NumberType, schemaType)
}
