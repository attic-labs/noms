// @flow

import {suite, test} from 'mocha';
import {assert} from 'chai';
import Chunk from './chunk.js';
import MemoryStore from './memory_store.js';
import DataStore from './datastore.js';

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
});
