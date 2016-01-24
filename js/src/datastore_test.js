// @flow

import {suite, test} from 'mocha';
import {assert} from 'chai';
import Chunk from './chunk.js';
import MemoryStore from './memory_store.js';
import {Commit, DataStore} from './datastore.js';

suite('DataStore', () => {
  test('access', async () => {
    const ms = new MemoryStore();
    const ds = new DataStore(ms);
    const input = 'abc';

    const c = Chunk.fromString(input);
    let c1 = await ds.get(c.ref);
    assert.isTrue(c1.isEmpty());

    let has = await ds.has(c.ref);
    assert.isFalse(has);

    ds.put(c);
    c1 = await ds.get(c.ref);
    assert.isFalse(c1.isEmpty());

    has = await ds.has(c.ref);
    assert.isTrue(has);
  });

  test('commit', async () => {
    const ms = new MemoryStore();
    const ds = new DataStore(ms);

    const datasets = ds.datasets;
    assert.strictEqual(0, datasets.size);

    // |a|
    const aCommit = new Commit([], 'a');


    const ds2, err := ds.Commit(datasetID, aCommit)
    assert.NoError(err)

    // The old datastore still still has no head.
    _, ok := ds.MaybeHead(datasetID)
    assert.False(ok)

    // The new datastore has |a|.
    aCommit1 := ds2.Head(datasetID)
    assert.True(aCommit1.Value().Equals(a))
    ds = ds2

    // |a| <- |b|
    b := types.NewString("b")
    bCommit := NewCommit(cs).SetValue(b).SetParents(NewSetOfRefOfCommit(cs).Insert(NewRefOfCommit(aCommit.Ref())))
    ds, err = ds.Commit(datasetID, bCommit)
    assert.NoError(err)
    assert.True(ds.Head(datasetID).Value().Equals(b))

    // |a| <- |b|
    //   \----|c|
    // Should be disallowed.
    c := types.NewString("c")
    cCommit := NewCommit(cs).SetValue(c)
    ds, err = ds.Commit(datasetID, cCommit)
    assert.Error(err)
    assert.True(ds.Head(datasetID).Value().Equals(b))

    // |a| <- |b| <- |d|
    d := types.NewString("d")
    dCommit := NewCommit(cs).SetValue(d).SetParents(NewSetOfRefOfCommit(cs).Insert(NewRefOfCommit(bCommit.Ref())))
    ds, err = ds.Commit(datasetID, dCommit)
    assert.NoError(err)
    assert.True(ds.Head(datasetID).Value().Equals(d))

    // Attempt to recommit |b| with |a| as parent.
    // Should be disallowed.
    ds, err = ds.Commit(datasetID, bCommit)
    assert.Error(err)
    assert.True(ds.Head(datasetID).Value().Equals(d))

    // Add a commit to a different datasetId
    _, err = ds.Commit("otherDs", aCommit)
    assert.NoError(err)

    // Get a fresh datastore, and verify that both datasets are present
    newDs := NewDataStore(cs)
    datasets2 := newDs.Datasets()
    assert.Equal(uint64(2), datasets2.Len())
  });
});
