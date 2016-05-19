// @flow

import {suite, test} from 'mocha';
import {assert} from 'chai';
import {notNull} from './assert.js';
import OrderedPutCache from './put-cache.js';
import Chunk from './chunk.js';

suite('OrderedPutCache', () => {
  test('insert', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def')];
    const cache = new OrderedPutCache();
    assert.isTrue(cache.insert(canned[0], 1));
    assert.isTrue(cache.insert(canned[1], 1));
    await cache.destroy();
  });

  test('repeated insert returns false', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def')];
    const cache = new OrderedPutCache();
    assert.isTrue(cache.insert(canned[0], 1));
    assert.isTrue(cache.insert(canned[1], 1));
    assert.isFalse(cache.insert(canned[0], 1));
    await cache.destroy();
  });

  test('get', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def')];
    const cache = new OrderedPutCache();
    assert.isTrue(cache.insert(canned[0], 1));

    let p = cache.get(canned[1].ref.toString());
    assert.isNull(p);

    assert.isTrue(cache.insert(canned[1], 1));
    p = cache.get(canned[1].ref.toString());
    assert.isNotNull(p);
    const chunk = await notNull(p);
    assert.isTrue(canned[1].ref.equals(chunk.ref));

    await cache.destroy();
  });

  test('dropUntil', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def')];
    const cache = new OrderedPutCache();
    for (const chunk of canned) {
      assert.isTrue(cache.insert(chunk, 1));
    }
    const firstGen = cache.gen;
    cache.gen++;

    const extraChunk = Chunk.fromString('ghi');
    assert.isTrue(cache.insert(extraChunk, 1));

    await cache.dropGeneration(firstGen);

    let p = cache.get(extraChunk.ref.toString());
    assert.isNotNull(p);
    const chunk = await notNull(p);
    assert.isTrue(extraChunk.ref.equals(chunk.ref));

    p = cache.get(canned[0].ref.toString());
    assert.isNull(p);
    p = cache.get(canned[1].ref.toString());
    assert.isNull(p);

    await cache.destroy();
  });

  test('extractChunks', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def'), Chunk.fromString('ghi')];
    const cache = new OrderedPutCache();
    // Insert chunks with different refHeights, so we can be sure they come out in the right order.
    assert.isTrue(cache.insert(canned[2], 1));
    assert.isTrue(cache.insert(canned[0], 1));
    assert.isTrue(cache.insert(canned[1], 2));

    const chunkStream = await cache.extractChunks(cache.gen);
    const chunks = [];
    await chunkStream(chunk => { chunks.push(chunk); });

    assert.isTrue(canned[2].ref.equals(chunks[0].ref));
    assert.isTrue(canned[0].ref.equals(chunks[1].ref));
    assert.isTrue(canned[1].ref.equals(chunks[2].ref));

    await cache.destroy();
  });

  test('extractChunks one generation only', async () => {
    const canned = [Chunk.fromString('abc'), Chunk.fromString('def'), Chunk.fromString('ghi')];
    const cache = new OrderedPutCache();
    for (const chunk of canned) {
      assert.isTrue(cache.insert(chunk, 1));
    }
    const firstGen = cache.gen;
    cache.gen++;

    const extraChunk = Chunk.fromString('123');
    assert.isTrue(cache.insert(extraChunk, 1));

    const chunkStream = await cache.extractChunks(firstGen);
    const chunks = [];
    await chunkStream(chunk => { chunks.push(chunk); });

    assert.equal(canned.length, chunks.length);
    for (let i = 0; i < canned.length; i++) {
      assert.isTrue(canned[i].ref.equals(chunks[i].ref));
    }

    await cache.destroy();
  });
});
