// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
	"github.com/julienschmidt/httprouter"
)

// Database provides versioned storage for noms values. Each Database instance represents one moment in history. Heads() returns the Commit from each active fork at that moment. The Commit() method returns a new Database, representing a new moment in history.
type RemoteDatabaseClient struct {
	databaseCommon
	httpBS *httpBatchStore
}

func NewRemoteDatabase(baseURL, auth string) *RemoteDatabaseClient {
	httpBS := newHTTPBatchStore(baseURL, auth)
	return &RemoteDatabaseClient{
		newDatabaseCommon(newCachingChunkHaver(httpBS), types.NewValueStore(httpBS), httpBS),
		httpBS,
	}
}

func (rdb *RemoteDatabaseClient) validatingBatchStore() (bs types.BatchStore) {
	bs = rdb.vs.BatchStore()
	d.Chk.True(bs.IsValidating())
	return
}

func (rdb *RemoteDatabaseClient) Commit(datasetID string, commit types.Struct, progressChan chan CommitProgress) (Database, error) {
	rdb.httpBS.progressChan = progressChan
	defer func() {
		rdb.httpBS.progressChan = nil
	}()
	err := rdb.commit(datasetID, commit)
	return &RemoteDatabaseClient{
		newDatabaseCommon(rdb.cch, rdb.vs, rdb.rt),
		rdb.httpBS,
	}, err
}

func (rdb *RemoteDatabaseClient) Delete(datasetID string) (Database, error) {
	err := rdb.doDelete(datasetID)
	return &RemoteDatabaseClient{
		newDatabaseCommon(rdb.cch, rdb.vs, rdb.rt),
		rdb.httpBS,
	}, err
}

func (f RemoteStoreFactory) CreateStore(ns string) Database {
	return NewRemoteDatabase(f.host+httprouter.CleanPath(ns), f.auth)
}

func (f RemoteStoreFactory) Create(ns string) (Database, bool) {
	if ds := f.CreateStore(ns); ds != nil {
		return ds, true
	}
	return &LocalDatabase{}, false
}

func (f RemoteStoreFactory) Shutter() {}

func NewRemoteStoreFactory(host, auth string) Factory {
	return RemoteStoreFactory{host: host, auth: auth}
}

type RemoteStoreFactory struct {
	host string
	auth string
}
