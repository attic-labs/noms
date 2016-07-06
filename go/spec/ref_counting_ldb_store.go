// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type refCountingLdbStore struct {
	store    *chunks.LevelDBStore
	refCount int
	closeFn  func()
}

func newRefCountingLdbStore(path string, closeFn func()) *refCountingLdbStore {
	return &refCountingLdbStore{chunks.NewLevelDBStoreUseFlags(path, ""), 1, closeFn}
}

func (r *refCountingLdbStore) Get(h hash.Hash) chunks.Chunk {
	return r.store.Get(h)
}

func (r *refCountingLdbStore) Has(h hash.Hash) bool {
	return r.store.Has(h)
}

func (r *refCountingLdbStore) Version() string {
	return r.store.Version()
}

func (r *refCountingLdbStore) Put(c chunks.Chunk) {
	r.store.Put(c)
}

func (r *refCountingLdbStore) PutMany(chunks []chunks.Chunk) chunks.BackpressureError {
	return r.store.PutMany(chunks)
}

func (r *refCountingLdbStore) Root() hash.Hash {
	return r.store.Root()
}

func (r *refCountingLdbStore) UpdateRoot(current, last hash.Hash) bool {
	return r.store.UpdateRoot(current, last)
}

func (r *refCountingLdbStore) AddRef() {
	r.refCount++
}

func (r *refCountingLdbStore) Close() (err error) {
	d.Chk.True(r.refCount > 0)
	r.refCount--
	if r.refCount == 0 {
		err = r.store.Close()
		r.closeFn()
	}
	return
}
