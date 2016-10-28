// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"errors"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
)

// ResourceCache is a Map<String, Ref<Blob>>
type ResourceCache struct {
	cache types.Map
	orig  types.Map
	mutex sync.Mutex
}

func GetResourceCache(db datas.Database, dsname string) (*ResourceCache, error) {
	m, ok := db.GetDataset(dsname).MaybeHeadValue()
	if ok {
		refOfBlobType := types.MakeRefType(types.BlobType)
		mapOfStringToRefOfBlobType := types.MakeMapType(types.StringType, refOfBlobType)
		if !types.IsSubtype(mapOfStringToRefOfBlobType, m.Type()) {
			return nil, errors.New("commit value in cache-ds must be Map<String, Ref<Blob>>")
		}
	} else {
		m = types.NewMap()
	}
	return &ResourceCache{cache: m.(types.Map), orig: m.(types.Map)}, nil
}

func (c *ResourceCache) Commit(db datas.Database, dsname string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.cache.Equals(c.orig) {
		meta, _ := spec.CreateCommitMetaStruct(db, "", "", nil, nil)
		dset := db.GetDataset(dsname)
		commitOptions := datas.CommitOptions{Meta: meta}
		_, err := db.Commit(dset, c.cache, commitOptions)
		return err
	}
	return nil
}

func (c *ResourceCache) Get(k types.String) (types.Ref, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if v, ok := c.cache.MaybeGet(k); ok {
		return v.(types.Ref), true
	}
	return types.Ref{}, false
}

func (c *ResourceCache) Set(k types.String, v types.Ref) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = c.cache.Set(k, v)
}

func (c *ResourceCache) Len() uint64 {
	return c.cache.Len()
}
