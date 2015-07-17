package datas

import (
	"github.com/attic-labs/noms/chunks"
	. "github.com/attic-labs/noms/dbg"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

type DataStore struct {
	chunks.ChunkStore

	rt    chunks.RootTracker
	rc    *rootCache
	roots RootSet
}

func NewDataStore(cs chunks.ChunkStore, rt chunks.RootTracker) DataStore {
	return newDataStoreInternal(cs, rt, NewRootCache(cs))
}

func newDataStoreInternal(cs chunks.ChunkStore, rt chunks.RootTracker, rc *rootCache) DataStore {
	return DataStore{
		cs, rt, rc, rootSetFromRef(rt.Root(), cs),
	}
}

func rootSetFromRef(rootRef ref.Ref, cs chunks.ChunkSource) RootSet {
	var roots RootSet
	if (rootRef == ref.Ref{}) {
		roots = NewRootSet()
	} else {
		roots = RootSetFromVal(types.MustReadValue(rootRef, cs).(types.Set))
	}

	return roots
}

func (ds *DataStore) Roots() RootSet {
	return ds.roots
}

func (ds *DataStore) Commit(newRoots RootSet) DataStore {
	Chk.True(newRoots.Len() > 0)
	for !ds.doCommit(newRoots) {
	}
	return newDataStoreInternal(ds.ChunkStore, ds.rt, ds.rc)
}

func (ds *DataStore) doCommit(roots RootSet) bool {
	Chk.True(roots.Len() > 0)

	currentRootRef := ds.rt.Root()

	// Note that |currentRoots| may be different from |ds.Roots| if someone else has commited since this Datastore was created. This computation must be based on the *current root* not the root associated with this Datastore.
	var currentRoots RootSet
	if currentRootRef == ds.roots.Ref() {
		currentRoots = ds.roots
	} else {
		currentRoots = rootSetFromRef(currentRootRef, ds)
	}

	newRoots := roots.Union(currentRoots)

	roots.Iter(func(root Root) (stop bool) {
		if ds.isPrexisting(root, currentRoots) {
			newRoots = newRoots.Remove(root)
		} else {
			newRoots = RootSetFromVal(newRoots.NomsValue().Subtract(root.Parents()))
		}

		return
	})

	if newRoots.Len() == 0 || newRoots.Equals(currentRoots) {
		return true
	}

	// TODO: This set will be orphaned if this UpdateRoot below fails
	newRootRef, err := types.WriteValue(newRoots.NomsValue(), ds)
	Chk.NoError(err)

	return ds.rt.UpdateRoot(newRootRef, currentRootRef)
}

func (ds *DataStore) isPrexisting(root Root, currentRoots RootSet) bool {
	if currentRoots.Has(root) {
		return true
	}

	// If a new root superceeds a (non-reachable) current root, it can't have already been committed because it hash would be uncomputable
	superceedsCurrentRoot := false
	root.Parents().Iter(func(parent types.Value) (stop bool) {
		superceedsCurrentRoot = currentRoots.Has(RootFromVal(parent))
		return superceedsCurrentRoot
	})
	if superceedsCurrentRoot {
		return false
	}

	ds.rc.Update(currentRoots)
	return ds.rc.Contains(root.Ref())
}
