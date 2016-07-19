// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// Database provides versioned storage for noms values. Each Database instance represents one moment in history. Heads() returns the Commit from each active fork at that moment. The Commit() method returns a new Database, representing a new moment in history.
type LocalDatabase struct {
	databaseCommon
	cs chunks.ChunkStore
}

func newLocalDatabase(cs chunks.ChunkStore) *LocalDatabase {
	bs := types.NewBatchStoreAdaptor(cs)
	return &LocalDatabase{
		newDatabaseCommon(newCachingChunkHaver(cs), types.NewValueStore(bs), bs),
		cs,
	}
}

func (ldb *LocalDatabase) Commit(datasetID string, commit types.Struct, progressCh chan CommitProgress) (Database, error) {
	err := ldb.commit(datasetID, commit)
	return &LocalDatabase{newDatabaseCommon(ldb.cch, ldb.vs, ldb.rt), ldb.cs}, err
}

func (ldb *LocalDatabase) Delete(datasetID string) (Database, error) {
	err := ldb.doDelete(datasetID)
	return &LocalDatabase{newDatabaseCommon(ldb.cch, ldb.vs, ldb.rt), ldb.cs}, err
}

func (ldb *LocalDatabase) validatingBatchStore() (bs types.BatchStore) {
	bs = ldb.vs.BatchStore()
	if !bs.IsValidating() {
		bs = newLocalBatchStore(ldb.cs)
		ldb.vs = types.NewValueStore(bs)
		ldb.rt = bs
	}
	d.Chk.True(bs.IsValidating())
	return bs
}
