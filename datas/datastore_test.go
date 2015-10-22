package datas

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/types"
)

func TestDataStoreCommit(t *testing.T) {
	assert := assert.New(t)
	chunks := chunks.NewMemoryStore()
	ds := NewDataStore(chunks)
	datasetID := "ds1"

	datasets := ds.Datasets()
	assert.Zero(datasets.Len())

	// |a|
	a := types.NewString("a")
	ra := NewRefOfValue(types.WriteValue(a, chunks))
	aCommit := NewCommit().SetValue(ra)
	ds2, ok := ds.Commit(datasetID, aCommit)
	assert.True(ok)

	// The old datastore still still has no head.
	_, ok = ds.MaybeHead(datasetID)
	assert.False(ok)

	// The new datastore has |a|.
	aCommit1 := ds2.Head(datasetID)
	assert.True(aCommit1.Value().Equals(ra))
	ds = ds2

	// |a| <- |b|
	b := types.NewString("b")
	rb := NewRefOfValue(types.WriteValue(b, chunks))
	bCommit := NewCommit().SetValue(rb).SetParents(NewSetOfRefOfCommit().Insert(NewRefOfCommit(aCommit.Ref())))
	ds, ok = ds.Commit(datasetID, bCommit)
	assert.True(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rb))

	// |a| <- |b|
	//   \----|c|
	// Should be disallowed.
	c := types.NewString("c")
	rc := NewRefOfValue(types.WriteValue(c, chunks))
	cCommit := NewCommit().SetValue(rc)
	ds, ok = ds.Commit(datasetID, cCommit)
	assert.False(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rb))

	// |a| <- |b| <- |d|
	d := types.NewString("d")
	rd := NewRefOfValue(types.WriteValue(d, chunks))
	dCommit := NewCommit().SetValue(rd).SetParents(NewSetOfRefOfCommit().Insert(NewRefOfCommit(bCommit.Ref())))
	ds, ok = ds.Commit(datasetID, dCommit)
	assert.True(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rd))

	// Attempt to recommit |b| with |a| as parent.
	// Should be disallowed.
	ds, ok = ds.Commit(datasetID, bCommit)
	assert.False(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rd))

	// Add a commit to a different datasetId
	_, ok = ds.Commit("otherDs", aCommit)
	assert.True(ok)

	// Get a fresh datastore, and verify that both datasets are present
	newDs := NewDataStore(chunks)
	datasets2 := newDs.Datasets()
	assert.Equal(uint64(2), datasets2.Len())
}

func TestDataStoreConcurrency(t *testing.T) {
	assert := assert.New(t)

	chunks := chunks.NewMemoryStore()
	ds := NewDataStore(chunks)
	datasetID := "ds1"

	// Setup:
	// |a| <- |b|
	a := types.NewString("a")
	ra := NewRefOfValue(types.WriteValue(a, chunks))
	aCommit := NewCommit().SetValue(ra)
	ds, ok := ds.Commit(datasetID, aCommit)
	b := types.NewString("b")
	rb := NewRefOfValue(types.WriteValue(b, chunks))
	bCommit := NewCommit().SetValue(rb).SetParents(NewSetOfRefOfCommit().Insert(NewRefOfCommit(aCommit.Ref())))
	ds, ok = ds.Commit(datasetID, bCommit)
	assert.True(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rb))

	// Important to create this here.
	ds2 := NewDataStore(chunks)

	// Change 1:
	// |a| <- |b| <- |c|
	c := types.NewString("c")
	rc := NewRefOfValue(types.WriteValue(c, chunks))
	cCommit := NewCommit().SetValue(rc).SetParents(NewSetOfRefOfCommit().Insert(NewRefOfCommit(bCommit.Ref())))
	ds, ok = ds.Commit(datasetID, cCommit)
	assert.True(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rc))

	// Change 2:
	// |a| <- |b| <- |e|
	// Should be disallowed, DataStore returned by Commit() should have |c| as Head.
	e := types.NewString("e")
	re := NewRefOfValue(types.WriteValue(e, chunks))
	eCommit := NewCommit().SetValue(re).SetParents(NewSetOfRefOfCommit().Insert(NewRefOfCommit(bCommit.Ref())))
	ds2, ok = ds2.Commit(datasetID, eCommit)
	assert.False(ok)
	assert.True(ds.Head(datasetID).Value().Equals(rc))
}
