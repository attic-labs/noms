package dataset

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/types"
)

func TestDatasetCommitTracker(t *testing.T) {
	assert := assert.New(t)
	id1 := "testdataset"
	id2 := "othertestdataset"
	ms := chunks.NewMemoryStore()

	ds1 := NewDataset(datas.NewDataStore(ms), id1)
	ds1Commit := types.NewString("Commit value for " + id1)
	ds1, ok := ds1.Commit(ds1Commit)
	assert.True(ok)

	ds2 := NewDataset(datas.NewDataStore(ms), id2)
	ds2Commit := types.NewString("Commit value for " + id2)
	ds2, ok = ds2.Commit(ds2Commit)
	assert.True(ok)

	assert.EqualValues(ds1Commit, ds1.Head().Value().GetValue(ms))
	assert.EqualValues(ds2Commit, ds2.Head().Value().GetValue(ms))
	assert.False(ds2.Head().Value().Equals(ds1Commit))
	assert.False(ds1.Head().Value().Equals(ds2Commit))

	assert.Equal("sha1-d6e0164c200b12e1f849de26a587b02625ce56a0", ms.Root().String())
}

func newDS(id string, ms *chunks.MemoryStore) Dataset {
	store := datas.NewDataStore(ms)
	return NewDataset(store, id)
}

func TestExplicitBranchUsingDatasets(t *testing.T) {
	assert := assert.New(t)
	id1 := "testdataset"
	id2 := "othertestdataset"
	ms := chunks.NewMemoryStore()

	ds1 := newDS(id1, ms)

	// ds1: |a|
	a := types.NewString("a")
	ds1, ok := ds1.Commit(a)
	assert.True(ok)
	assert.True(ds1.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))

	// ds1: |a|
	//        \ds2
	ds2 := newDS(id2, ms)
	ds2, ok = ds2.Commit(ds1.Head().Value().GetValue(ms))
	assert.True(ok)
	assert.True(ds2.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))

	// ds1: |a| <- |b|
	b := types.NewString("b")
	ds1, ok = ds1.Commit(b)
	assert.True(ok)
	assert.True(ds1.Head().Value().Equals(datas.NewRefOfValue(b.Ref())))

	// ds1: |a|    <- |b|
	//        \ds2 <- |c|
	c := types.NewString("c")
	ds2, ok = ds2.Commit(c)
	assert.True(ok)
	assert.True(ds2.Head().Value().Equals(datas.NewRefOfValue(c.Ref())))

	// ds1: |a|    <- |b| <--|d|
	//        \ds2 <- |c| <--/
	mergeParents := datas.NewSetOfRefOfCommit().Insert(datas.NewRefOfCommit(ds1.Head().Ref())).Insert(datas.NewRefOfCommit(ds2.Head().Ref()))
	d := types.NewString("d")
	ds2, ok = ds2.CommitWithParents(d, mergeParents)
	assert.True(ok)
	assert.True(ds2.Head().Value().Equals(datas.NewRefOfValue(d.Ref())))

	ds1, ok = ds1.CommitWithParents(d, mergeParents)
	assert.True(ok)
	assert.True(ds1.Head().Value().Equals(datas.NewRefOfValue(d.Ref())))
}

func TestTwoClientsWithEmptyDataset(t *testing.T) {
	assert := assert.New(t)
	id1 := "testdataset"
	ms := chunks.NewMemoryStore()

	dsx := newDS(id1, ms)
	dsy := newDS(id1, ms)

	// dsx: || -> |a|
	a := types.NewString("a")
	dsx, ok := dsx.Commit(a)
	assert.True(ok)
	assert.True(dsx.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))

	// dsy: || -> |b|
	_, ok = dsy.MaybeHead()
	assert.False(ok)
	b := types.NewString("b")
	dsy, ok = dsy.Commit(b)
	assert.False(ok)
	// Commit failed, but ds1 now has latest head, so we should be able to just try again.
	// dsy: |a| -> |b|
	dsy, ok = dsy.Commit(b)
	assert.True(ok)
	assert.True(dsy.Head().Value().Equals(datas.NewRefOfValue(b.Ref())))
}

func TestTwoClientsWithNonEmptyDataset(t *testing.T) {
	assert := assert.New(t)
	id1 := "testdataset"
	ms := chunks.NewMemoryStore()

	a := types.NewString("a")
	{
		// ds1: || -> |a|
		ds1 := newDS(id1, ms)
		ds1, ok := ds1.Commit(a)
		assert.True(ok)
		assert.True(ds1.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))
	}

	dsx := newDS(id1, ms)
	dsy := newDS(id1, ms)

	// dsx: |a| -> |b|
	assert.True(dsx.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))
	b := types.NewString("b")
	dsx, ok := dsx.Commit(b)
	assert.True(ok)
	assert.True(dsx.Head().Value().Equals(datas.NewRefOfValue(b.Ref())))

	// dsy: |a| -> |c|
	assert.True(dsy.Head().Value().Equals(datas.NewRefOfValue(a.Ref())))
	c := types.NewString("c")
	dsy, ok = dsy.Commit(c)
	assert.False(ok)
	assert.True(dsy.Head().Value().Equals(datas.NewRefOfValue(b.Ref())))
	// Commit failed, but dsy now has latest head, so we should be able to just try again.
	// dsy: |b| -> |c|
	dsy, ok = dsy.Commit(c)
	assert.True(ok)
	assert.True(dsy.Head().Value().Equals(datas.NewRefOfValue(c.Ref())))
}
