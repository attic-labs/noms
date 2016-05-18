// @flow

import {assert} from 'chai';
import {suite, test} from 'mocha';

import Database from './database.js';
import MemoryStore from './memory-store.js';
import RefValue from './ref-value.js';
import BatchStore from './batch-store.js';
import {BatchStoreAdaptorDelegate, makeTestingBatchStore} from './batch-store-adaptor.js';
import {newStruct, default as Struct} from './struct.js';
import {flatten, flattenParallel, deriveCollectionHeight} from './test-util.js';
import {invariant} from './assert.js';
import Chunk from './chunk.js';
import {newMapLeafSequence, newMap, default as NomsMap} from './map.js';
import {MetaTuple, newMapMetaSequence} from './meta-sequence.js';
import Ref from './ref.js';
import type {ValueReadWriter} from './value-store.js';
import {compare, equals} from './compare.js';

const testMapSize = 1000;
const mapOfNRef = 'sha1-2bc451349d04c5f90cfe73d1e6eb3ee626db99a1';
const smallRandomMapSize = 50;
const randomMapSize = 500;

class CountingMemoryStore extends MemoryStore {
  getCount: number;

  constructor() {
    super();
    this.getCount = 0;
  }

  get(ref: Ref): Promise<Chunk> {
    this.getCount++;
    return super.get(ref);
  }
}

suite('BuildMap', () => {

  test('unique keys - strings', async () => {
    const kvs = [
      'hello', 'world',
      'foo', 'bar',
      'bar', 'foo',
      'hello', 'foo'];
    const m = await newMap(kvs);
    assert.strictEqual(3, m.size);
    assert.strictEqual('foo', await m.get('hello'));
  });

  test('unique keys - number', async () => {
    const kvs = [
      4, 1,
      0, 2,
      1, 2,
      3, 4,
      1, 5];
    const m = await newMap(kvs);
    assert.strictEqual(4, m.size);
    assert.strictEqual(5, await m.get(1));
  });

  test('LONG: set of n numbers', async () => {
    const kvs = [];
    for (let i = 0; i < testMapSize; i++) {
      kvs.push(i, i + 1);
    }

    const m = await newMap(kvs);
    assert.strictEqual(m.ref.toString(), mapOfNRef);

    // shuffle kvs, and test that the constructor sorts properly
    const pairs = [];
    for (let i = 0; i < kvs.length; i += 2) {
      pairs.push({k: kvs[i], v: kvs[i + 1]});
    }
    pairs.sort(() => Math.random() > .5 ? 1 : -1);
    kvs.length = 0;
    pairs.forEach(kv => kvs.push(kv.k, kv.v));
    const m2 = await newMap(kvs);
    assert.strictEqual(m2.ref.toString(), mapOfNRef);

    const height = deriveCollectionHeight(m);
    assert.isTrue(height > 0);
    assert.strictEqual(height, deriveCollectionHeight(m2));
    assert.strictEqual(height, m.sequence.items[0].ref.height);
  });

  test('LONG: map of ref to ref, set of n numbers', async () => {
    const kvs = [];
    for (let i = 0; i < testMapSize; i++) {
      kvs.push(i, i + 1);
    }

    const kvRefs = kvs.map(n => new RefValue(newStruct('num', {n})));
    const m = await newMap(kvRefs);
    assert.strictEqual(m.ref.toString(), 'sha1-5c9a17f6da0ebfebc1f82f498ac46992fad85250');
    const height = deriveCollectionHeight(m);
    assert.isTrue(height > 0);
    // height + 1 because the leaves are RefValue values (with height 1).
    assert.strictEqual(height + 1, m.sequence.items[0].ref.height);
  });

  test('LONG: set', async () => {
    const kvs = [];
    for (let i = 0; i < testMapSize - 10; i++) {
      kvs.push(i, i + 1);
    }

    let m = await newMap(kvs);
    for (let i = testMapSize - 10; i < testMapSize; i++) {
      m = await m.set(i, i + 1);
      assert.strictEqual(i + 1, m.size);
    }

    assert.strictEqual(m.ref.toString(), mapOfNRef);
  });

  test('LONG: set existing', async () => {
    const kvs = [];
    for (let i = 0; i < testMapSize; i++) {
      kvs.push(i, i + 1);
    }

    let m = await newMap(kvs);
    for (let i = 0; i < testMapSize; i++) {
      m = await m.set(i, i + 1);
      assert.strictEqual(testMapSize, m.size);
    }

    assert.strictEqual(m.ref.toString(), mapOfNRef);
  });

  test('LONG: remove', async () => {
    const kvs = [];
    for (let i = 0; i < testMapSize + 10; i++) {
      kvs.push(i, i + 1);
    }

    let m = await newMap(kvs);
    for (let i = testMapSize; i < testMapSize + 10; i++) {
      m = await m.remove(i);
    }

    assert.strictEqual(m.ref.toString(), mapOfNRef);
    assert.strictEqual(testMapSize, m.size);
  });

  test('LONG: write, read, modify, read', async () => {
    const ds = new Database(makeTestingBatchStore());

    const kvs = [];
    for (let i = 0; i < testMapSize; i++) {
      kvs.push(i, i + 1);
    }

    const m = await newMap(kvs);

    const r = ds.writeValue(m).targetRef;
    const m2 = await ds.readValue(r);
    const outKvs = [];
    await m2.forEach((v, k) => outKvs.push(k, v));
    assert.deepEqual(kvs, outKvs);
    assert.strictEqual(testMapSize, m2.size);

    invariant(m2 instanceof NomsMap);
    const m3 = await m2.remove(testMapSize - 1);
    const outKvs2 = [];
    await m3.forEach((v, k) => outKvs2.push(k, v));
    kvs.splice(testMapSize * 2 - 2, 2);
    assert.deepEqual(kvs, outKvs2);
    assert.strictEqual(testMapSize - 1, m3.size);
  });

  test('LONG: union write, read, modify, read', async () => {
    const ds = new Database(makeTestingBatchStore());

    const keys = [];
    const kvs = [];
    const numbers = [];
    const strings = [];
    const structs = [];
    for (let i = 0; i < testMapSize; i++) {
      let v = i;
      if (i % 3 === 0) {
        v = String(v);
        strings.push(v);
      } else if (v % 3 === 1) {
        v = await newStruct('num', {n: v});
        structs.push(v);
      } else {
        numbers.push(v);
      }
      kvs.push(v, i);
      keys.push(v);
    }

    strings.sort();
    structs.sort(compare);
    const sortedKeys = numbers.concat(strings, structs);

    const m = await newMap(kvs);
    assert.strictEqual(m.ref.toString(), 'sha1-3840c9c93d79663e77a60f13f2877a8f5843da38');
    const height = deriveCollectionHeight(m);
    assert.isTrue(height > 0);
    assert.strictEqual(height, m.sequence.items[0].ref.height);

    // has
    for (let i = 0; i < keys.length; i += 5) {
      assert.isTrue(await m.has(keys[i]));
    }

    const r = ds.writeValue(m).targetRef;
    const m2 = await ds.readValue(r);
    const outVals = [];
    const outKeys = [];
    await m2.forEach((v, k) => {
      outVals.push(v);
      outKeys.push(k);
    });
    assert.equal(testMapSize, m2.size);

    function assertEqualVal(k, v) {
      if (k instanceof Struct) {
        assert.equal(k.n, v);
      } else if (typeof k === 'string') {
        assert.equal(Number(k), v);
      } else {
        assert.equal(k, v);
      }
    }

    for (let i = 0; i < sortedKeys.length; i += 5) {
      const k = sortedKeys[i];
      assert.isTrue(equals(k, outKeys[i]));
      const v = await m2.get(k);
      assertEqualVal(k, v);
    }

    invariant(m2 instanceof NomsMap);
    const m3 = await m2.remove(sortedKeys[testMapSize - 1]);  // removes struct
    const outVals2 = [];
    const outKeys2 = [];
    await m2.forEach((v, k) => {
      outVals2.push(v);
      outKeys2.push(k);
    });
    outVals2.splice(testMapSize - 1, 1);
    outKeys2.splice(testMapSize - 1, 1);
    assert.equal(testMapSize - 1, m3.size);
    for (let i = outKeys2.length - 1; i >= 0; i -= 5) {
      const k = sortedKeys[i];
      assert.isTrue(equals(k, outKeys[i]));
      const v = await m3.get(k);
      assertEqualVal(k, v);
    }
  });

});

suite('MapLeaf', () => {
  test('isEmpty/size', () => {
    const ds = new Database(makeTestingBatchStore());
    const newMap = entries => new NomsMap(newMapLeafSequence(ds, entries));
    let m = newMap([]);
    assert.isTrue(m.isEmpty());
    assert.strictEqual(0, m.size);
    m = newMap([{key: 'a', value: false}, {key: 'k', value: true}]);
    assert.isFalse(m.isEmpty());
    assert.strictEqual(2, m.size);
  });

  test('has', async () => {
    const ds = new Database(makeTestingBatchStore());
    const m = new NomsMap(
        newMapLeafSequence(ds, [{key: 'a', value: false}, {key: 'k', value: true}]));
    assert.isTrue(await m.has('a'));
    assert.isFalse(await m.has('b'));
    assert.isTrue(await m.has('k'));
    assert.isFalse(await m.has('z'));
  });

  test('first/last/get', async () => {
    const ds = new Database(makeTestingBatchStore());
    const m = new NomsMap(
        newMapLeafSequence(ds, [{key: 'a', value: 4}, {key: 'k', value: 8}]));

    assert.deepEqual(['a', 4], await m.first());
    assert.deepEqual(['k', 8], await m.last());

    assert.strictEqual(4, await m.get('a'));
    assert.strictEqual(undefined, await m.get('b'));
    assert.strictEqual(8, await m.get('k'));
    assert.strictEqual(undefined, await m.get('z'));
  });

  test('forEach', async () => {
    const ds = new Database(makeTestingBatchStore());
    const m = new NomsMap(
        newMapLeafSequence(ds, [{key: 'a', value: 4}, {key: 'k', value: 8}]));

    const kv = [];
    await m.forEach((v, k) => { kv.push(k, v); });
    assert.deepEqual(['a', 4, 'k', 8], kv);
  });

  test('iterator', async () => {
    const ds = new Database(makeTestingBatchStore());

    const test = async entries => {
      const m = new NomsMap(newMapLeafSequence(ds, entries));
      assert.deepEqual(entries, await flatten(m.iterator()));
      assert.deepEqual(entries, await flattenParallel(m.iterator(), entries.length));
    };

    await test([]);
    await test([{key: 'a', value: 4}]);
    await test([{key: 'a', value: 4}, {key: 'k', value: 8}]);
  });

  test('LONG: iteratorAt', async () => {
    const ds = new Database(makeTestingBatchStore());
    const build = entries => new NomsMap(newMapLeafSequence(ds, entries));

    assert.deepEqual([], await flatten(build([]).iteratorAt('a')));

    {
      const kv = [{key: 'b', value: 5}];
      assert.deepEqual(kv, await flatten(build(kv).iteratorAt('a')));
      assert.deepEqual(kv, await flatten(build(kv).iteratorAt('b')));
      assert.deepEqual([], await flatten(build(kv).iteratorAt('c')));
    }

    {
      const kv = [{key: 'b', value: 5}, {key: 'd', value: 10}];
      assert.deepEqual(kv, await flatten(build(kv).iteratorAt('a')));
      assert.deepEqual(kv, await flatten(build(kv).iteratorAt('b')));
      assert.deepEqual(kv.slice(1), await flatten(build(kv).iteratorAt('c')));
      assert.deepEqual(kv.slice(1), await flatten(build(kv).iteratorAt('d')));
      assert.deepEqual([], await flatten(build(kv).iteratorAt('e')));
    }
  });

  test('chunks', () => {
    const ds = new Database(makeTestingBatchStore());
    const r1 = ds.writeValue('x');
    const r2 = ds.writeValue(true);
    const r3 = ds.writeValue('b');
    const r4 = ds.writeValue(false);
    const m = new NomsMap(
        newMapLeafSequence(ds, [{key: r1, value: r2}, {key: r3, value: r4}]));
    assert.strictEqual(4, m.chunks.length);
    assert.isTrue(equals(r1, m.chunks[0]));
    assert.isTrue(equals(r2, m.chunks[1]));
    assert.isTrue(equals(r3, m.chunks[2]));
    assert.isTrue(equals(r4, m.chunks[3]));
  });

});

suite('CompoundMap', () => {
  function build(vwr: ValueReadWriter): Array<NomsMap> {
    const l1 = new NomsMap(newMapLeafSequence(vwr, [{key: 'a', value: false},
        {key:'b', value:false}]));
    const r1 = vwr.writeValue(l1);
    const l2 = new NomsMap(newMapLeafSequence(vwr, [{key: 'e', value: true},
        {key:'f', value:true}]));
    const r2 = vwr.writeValue(l2);
    const l3 = new NomsMap(newMapLeafSequence(vwr, [{key: 'h', value: false},
        {key:'i', value:true}]));
    const r3 = vwr.writeValue(l3);
    const l4 = new NomsMap(newMapLeafSequence(vwr, [{key: 'm', value: true},
        {key:'n', value:false}]));
    const r4 = vwr.writeValue(l4);

    const m1 = new NomsMap(newMapMetaSequence(vwr, [new MetaTuple(r1, 'b', 2),
        new MetaTuple(r2, 'f', 2)]));
    const rm1 = vwr.writeValue(m1);
    const m2 = new NomsMap(newMapMetaSequence(vwr, [new MetaTuple(r3, 'i', 2),
        new MetaTuple(r4, 'n', 2)]));
    const rm2 = vwr.writeValue(m2);

    const c = new NomsMap(newMapMetaSequence(vwr, [new MetaTuple(rm1, 'f', 4),
        new MetaTuple(rm2, 'n', 4)]));
    return [c, m1, m2];
  }

  test('isEmpty/size', () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    assert.isFalse(c.isEmpty());
    assert.strictEqual(8, c.size);
  });

  test('get', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);

    assert.strictEqual(false, await c.get('a'));
    assert.strictEqual(false, await c.get('b'));
    assert.strictEqual(undefined, await c.get('c'));
    assert.strictEqual(undefined, await c.get('d'));
    assert.strictEqual(true, await c.get('e'));
    assert.strictEqual(true, await c.get('f'));
    assert.strictEqual(false, await c.get('h'));
    assert.strictEqual(true, await c.get('i'));
    assert.strictEqual(undefined, await c.get('j'));
    assert.strictEqual(undefined, await c.get('k'));
    assert.strictEqual(undefined, await c.get('l'));
    assert.strictEqual(true, await c.get('m'));
    assert.strictEqual(false, await c.get('n'));
    assert.strictEqual(undefined, await c.get('o'));
  });

  test('first/last/has', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c, m1, m2] = build(ds);

    assert.deepEqual(['a', false], await c.first());
    assert.deepEqual(['n', false], await c.last());
    assert.deepEqual(['a', false], await m1.first());
    assert.deepEqual(['f', true], await m1.last());
    assert.deepEqual(['h', false], await m2.first());
    assert.deepEqual(['n', false], await m2.last());

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
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);

    const kv = [];
    await c.forEach((v, k) => { kv.push(k, v); });
    assert.deepEqual(['a', false, 'b', false, 'e', true, 'f', true, 'h', false, 'i', true, 'm',
        true, 'n', false], kv);
  });

  test('iterator', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    const expected = [{key: 'a', value: false}, {key: 'b', value: false}, {key: 'e', value: true},
                      {key: 'f', value: true}, {key: 'h', value: false}, {key: 'i', value: true},
                      {key: 'm', value: true}, {key: 'n', value: false}];
    assert.deepEqual(expected, await flatten(c.iterator()));
    assert.deepEqual(expected, await flattenParallel(c.iterator(), expected.length));
  });

  test('LONG: iteratorAt', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    const entries = [{key: 'a', value: false}, {key: 'b', value: false}, {key: 'e', value: true},
                     {key: 'f', value: true}, {key: 'h', value: false}, {key: 'i', value: true},
                     {key: 'm', value: true}, {key: 'n', value: false}];
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
      const slice = entries.slice(offsets[k]);
      assert.deepEqual(slice, await flatten(c.iteratorAt(k)));
      assert.deepEqual(slice, await flattenParallel(c.iteratorAt(k), slice.length));
    }
  });

  test('iterator return', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    const iter = c.iterator();
    const values = [];
    for (let res = await iter.next(); !res.done; res = await iter.next()) {
      values.push(res.value);
      if (values.length === 5) {
        await iter.return();
      }
    }
    assert.deepEqual([{key: 'a', value: false}, {key: 'b', value: false}, {key: 'e', value: true},
                      {key: 'f', value: true}, {key: 'h', value: false}],
                     values);
  });

  test('iterator return parallel', async () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    const iter = c.iterator();
    const values = await Promise.all([iter.next(), iter.next(), iter.return(), iter.next()]);
    assert.deepEqual([{done: false, value: {key: 'a', value: false}},
                      {done: false, value: {key: 'b', value: false}},
                      {done: true}, {done: true}],
                     values);
  });

  test('chunks', () => {
    const ds = new Database(makeTestingBatchStore());
    const [c] = build(ds);
    assert.strictEqual(2, c.chunks.length);
  });

  async function testRandomDiff(mapSize: number, inM1: number, inM2: number, inBoth: number) {
    invariant(inM1 + inM2 + inBoth <= 1);

    const kv1 = [], kv2 = [], added = [], removed = [], modified = [];

    // Randomly populate kv1/kv2 which will be the contents of m1/m2 respectively, and record which
    // numbers were added/removed.
    for (let i = 0; i < mapSize; i++) {
      const r = Math.random();
      if (r <= inM1) {
        kv1.push(i, i + '');
        removed.push(i);
      } else if (r <= inM1 + inM2) {
        kv2.push(i, i + '');
        added.push(i);
      } else if (r <= inM1 + inM2 + inBoth) {
        kv1.push(i, i + '');
        kv2.push(i, i + '_');
        modified.push(i);
      } else {
        kv1.push(i, i + '');
        kv2.push(i, i + '');
      }
    }

    let [m1, m2] = await Promise.all([newMap(kv1), newMap(kv2)]);

    if (m1.empty || m2.empty || added.length + removed.length + modified.length === 0) {
      return testRandomDiff(mapSize, inM1, inM2, inBoth);
    }

    const ms = new CountingMemoryStore();
    const ds = new Database(new BatchStore(3, new BatchStoreAdaptorDelegate(ms)));
    [m1, m2] = await Promise.all([m1, m2].map(s => ds.readValue(ds.writeValue(s).targetRef)));

    assert.deepEqual([[], [], []], await m1.diff(m1));
    assert.deepEqual([[], [], []], await m2.diff(m2));
    assert.deepEqual([removed, added, modified], await m1.diff(m2));
    assert.deepEqual([added, removed, modified], await m2.diff(m1));
  }

  async function testSmallRandomDiff(inM1: number, inM2: number, inBoth: number) {
    const rounds = randomMapSize / smallRandomMapSize;
    for (let i = 0; i < rounds; i++) {
      await testRandomDiff(smallRandomMapSize, inM1, inM2, inBoth);
    }
  }

  test('LONG: random small map diff 0.1/0.1/0.1', () => testSmallRandomDiff(0.1, 0.1, 0.1));
  test('LONG: random small map diff 0.1/0.5/0.1', () => testSmallRandomDiff(0.1, 0.5, 0.1));
  test('LONG: random small map diff 0.1/0.1/0.5', () => testSmallRandomDiff(0.1, 0.1, 0.5));
  test('LONG: random small map diff 0.1/0.9/0', () => testSmallRandomDiff(0.1, 0.9, 0));

  test('LONG: random map diff 0.0001/0.0001/0.0001',
       () => testRandomDiff(randomMapSize, 0.0001, 0.0001, 0.0001));
  test('LONG: random map diff 0.0001/0.5/0.0001',
       () => testRandomDiff(randomMapSize, 0.0001, 0.5, 0.0001));
  test('LONG: random map diff 0.0001/0.0001/0.5',
       () => testRandomDiff(randomMapSize, 0.0001, 0.0001, 0.5));
  test('LONG: random map diff 0.0001/0.9999/0',
       () => testRandomDiff(randomMapSize, 0.0001, 0.9999, 0));

  test('LONG: random map diff 0.001/0.001/0.001',
       () => testRandomDiff(randomMapSize, 0.001, 0.001, 0.001));
  test('LONG: random map diff 0.001/0.5/0.001',
       () => testRandomDiff(randomMapSize, 0.001, 0.5, 0.001));
  test('LONG: random map diff 0.001/0.001/0.5',
       () => testRandomDiff(randomMapSize, 0.001, 0.001, 0.5));
  test('LONG: random map diff 0.001/0.999/0', () => testRandomDiff(randomMapSize, 0.001, 0.999, 0));

  test('LONG: random map diff 0.01/0.01/0.01',
       () => testRandomDiff(randomMapSize, 0.01, 0.01, 0.01));
  test('LONG: random map diff 0.01/0.5/0.1', () => testRandomDiff(randomMapSize, 0.01, 0.5, 0.1));
  test('LONG: random map diff 0.01/0.1/0.5', () => testRandomDiff(randomMapSize, 0.01, 0.1, 0.5));
  test('LONG: random map diff 0.01/0.99', () => testRandomDiff(randomMapSize, 0.01, 0.99, 0));

  test('LONG: random map diff 0.1/0.1/0.1', () => testRandomDiff(randomMapSize, 0.1, 0.1, 0.1));
  test('LONG: random map diff 0.1/0.5/0.1', () => testRandomDiff(randomMapSize, 0.1, 0.5, 0.1));
  test('LONG: random map diff 0.1/0.1/0.5', () => testRandomDiff(randomMapSize, 0.1, 0.1, 0.5));
  test('LONG: random map diff 0.1/0.9/0', () => testRandomDiff(randomMapSize, 0.1, 0.9, 0));

  test('chunks', () => {
    const ds = new Database(makeTestingBatchStore());
    const m = build(ds)[1];
    const chunks = m.chunks;
    const sequence = m.sequence;
    assert.equal(2, chunks.length);
    assert.deepEqual(sequence.items[0].ref, chunks[0]);
    assert.deepEqual(sequence.items[1].ref, chunks[1]);
  });
});
