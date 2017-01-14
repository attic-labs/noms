// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import {suite, suiteSetup, suiteTeardown, test} from 'mocha';
import {assert} from 'chai';

import {createStructClass} from './struct.js';
import Database from './database.js';
import {
  blobType,
  boolType,
  makeCycleType,
  makeListType,
  makeMapType,
  makeRefType,
  makeSetType,
  makeStructType,
  makeUnionType,
  numberType,
  stringType,
  typeType,
  valueType,
} from './type.js';
import Blob from './blob.js';
import List from './list.js';
import Map from './map.js';
import NomsSet from './set.js'; // namespace collision with JS Set
import walk from './walk.js';
import type Value from './value.js';
import {smallTestChunks, normalProductionChunks} from './rolling-value-hasher.js';
import {randomBuff} from './blob-test.js';
import {TestDatabase} from './test-util.js';

suite('walk', () => {
  let ds;
  suiteSetup(() => {
    smallTestChunks();
    ds = new TestDatabase();
  });

  suiteTeardown((): Promise<void> => {
    normalProductionChunks();
    return ds.close();
  });

  test('primitives', async () => {
    await Promise.all([true, false, 42, 88.8, 'hello!', ''].map(async v => {
      await callbackHappensOnce(v, ds, false);
    }));
  });

  test('blob', async () => {
    const arr = randomBuff(1 << 10);
    const blob = new Blob(new Uint8Array(arr.buffer));
    assert.equal(blob.length, arr.length);
    assert.isAbove(blob.chunks.length, 1);

    await callbackHappensOnce(blob, ds, false);
  });

  test('blob doesnt load chunks', async () => {
    const arr = randomBuff(1 << 16);
    const blob = new Blob(new Uint8Array(arr.buffer));
    const r = ds.writeValue(blob);
    assert.isTrue(r.height > 1);
    const outBlob = await ds.readValue(r.targetHash);
    await callbackHappensOnce(outBlob, ds, false);
    assert.strictEqual(1, ds.readCount);
  });

  test('type', async () => {
    const assertVisitedOnce = async (root, v) => {
      let count = 0;
      await walk(root, ds, v2 => {
        if (v === v2) {
          count++;
        }
      });
      assert.equal(1, count);
    };

    const t = makeStructType('TestStruct', {
      s: stringType,
      b: boolType,
      n: numberType,
      bl: blobType,
      t: typeType,
      v: valueType,
    });
    await assertVisitedOnce(t, t);
    await assertVisitedOnce(t, boolType);
    await assertVisitedOnce(t, numberType);
    await assertVisitedOnce(t, stringType);
    await assertVisitedOnce(t, blobType);
    await assertVisitedOnce(t, typeType);
    await assertVisitedOnce(t, valueType);

    for (const m of [makeListType, makeSetType, makeRefType]) {
      const t2 = m(boolType);
      await assertVisitedOnce(t2, t2);
      await assertVisitedOnce(t2, boolType);
    }

    const t2 = makeMapType(numberType, stringType);
    await assertVisitedOnce(t2, t2);
    await assertVisitedOnce(t2, numberType);
    await assertVisitedOnce(t2, stringType);

    const t3 = makeUnionType([numberType, stringType, boolType]);
    await assertVisitedOnce(t3, t3);
    await assertVisitedOnce(t3, boolType);
    await assertVisitedOnce(t3, numberType);
    await assertVisitedOnce(t3, stringType);

    const t4 = makeCycleType(11);
    await assertVisitedOnce(t4, t4);
  });

  test('list', async () => {
    const expected = new Set();
    for (let i = 0; i < 1000; i++) {
      expected.add(i);
    }
    const list = new List(Array.from(expected));

    await callbackHappensOnce(list, ds, true);

    const test = async (list, ds, expected) => {
      await walk(list, ds, async v => {
        assert.isOk(expected.delete(v));
      });
      assert.equal(0, expected.size);
    };

    expected.add(list);
    await test(list, ds, new Set(expected));
    expected.delete(list);

    const r = ds.writeValue(list);
    const outList = await ds.readValue(r.targetHash);

    expected.add(outList);
    await test(outList, ds, new Set(expected));
  });

  test('set', async () => {
    const expected = new Set();
    for (let i = 0; i < 1000; i++) {
      expected.add(String(i));
    }
    const set = new NomsSet(Array.from(expected));

    await callbackHappensOnce(set, ds, true);

    const test = async (set, ds, expected) => {
      await walk(set, ds, async v => {
        assert.isOk(expected.delete(v));
      });
      assert.equal(0, expected.size);
    };

    expected.add(set);
    await test(set, ds, new Set(expected));
    expected.delete(set);

    const r = ds.writeValue(set);
    const outSet = await ds.readValue(r.targetHash);

    expected.add(outSet);
    await test(outSet, ds, new Set(expected));
  });

  test('map', async () => {
    const expected = [];
    const entries = [];
    for (let i = 0; i < 1000; i++) {
      expected.push(i);
      expected.push('value' + i);
      entries.push([i, 'value' + i]);
    }
    const map = new Map(entries);

    await callbackHappensOnce(map, ds, true);

    const test = async (map, ds, expected) => {
      await walk(map, ds, async v => {
        const idx = expected.indexOf(v);
        assert.isAbove(idx, -1);
        assert.equal(expected.splice(idx, 1).length, 1);
      });
      assert.equal(0, expected.length);
    };

    expected.push(map);
    await test(map, ds, expected.slice());
    expected.pop();

    const r = ds.writeValue(map);
    const outMap = await ds.readValue(r.targetHash);

    expected.push(outMap);
    await test(outMap, ds, expected);
  });

  test('struct', async () => {
    const t = makeStructType('Thing', {
      foo: stringType,
      list: makeListType(numberType),
      num: numberType,
    });

    const c = createStructClass(t);
    const val = new c({
      foo: 'bar',
      num: 42,
      list: new List([1, 2]),
    });

    await callbackHappensOnce(val, ds, true);

    const expected = new Set([val, val.foo, val.num, val.list, 1, 2]);
    await walk(val, ds, async v => {
      assert.isOk(expected.delete(v));
    });
    assert.equal(0, expected.size);
  });

  test('ref-value', async () => {
    const rv = ds.writeValue(42);
    const expected = new Set([rv, 42]);
    await callbackHappensOnce(rv, ds, true);
    await walk(rv, ds, async v => {
      assert.isOk(expected.delete(v));
    });
    assert.equal(0, expected.size);
  });

  test('cb-should-recurse', async () => {
    const testShouldRecurse = async (cb, expectSkip) => {
      const rv = ds.writeValue(42);
      const expected = new Set([rv, 42]);
      await walk(rv, ds, v => {
        assert.isOk(expected.delete(v));
        return cb();
      });
      assert.equal(expectSkip ? 1 : 0, expected.size);
    };

    // Return void, Promise<void>, false, or Promise<false> -- should recurse.
    await testShouldRecurse(() => { return; }, false); // eslint-disable-line
    await testShouldRecurse(() => Promise.resolve(), false);
    await testShouldRecurse(() => false, false);
    await testShouldRecurse(() => Promise.resolve(false), false);

    // Return true or Promise<true> -- should skip
    await testShouldRecurse(() => true, true);
    await testShouldRecurse(() => Promise.resolve(true), true);
  });
});

async function callbackHappensOnce(v: Value, ds: Database, skip: boolean): Promise<void> {
  // Test that our callback only gets called once.
  let count = 0;
  await walk(v, ds, cv => {
    assert.strictEqual(v, cv);
    count++;
    return skip;
  });
  assert.equal(1, count);
}
