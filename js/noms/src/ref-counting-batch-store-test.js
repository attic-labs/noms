// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import {suite, test} from 'mocha';
import {assert} from 'chai';
import MemoryStore from './memory-store.js';
import {BatchStoreAdaptor} from './batch-store.js';
import RefCountingBatchStore from './ref-counting-batch-store.js';

suite('RefCountingBatchStore', () => {
  test('works', async () => {
    let closed = false;

    const delegate = new BatchStoreAdaptor(new MemoryStore());
    const bs = new RefCountingBatchStore(delegate, () => {
      assert.isFalse(closed);
      closed = true;
    });

    bs.addRef();
    assert.isFalse(closed);
    bs.addRef();
    assert.isFalse(closed);
    await bs.close();
    assert.isFalse(closed);
    await bs.close();
    assert.isFalse(closed);
    await bs.close();
    assert.isTrue(closed);

    assert.throws(() => bs.close());
    assert.throws(() => bs.addRef());
    assert.throws(() => bs.flush());
  });
});
