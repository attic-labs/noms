// @flow

import {blobType, refOfBlobType} from './type.js';
import {assert} from 'chai';
import Blob, {newBlob, BlobWriter} from './blob.js';
import {suite, test} from 'mocha';
import {
  assertChunkCountAndType,
  assertValueRef,
  assertValueType,
  chunkDiffCount,
  testRoundTripAndValidate,
} from './test-util.js';
import {invariant} from './assert.js';
import {equals} from './compare.js';

// IMPORTANT: These tests and in particular the hash of the values should stay in sync with the
// corresponding tests in go

suite('Blob', () => {

  async function assertReadFull(expect: Uint8Array, blob: Blob): Promise<void> {
    const length = expect.length;
    const reader = blob.getReader();
    let i = 0;

    while (i < length) {
      const next = await reader.read();
      assert.isFalse(next.done);
      const arr = next.value;
      invariant(arr);
      for (let j = 0; j < arr.length; j++) {
        assert.strictEqual(expect[i], arr[j]);
        i++;
      }
    }
  }

  async function testPrependChunkDiff(buff: Uint8Array, blob: Blob, expectCount: number):
      Promise<void> {
    const nb = new Uint8Array(buff.length + 1);
    for (let i = 0; i < buff.length; i++) {
      nb[i + 1] = buff[i];
    }

    const v2 = await newBlob(nb);
    assert.strictEqual(expectCount, chunkDiffCount(blob, v2));
  }

  async function testAppendChunkDiff(buff: Uint8Array, blob: Blob, expectCount: number):
      Promise<void> {
    const nb = new Uint8Array(buff.length + 1);
    for (let i = 0; i < buff.length; i++) {
      nb[i] = buff[i];
    }

    const v2 = await newBlob(nb);
    assert.strictEqual(expectCount, chunkDiffCount(blob, v2));
  }

  function randomBuff(len: number): Uint8Array {
    const r = new CountingByteReader();
    const a = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
      a[i] = r.nextUint8();
    }
    return a;
  }

  async function blobTestSuite(size: number, expectRefStr: string, expectChunkCount: number,
                               expectPrependChunkDiff: number,
                               expectAppendChunkDiff: number) {
    const length = 1 << size;
    const buff = randomBuff(length);
    const blob = await newBlob(buff);

    assertValueRef(expectRefStr, blob);
    assertValueType(blobType, blob);
    assert.strictEqual(length, blob.length);
    assertChunkCountAndType(expectChunkCount, refOfBlobType, blob);

    await testRoundTripAndValidate(blob, async(b2) => {
      await assertReadFull(buff, b2);
    });

    // TODO: Random Read

    await testPrependChunkDiff(buff, blob, expectPrependChunkDiff);
    await testAppendChunkDiff(buff, blob, expectAppendChunkDiff);
  }

  class CountingByteReader {
    _z: number;
    _value: number;
    _count: number;

    constructor(seed: number = 0) {
      this._z = seed;
      this._value = seed;
      this._count = 4;
    }

    nextUint8(): number {
      // Increment number
      if (this._count === 0) {
        this._z++;
        this._value = this._z;
        this._count = 4;
      }

      // Unshift a uint8 from our current number
      const retval = this._value & 0xff;
      this._value = this._value >>> 8;
      this._count--;

      return retval;
    }
  }

  test('Blob 1K', async () => {
    await blobTestSuite(10, 'sha1-f9fc78f387d90a334b85270a46484ebb86f32a3f', 3, 2, 2);
  });

  test('LONG: Blob 4K', async () => {
    await blobTestSuite(12, 'sha1-060e57a95676be6078a2958f4586a8b3d3e6723d', 9, 2, 2);
  });

  test('LONG: Blob 16K', async () => {
    await blobTestSuite(14, 'sha1-73d3ce5da681cf651509194ce63a055c51790385', 33, 2, 2);
  });

  test('LONG: Blob 64K', async () => {
    await blobTestSuite(16, 'sha1-130b214a2e7edbdd5583dc7920b52838cad49471', 4, 2, 2);
  });

  test('LONG: Blob 256K', async () => {
    await blobTestSuite(18, 'sha1-42d53b4f225322f70d725d53f8bc631d4549b6e4', 13, 2, 2);
  });

  test('BlobWriter', async () => {
    const a = randomBuff(15);
    const b1 = await newBlob(a);
    const w = new BlobWriter();
    w.write(new Uint8Array(a.buffer, 0, 5));
    w.write(new Uint8Array(a.buffer, 5, 5));
    w.write(new Uint8Array(a.buffer, 10, 5));
    await w.close();
    const b2 = w.blob;
    const b3 = w.blob;
    assert.strictEqual(b2, b3);
    assert.isTrue(equals(b1, b2));
  });

  test('BlobWriter blob throws', async () => {
    const a = randomBuff(15);
    const w = new BlobWriter();
    w.write(a);
    w.close();  // No await, so not closed
    let ex;
    try {
      w.blob;
    } catch (e) {
      ex = e;
    }
    assert.instanceOf(ex, TypeError);

    try {
      await w.close();  // Cannot close twice.
    } catch (e) {
      ex = e;
    }
    assert.instanceOf(ex, TypeError);
  });

  test('BlobWriter close throws', async () => {
    const a = randomBuff(15);
    const w = new BlobWriter();
    w.write(a);
    w.close();  // No await, so closing

    let ex;
    try {
      await w.close();  // Cannot close twice.
    } catch (e) {
      ex = e;
    }
    assert.instanceOf(ex, TypeError);
  });

  test('BlobWriter write throws', async () => {
    const a = randomBuff(15);
    const w = new BlobWriter();
    w.write(a);
    await w.close();  // No await, so closing

    let ex;
    try {
      w.write(a);
    } catch (e) {
      ex = e;
    }
    assert.instanceOf(ex, TypeError);
  });
});
