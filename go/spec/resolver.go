// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"fmt"
	"strings"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
)

type Resolver struct {
	config      *Config
	deferredErr error
}


func NewResolver() *Resolver {
	c, err := FindNomsConfig()
	if err != nil {
		if err != NoConfig {
			return &Resolver{ deferredErr: err }
		}
		return &Resolver{}
	}
	return &Resolver{ c, nil }
}


func (r *Resolver) verbose(orig string, replacement string) string {
	if Verbose() && orig != replacement {
		if orig == "" {
			orig = `""`
		}
		fmt.Printf("\t%s -> %s\n", orig, replacement)
	}
	return replacement
}

func (r *Resolver) ResolveDatabase(str string) string {
	if r.config != nil {
		if str == "" {
			return r.config.Default.Url
		}
		if val, ok := r.config.Db[str]; ok {
			return val.Url
		}
	}
	return str
}

func (r *Resolver) ResolvePath(str string) string {
	if r.config != nil {
		split := strings.SplitN(str, separator, 2)
		db, rest := "", split[0]
		if len(split) > 1 {
			db, rest = split[0], split[1]
		}
		return r.ResolveDatabase(db)+separator+rest
	}
	return str
}

// Resolve string to database spec. If a config is present,
//   - resolve a db alias to its db spec
//   - resolve "" to the default db spec
func (r *Resolver) GetDatabase(str string) (datas.Database, error) {
	if r.deferredErr != nil {
		return nil, r.deferredErr
	}
	return GetDatabase(r.verbose(str, r.ResolveDatabase(str)))
}

// Resolve string to a chunkstore. Like ResolveDatabase, but returns the underlying ChunkStore
func (r *Resolver) GetChunkStore(str string) (chunks.ChunkStore, error) {
	if r.deferredErr != nil {
		return nil, r.deferredErr
	}
	return GetChunkStore(r.verbose(str, r.ResolveDatabase(str)))
}

// Resolve string to a dataset. If a config is present,
//  - if no db prefix is present, assume the default db
//  - if the db prefix is an alias, replace it
func (dsr *Resolver) GetDataset(str string) (datas.Database, datas.Dataset, error) {
	if dsr.deferredErr != nil {
		return datas.Dataset{}, dsr.deferredErr
	}
	return GetDataset(dsr.ResolvePath(str))
}

// Resolve string to a value path. If a config is present,
//  - if no db spec is present, assume the default db
//  - if the db spec is an alias, replace it
func (r *Resolver) GetPath(str string) (datas.Database, types.Value, error) {
	if r.deferredErr != nil {
		return nil, nil, r.deferredErr
	}
	return GetPath(r.verbose(str, r.ResolvePath(str)))
}
