// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
)

// Database provides versioned storage for noms values. Each Database instance represents one moment in history. Heads() returns the Commit from each active fork at that moment. The Commit() method returns a new Database, representing a new moment in history.
type LocalDatabase struct {
	databaseCommon
	cs   chunks.ChunkStore
	lvbs types.BatchStore
}

func newLocalDatabase(cs chunks.ChunkStore) *LocalDatabase {
	return &LocalDatabase{
		newDatabaseCommon(newCachingChunkHaver(cs), types.NewValueStore(types.NewBatchStoreAdapter(cs)), cs),
		cs,
		nil,
	}
}

func (lds *LocalDatabase) Commit(datasetID string, commit types.Struct) (Database, error) {
	err := lds.commit(datasetID, commit)
	lds.vs.Flush()
	return &LocalDatabase{
		newDatabaseCommon(lds.cch, lds.vs, lds.rt),
		lds.cs,
		nil,
	}, err
}

func (lds *LocalDatabase) Delete(datasetID string) (Database, error) {
	err := lds.doDelete(datasetID)
	lds.vs.Flush()
	return &LocalDatabase{newDatabaseCommon(lds.cch, lds.vs, lds.rt), lds.cs, nil}, err
}

func (lds *LocalDatabase) ValidatingBatchStore() types.BatchStore {
	return lds.vs.ValidatingBatchStore()
}
