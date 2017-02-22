// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import "github.com/attic-labs/noms/go/types"

const (
	schemaField = "schema"
	metaField   = "meta"
)

// getCommitSchema gets the schema field from the meta struct of a commit.
func getCommitSchema(commit types.Value) *types.Type {
	if commit, ok := commit.(types.Struct); ok {
		if meta, ok := commit.MaybeGet(metaField); ok {
			if meta, ok := meta.(types.Struct); ok {
				if schema, ok := meta.MaybeGet(schemaField); ok {
					if schema, ok := schema.(*types.Type); ok {
						return schema
					}
				}
			}
		}
	}
	return nil
}
