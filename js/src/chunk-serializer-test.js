// @flow

import {suite, test} from 'mocha';
import {assert} from 'chai';
import Chunk from './chunk.js';
import Ref from './ref.js';
import {deserialize, serialize} from './chunk-serializer.js';
import type {ChunkStream} from './chunk-serializer.js';

suite('ChunkSerializer', () => {

  function assertHints(expect: Array<Ref>, actual: Array<Ref>) {
    assert.strictEqual(actual.length, expect.length);
    for (let i = 0; i < expect.length; i++) {
      assert.isTrue(expect[i].equals(actual[i]));
    }
  }

  function assertChunks(expect: Array<Chunk>, actual: Array<Chunk>) {
    assert.strictEqual(actual.length, expect.length);
    for (let i = 0; i < expect.length; i++) {
      assert.isTrue(expect[i].ref.equals(actual[i].ref));
    }
  }

  test('simple', async () => {
    const expHints = [];
    const expChunks = [Chunk.fromString('abc'), Chunk.fromString('def'), Chunk.fromString('ghi'),
                       Chunk.fromString('wacka wack wack')];

    const pSerialized = serialize(new Set(expHints), createChunkStream(expChunks));
    const {hints, chunks} = deserialize(await pSerialized);

    assertHints(expHints, hints);
    assertChunks(expChunks, chunks);
  });

  test('leading & trailing empty', async () => {
    const expHints = [];
    const expChunks = [Chunk.fromString(''), Chunk.fromString('A'), Chunk.fromString('')];

    const pSerialized = serialize(new Set(expHints), createChunkStream(expChunks));
    const {hints, chunks} = deserialize(await pSerialized);

    assertHints(expHints, hints);
    assertChunks(expChunks, chunks);
  });

  test('all empty', async () => {
    const expHints = [];
    const expChunks = [];


    const pSerialized = serialize(new Set(expHints), createChunkStream(expChunks));
    const {hints, chunks} = deserialize(await pSerialized);

    assertHints(expHints, hints);
    assertChunks(expChunks, chunks);
  });

  test('with hints', async () => {
    const expHints = [
      Chunk.fromString('123').ref,
      Chunk.fromString('456').ref,
      Chunk.fromString('789').ref,
      Chunk.fromString('wacka wack wack').ref,
    ];
    const expChunks = [Chunk.fromString('abc'), Chunk.fromString('def'), Chunk.fromString('ghi')];

    const pSerialized = serialize(new Set(expHints), createChunkStream(expChunks));
    const {hints, chunks} = deserialize(await pSerialized);

    assertHints(expHints, hints);
    assertChunks(expChunks, chunks);
  });

  test('large chunk', async () => {
    const expHints = [];
    const expChunks = [
      new Chunk(new Uint8Array(1024)),
      Chunk.fromString('abc'),
      Chunk.fromString('def'),
      new Chunk(new Uint8Array(2048))];

    const pSerialized = serialize(new Set(expHints), createChunkStream(expChunks));
    const {hints, chunks} = deserialize(await pSerialized);

    assertHints(expHints, hints);
    assertChunks(expChunks, chunks);
  });
});

function createChunkStream(chunks: Array<Chunk>): ChunkStream {
  return function(cb: (chunk: Chunk) => void): Promise<void> {
    return new Promise(fulfill => {
      for (const chunk of chunks) {
        cb(chunk);
      }
      fulfill();
    });
  };
}
