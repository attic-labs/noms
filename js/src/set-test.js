// @flow

// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import {assert} from 'chai';
import {suite, setup, teardown, test} from 'mocha';


import Ref from './ref.js';
import Set, {SetLeafSequence} from './set.js';
import type {ValueReadWriter} from './value-store.js';
import {MetaTuple, newSetMetaSequence} from './meta-sequence.js';
import {OrderedSequence} from './ordered-sequence.js';
import {compare, equals} from './compare.js';
import {
  flatten,
  flattenParallel,
  intSequence,
  deriveCollectionHeight,
  TestDatabase,
} from './test-util.js';
import {invariant, notNull} from './assert.js';
import {newStruct} from './struct.js';
import {OrderedMetaSequence} from './meta-sequence.js';
import {
  makeSetType,
  makeUnionType,
  numberType,
  stringType,
} from './type.js';

const testSetSize = 5000;
const setOfNRef = 'sha1-1fc97c4e369770b643e013569c68687765601514';
const smallRandomSetSize = 200;
const randomSetSize = 2000;

async function validateSet(s: Set, values: number[]): Promise<void> {
  assert.isTrue(equals(new Set(values), s));

  const out = [];
  await s.forEach(v => { out.push(v); });
  assert.deepEqual(values, out);
}

suite('BuildSet', () => {
  test('unique keys - strings', async () => {
    const strs = ['hello', 'world', 'hello'];
    const s = new Set(strs);
    assert.strictEqual(2, s.size);
    assert.isTrue(await s.has('hello'));
    assert.isTrue(await s.has('world'));
    assert.isFalse(await s.has('foo'));
  });

  test('unique keys - number', async () => {
    const nums = [4, 1, 0, 0, 1, 3];
    const s = new Set(nums);
    assert.strictEqual(4, s.size);
    assert.isTrue(await s.has(4));
    assert.isTrue(await s.has(1));
    assert.isTrue(await s.has(0));
    assert.isTrue(await s.has(3));
    assert.isFalse(await s.has(2));
  });

  test('LONG: set of n numbers', async () => {
    const nums = intSequence(testSetSize);
    const s = new Set(nums);
    assert.strictEqual(setOfNRef, s.hash.toString());

    // shuffle kvs, and test that the constructor sorts properly
    nums.sort(() => Math.random() > .5 ? 1 : -1);
    const s2 = new Set(nums);
    assert.strictEqual(setOfNRef, s2.hash.toString());
  });

  test('LONG: set of struct, set of n numbers', async () => {
    const nums = intSequence(testSetSize);
    const structs = nums.map(n => newStruct('num', {n}));
    const s = new Set(structs);
    assert.strictEqual('sha1-4e6c7eb66b2cc252611a38cb06eed751d2bdf3c3', s.hash.toString());
    const height = deriveCollectionHeight(s);
    assert.isTrue(height > 0);
    assert.strictEqual(height, s.sequence.items[0].ref.height);

    // has
    for (let i = 0; i < structs.length; i++) {
      assert.isTrue(await s.has(structs[i]));
    }
  });

  test('LONG: set of ref, set of n numbers', async () => {
    const nums = intSequence(testSetSize);
    const refs = nums.map(n => new Ref(newStruct('num', {n})));
    const s = new Set(refs);
    assert.strictEqual('sha1-c8396816c8fb961939cc7c9dcf8011efe9040103', s.hash.toString());
    const height = deriveCollectionHeight(s);
    assert.isTrue(height > 0);
    // height + 1 because the leaves are Ref values (with height 1).
    assert.strictEqual(height + 1, s.sequence.items[0].ref.height);
  });

  test('LONG: add', async () => {
    const nums = intSequence(testSetSize);
    const build = nums.slice(0, testSetSize - 10);
    const insert = nums.slice(testSetSize - 10);
    let s = new Set(build);
    for (let i = 0; i < insert.length; i++) {
      s = await s.add(insert[i]);
      assert.strictEqual(build.length + i + 1, s.size);
    }

    await validateSet(s, nums);
  });

  async function validateInsertion(values: number[]): Promise<void> {
    let s = new Set();
    for (let i = 0; i < values.length; i++) {
      s = await s.add(values[i]);
      await validateSet(s, values.slice(0, i + 1));
    }
  }

  test('LONG: validate - add ascending', async () => {
    await validateInsertion(intSequence(300));
  });

  test('LONG: delete', async () => {
    const nums = intSequence(testSetSize + 10);
    let s = new Set(nums);
    let count = 10;
    while (count-- > 0) {
      s = await s.delete(testSetSize + count);
      assert.strictEqual(testSetSize + count, s.size);
    }

    assert.strictEqual(s.hash.toString(), setOfNRef);
  });

  test('LONG: write, read, modify, read', async () => {
    const db = new TestDatabase();

    const nums = intSequence(testSetSize);
    const s = new Set(nums);
    const r = db.writeValue(s).targetHash;
    const s2 = await db.readValue(r);
    const outNums = [];
    await s2.forEach(k => outNums.push(k));
    assert.deepEqual(nums, outNums);
    assert.strictEqual(testSetSize, s2.size);

    invariant(s2 instanceof Set);
    const s3 = await s2.delete(testSetSize - 1);
    const outNums2 = [];
    await s3.forEach(k => outNums2.push(k));
    nums.splice(testSetSize - 1, 1);
    assert.deepEqual(nums, outNums2);
    assert.strictEqual(testSetSize - 1, s3.size);
    await db.close();
  });

  test('LONG: union write, read, modify, read', async () => {
    const db = new TestDatabase();

    const tmp = intSequence(testSetSize);
    const numbers = [];
    const strings = [];
    const structs = [];
    const vals = [];
    for (let i = 0; i < tmp.length; i++) {
      let v = tmp[i];
      if (i % 3 === 0) {
        v = String(v);
        strings.push(v);
      } else if (v % 3 === 1) {
        v = await newStruct('num', {n: v});
        structs.push(v);
      } else {
        numbers.push(v);
      }
      vals.push(v);
    }
    strings.sort();
    structs.sort(compare);
    vals.sort(compare);

    const s = new Set(vals);
    assert.strictEqual('sha1-4af5578e97bbe7ae1326087f004e828716ac7b85', s.hash.toString());
    const height = deriveCollectionHeight(s);
    assert.isTrue(height > 0);
    assert.strictEqual(height, s.sequence.items[0].ref.height);

    // has
    for (let i = 0; i < vals.length; i += 5) {
      assert.isTrue(await s.has(vals[i]));
    }

    const r = db.writeValue(s).targetHash;
    const s2 = await db.readValue(r);
    const outVals = [];
    await s2.forEach(k => outVals.push(k));
    assert.equal(testSetSize, s2.size);
    for (let i = 0; i < vals.length; i += 5) {
      assert.isTrue(equals(vals[i], outVals[i]));
    }

    invariant(s2 instanceof Set);
    const s3 = await s2.delete(vals[testSetSize - 1]);  // removes struct
    const outVals2 = [];
    await s3.forEach(k => outVals2.push(k));
    vals.splice(testSetSize - 1, 1);
    assert.equal(testSetSize - 1, s3.size);
    for (let i = vals.length - 1; i >= 0; i -= 5) {
      assert.isTrue(equals(vals[i], outVals2[i]));
    }
    await db.close();
  });
});

suite('SetLeaf', () => {
  let db;

  setup(() => {
    db = new TestDatabase();
  });

  teardown((): Promise<void> => db.close());

  test('isEmpty/size', () => {
    let s = new Set();
    assert.isTrue(s.isEmpty());
    assert.strictEqual(0, s.size);
    s = new Set(['a', 'k']);
    assert.isFalse(s.isEmpty());
    assert.strictEqual(2, s.size);
  });

  test('first/last/has', async () => {
    const s = new Set(['a', 'k']);

    assert.strictEqual('a', await s.first());
    assert.strictEqual('k', await s.last());

    assert.isTrue(await s.has('a'));
    assert.isFalse(await s.has('b'));
    assert.isTrue(await s.has('k'));
    assert.isFalse(await s.has('z'));
  });

  test('forEach', async () => {
    const m = new Set(['a', 'b']);

    const values = [];
    await m.forEach((k) => { values.push(k); });
    assert.deepEqual(['a', 'b'], values);
  });

  test('forEachAsyncCB', async () => {
    const m = new Set(['a', 'b']);

    let resolver = null;
    const p = new Promise(resolve => resolver = resolve);

    const values = [];
    const foreachPromise = m.forEach(k => p.then(() => {
      values.push(k);
    }));

    notNull(resolver)();
    return foreachPromise.then(() => assert.deepEqual(['a', 'b'], values));
  });

  test('iterator', async () => {
    const test = async items => {
      const m = new Set(items);
      assert.deepEqual(items, await flatten(m.iterator()));
      assert.deepEqual(items, await flattenParallel(m.iterator(), items.length));
    };

    await test([]);
    await test(['a']);
    await test(['a', 'b']);
  });

  test('LONG: iteratorAt', async () => {
    const build = items => new Set(items);

    assert.deepEqual([], await flatten(build([]).iteratorAt('a')));

    assert.deepEqual(['b'], await flatten(build(['b']).iteratorAt('a')));
    assert.deepEqual(['b'], await flatten(build(['b']).iteratorAt('b')));
    assert.deepEqual([], await flatten(build(['b']).iteratorAt('c')));

    assert.deepEqual(['b', 'd'], await flatten(build(['b', 'd']).iteratorAt('a')));
    assert.deepEqual(['b', 'd'], await flatten(build(['b', 'd']).iteratorAt('b')));
    assert.deepEqual(['d'], await flatten(build(['b', 'd']).iteratorAt('c')));
    assert.deepEqual(['d'], await flatten(build(['b', 'd']).iteratorAt('d')));
    assert.deepEqual([], await flatten(build(['b', 'd']).iteratorAt('e')));
  });

  test('chunks', () => {
    const refs = ['x', 'a', 'b'].map(c => db.writeValue(c));
    refs.sort(compare);
    const s = new Set(['z', ...refs]);
    assert.deepEqual(refs, s.chunks);
  });
});

suite('CompoundSet', () => {
  let db;

  setup(() => {
    db = new TestDatabase();
  });

  teardown((): Promise<void> => db.close());

  function build(vwr: ValueReadWriter, values: Array<string>): Set {
    assert.isTrue(values.length > 1 && Math.log2(values.length) % 1 === 0);

    let tuples = [];
    for (let i = 0; i < values.length; i += 2) {
      const s = new Set([values[i], values[i + 1]]);
      const r = vwr.writeValue(s);
      tuples.push(new MetaTuple(r, values[i + 1], 2, null));
    }

    let last: ?Set = null;
    while (tuples.length > 1) {
      const next = [];
      for (let i = 0; i < tuples.length; i += 2) {
        last = Set.fromSequence(newSetMetaSequence(vwr, [tuples[i], tuples[i + 1]]));
        const r = vwr.writeValue(last);
        next.push(new MetaTuple(r, tuples[i + 1].value,
                                tuples[i].numLeaves + tuples[i + 1].numLeaves, null));
      }

      tuples = next;
    }

    return notNull(last);
  }

  test('isEmpty/size', () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    assert.isFalse(c.isEmpty());
    assert.strictEqual(8, c.size);
  });

  test('first/last/has', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    assert.strictEqual('a', await c.first());
    assert.strictEqual('n', await c.last());
    assert.isTrue(await c.has('a'));
    assert.isTrue(await c.has('b'));
    assert.isFalse(await c.has('c'));
    assert.isFalse(await c.has('d'));
    assert.isTrue(await c.has('e'));
    assert.isTrue(await c.has('f'));
    assert.isTrue(await c.has('h'));
    assert.isTrue(await c.has('i'));
    assert.isFalse(await c.has('j'));
    assert.isFalse(await c.has('k'));
    assert.isFalse(await c.has('l'));
    assert.isTrue(await c.has('m'));
    assert.isTrue(await c.has('n'));
    assert.isFalse(await c.has('o'));
  });

  test('forEach', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    const values = [];
    await c.forEach((k) => { values.push(k); });
    assert.deepEqual(['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n'], values);
  });

  test('forEachAsyncCB', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);

    let resolver = null;
    const p = new Promise(resolve => resolver = resolve);

    const values = [];
    const foreachPromise = c.forEach(k => p.then(() => {
      values.push(k);
    }));

    notNull(resolver)();
    return foreachPromise.then(() => {
      assert.deepEqual(['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n'], values);
    });
  });

  test('iterator', async () => {
    const values = ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n'];
    const c = build(db, values);
    assert.deepEqual(values, await flatten(c.iterator()));
    assert.deepEqual(values, await flattenParallel(c.iterator(), values.length));
  });

  test('LONG: iteratorAt', async () => {
    const values = ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n'];
    const c = build(db, values);
    const offsets = {
      _: 0, a: 0,
      b: 1,
      c: 2, d: 2, e: 2,
      f: 3,
      g: 4, h: 4,
      i: 5,
      j: 6, k: 6, l: 6, m: 6,
      n: 7,
      o: 8,
    };
    for (const k in offsets) {
      const slice = values.slice(offsets[k]);
      assert.deepEqual(slice, await flatten(c.iteratorAt(k)));
      assert.deepEqual(slice, await flattenParallel(c.iteratorAt(k), slice.length));
    }
  });

  test('iterator return', async () => {
    const values = ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n'];
    const c = build(db, values);
    const iter = c.iterator();
    const values2 = [];
    for (let res = await iter.next(); !res.done; res = await iter.next()) {
      values2.push(res.value);
      if (values2.length === 5) {
        await iter.return();
      }
    }
    assert.deepEqual(values.slice(0, 5), values2);
  });

  test('iterator return parallel', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    const iter = c.iterator();
    const values = await Promise.all([iter.next(), iter.next(), iter.return(), iter.next()]);
    assert.deepEqual(
        [{done: false, value: 'a'}, {done: false, value: 'b'}, {done: true}, {done: true}],
        values);
  });

  test('chunks', () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    assert.strictEqual(2, c.chunks.length);
  });

  test('map', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    const values = await c.map((k) => k + '*');
    assert.deepEqual(['a*', 'b*', 'e*', 'f*', 'h*', 'i*', 'm*', 'n*'], values);
  });

  test('map async', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);
    const values = await c.map((k) => Promise.resolve(k + '*'));
    assert.deepEqual(['a*', 'b*', 'e*', 'f*', 'h*', 'i*', 'm*', 'n*'], values);
  });

  async function asyncAssertThrows(f: () => any):Promise<boolean> {
    let error: any = null;
    try {
      await f();
    } catch (er) {
      error = er;
    }

    return error !== null;
  }

  test('advanceTo', async () => {
    const c = build(db, ['a', 'b', 'e', 'f', 'h', 'i', 'm', 'n']);

    invariant(c.sequence instanceof OrderedSequence);
    let cursor = await c.sequence.newCursorAt(null);
    assert.ok(cursor);
    assert.strictEqual('a', cursor.getCurrent());

    assert.isTrue(await cursor.advanceTo('h'));
    assert.strictEqual('h', cursor.getCurrent());

    assert.isTrue(await cursor.advanceTo('k'));
    assert.strictEqual('m', cursor.getCurrent());

    assert.isFalse(await cursor.advanceTo('z')); // not found
    assert.isFalse(cursor.valid);

    invariant(c.sequence instanceof OrderedSequence);
    cursor = await c.sequence.newCursorAt('x'); // not found
    assert.isFalse(cursor.valid);

    invariant(c.sequence instanceof OrderedSequence);
    cursor = await c.sequence.newCursorAt('e');
    assert.ok(cursor);
    assert.strictEqual('e', cursor.getCurrent());

    assert.isTrue(await cursor.advanceTo('m'));
    assert.strictEqual('m', cursor.getCurrent());

    assert.isTrue(await cursor.advanceTo('n'));
    assert.strictEqual('n', cursor.getCurrent());

    assert.isFalse(await cursor.advanceTo('s'));
    assert.isFalse(cursor.valid);

    asyncAssertThrows(async () => {
      await notNull(cursor).advanceTo('x');
    });
  });

  test('iterator at 0', async () => {
    const test = async (expected, items) => {
      const set = new Set(items);
      const iter = set.iteratorAt(0);
      assert.deepEqual(expected, await flatten(iter));
    };

    await test([1, 2], [1, 2]);
    await test([0, 1, 2], [0, 1, 2]);
    await test([1, 2], [-2, -1, 1, 2]);
    await test([0, 1, 2], [-2, -1, 0, 1, 2]);
  });

  test('set of bool', async () => {
    const set = new Set([true]);
    assert.isTrue(await set.has(true));
    assert.isFalse(await set.has(false));
  });

  test('LONG: canned set diff', async () => {
    let s1 = new Set(intSequence(testSetSize));
    s1 = await db.readValue(db.writeValue(s1).targetHash);

    {
      // Insert/delete at start.
      const s2 = await s1.add(-1);
      assert.deepEqual([[-1], []], await s2.diff(s1));
      assert.deepEqual([[], [-1]], await s1.diff(s2));
    }
    {
      // Insert/delete at end.
      const s2 = await s1.add(testSetSize);
      assert.deepEqual([[testSetSize], []], await s2.diff(s1));
      assert.deepEqual([[], [testSetSize]], await s1.diff(s2));
    }
    {
      // Insert/delete in middle.
      const s2 = await s1.delete(testSetSize / 2);
      assert.deepEqual([[], [testSetSize / 2]], await s2.diff(s1));
      assert.deepEqual([[testSetSize / 2], []], await s1.diff(s2));
    }
  });

  async function testRandomDiff(setSize: number, inS1: number, inS2: number): Promise<void> {
    invariant(inS1 + inS2 <= 1);

    const nums1 = [], nums2 = [], added = [], removed = [];

    // Randomly populate nums1/nums2 which will be the contents of s1/s2 respectively, and record
    // which numbers were added/removed.
    for (let i = 0; i < setSize; i++) {
      const r = Math.random();
      if (r <= inS1) {
        nums1.push(i);
        removed.push(i);
      } else if (r <= inS1 + inS2) {
        nums2.push(i);
        added.push(i);
      } else {
        nums1.push(i);
        nums2.push(i);
      }
    }

    let [s1, s2] = await Promise.all([new Set(nums1), new Set(nums2)]);

    if (s1.empty || s2.empty || added.length + removed.length === 0) {
      return testRandomDiff(setSize, inS1, inS2);
    }

    const db = new TestDatabase();
    [s1, s2] = await Promise.all([s1, s2].map(s => db.readValue(db.writeValue(s).targetHash)));

    assert.deepEqual([[], []], await s1.diff(s1));
    assert.deepEqual([[], []], await s2.diff(s2));
    assert.deepEqual([removed, added], await s1.diff(s2));
    assert.deepEqual([added, removed], await s2.diff(s1));
    await db.close();
  }

  function testSmallRandomDiff(inS1: number, inS2: number): Promise<void> {
    const rounds = randomSetSize / smallRandomSetSize;
    const tests = [];
    for (let i = 0; i < rounds; i++) {
      tests.push(testRandomDiff(smallRandomSetSize, inS1, inS2));
    }
    return Promise.all(tests).then(() => undefined);
  }

  test('LONG: random small set diff 0.1/0.1', () => testSmallRandomDiff(0.1, 0.1));
  test('LONG: random small set diff 0.1/0.5', () => testSmallRandomDiff(0.1, 0.5));
  test('LONG: random small set diff 0.1/0.9', () => testSmallRandomDiff(0.1, 0.9));

  test('LONG: random set diff 0.0001/0.0001', () => testRandomDiff(randomSetSize, 0.0001, 0.0001));
  test('LONG: random set diff 0.0001/0.5', () => testRandomDiff(randomSetSize, 0.0001, 0.5));
  test('LONG: random set diff 0.0001/0.9999', () => testRandomDiff(randomSetSize, 0.0001, 0.9900));

  test('LONG: random set diff 0.001/0.001', () => testRandomDiff(randomSetSize, 0.001, 0.001));
  test('LONG: random set diff 0.001/0.5', () => testRandomDiff(randomSetSize, 0.001, 0.5));
  test('LONG: random set diff 0.001/0.999', () => testRandomDiff(randomSetSize, 0.001, 0.999));

  test('LONG: random set diff 0.01/0.01', () => testRandomDiff(randomSetSize, 0.01, 0.01));
  test('LONG: random set diff 0.01/0.5', () => testRandomDiff(randomSetSize, 0.01, 0.5));
  test('LONG: random set diff 0.01/0.99', () => testRandomDiff(randomSetSize, 0.01, 0.99));

  test('LONG: random set diff 0.1/0.1', () => testRandomDiff(randomSetSize, 0.1, 0.1));
  test('LONG: random set diff 0.1/0.5', () => testRandomDiff(randomSetSize, 0.1, 0.5));
  test('LONG: random set diff 0.1/0.9', () => testRandomDiff(randomSetSize, 0.1, 0.9));

  test('chunks', () => {
    const s = build(db, ['a', 'b', 'c', 'd']);
    const chunks = s.chunks;
    const sequence = s.sequence;
    assert.equal(2, chunks.length);
    assert.deepEqual(sequence.items[0].ref, chunks[0]);
    assert.deepEqual(sequence.items[1].ref, chunks[1]);
  });

  test('Type after mutations', async () => {
    async function t(n, c) {
      const values: any = intSequence(n);

      let s = new Set(values);
      assert.equal(s.size, n);
      assert.instanceOf(s.sequence, c);
      assert.isTrue(equals(s.type, makeSetType(numberType)));

      s = await s.add('a');
      assert.equal(s.size, n + 1);
      assert.instanceOf(s.sequence, c);
      assert.isTrue(equals(s.type, makeSetType(makeUnionType([numberType, stringType]))));

      s = await s.delete('a');
      assert.equal(s.size, n);
      assert.instanceOf(s.sequence, c);
      assert.isTrue(equals(s.type, makeSetType(numberType)));
    }

    await t(10, SetLeafSequence);
    await t(100, OrderedMetaSequence);
  });
});
