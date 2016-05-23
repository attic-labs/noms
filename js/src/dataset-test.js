// @flow

import {suite, test} from 'mocha';
import {makeTestingBatchStore} from './batch-store-adaptor.js';
import {assert} from 'chai';
import Dataset from './dataset.js';
import Database from './database.js';
import {invariant, notNull} from './assert.js';

suite('Dataset', () => {
  test('commit', async () => {
    const bs = makeTestingBatchStore();
    const db = new Database(bs);
    let ds = new Dataset(db, 'ds1');

    // |a|
    const ds2 = await ds.commit('a');

    // The old dataset still still has no head.
    assert.isNull(await ds.head());

    // The new dataset has |a|.
    const aRef = notNull(await ds2.headRef());
    const aCommit = notNull(await ds2.head());
    assert.strictEqual('a', aCommit.value);
    ds = ds2;

    // |a| <- |b|
    ds = await ds.commit('b', [aRef]);
    assert.strictEqual('b', notNull(await ds.head()).value);

    // |a| <- |b|
    //   \----|c|
    // Should be disallowed.
    let ex;
    try {
      await ds.commit('c', [aRef]);
    } catch (e) {
      ex = e;
    }
    invariant(ex instanceof Error);
    assert.strictEqual('Merge needed', ex.message);
    const bCommit = notNull(await ds.head());
    assert.strictEqual('b', bCommit.value);

    // |a| <- |b| <- |d|
    ds = await ds.commit('d');
    assert.strictEqual('d', notNull(await ds.head()).value);


    // Add a commit to a different datasetId
    ds = new Dataset(db, 'otherDs');
    ds = await ds.commit('a');
    assert.strictEqual('a', notNull(await ds.head('otherDs')).value);

    // Get a fresh database, and verify that both datasets are present
    const newStore = new Database(bs);
    assert.strictEqual('d', notNull(await newStore.head('ds1')).value);
    assert.strictEqual('a', notNull(await newStore.head('otherDs')).value);
    await newStore.close();
  });

  test('id validation', () => {
    const db = new Database(makeTestingBatchStore());

    const invalidDatasetNames = [' ', '', 'a ', ' a', '$', '#', ':', '\n', '💩'];
    for (const s of invalidDatasetNames) {
      assert.throws(() => { new Dataset(db, s); });
    }
  });
});
