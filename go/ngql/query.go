// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/graphql-go/graphql"
)

const (
	atKey          = "at"
	countKey       = "count"
	keyKey         = "key"
	queryKey       = "Query"
	targetHashKey  = "targetHash"
	targetValueKey = "targetValue"
	valueKey       = "value"
	vrKey          = "vr"
	tmKey          = "tm"
)

func constructQueryType(rootValue types.Value, tm typeMap) *graphql.Object {
	getValueField := func(root interface{}, fieldName string, ctx context.Context) types.Value {
		m := root.(map[string]interface{})
		v := m[fieldName]
		return v.(types.Value)
	}

	rootTyp := rootValue.Type()
	rootType := nomsTypeToGraphQLType(rootTyp, tm)
	args, resolveFn := getArgsAndResolveFn(rootTyp.Kind(), getValueField)

	return graphql.NewObject(graphql.ObjectConfig{
		Name: queryKey,
		Fields: graphql.Fields{
			valueKey: &graphql.Field{
				Type:    rootType,
				Args:    args,
				Resolve: resolveFn,
			},
		}})
}

func Query(rootValue types.Value, query string, vr types.ValueReader, w io.Writer) error {
	tm := typeMap{}

	queryObj := constructQueryType(rootValue, tm)
	schemaConfig := graphql.SchemaConfig{Query: queryObj}
	schema, _ := graphql.NewSchema(schemaConfig)
	ctx := context.WithValue(context.WithValue(context.Background(), vrKey, vr), tmKey, tm)

	r := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
		RootObject: map[string]interface{}{
			valueKey: rootValue,
		},
		Context: ctx,
	})

	rJSON, _ := json.Marshal(r)
	io.Copy(w, bytes.NewBuffer([]byte(rJSON)))
	return nil
}
