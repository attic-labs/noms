// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import * as Bytes from './bytes.js';

export const maxVarintLength = 10;

const mathPowTwoThirtyTwo = Math.pow(2, 32);

function toUint32(n: number): number {
  return n >>> 0;
}

/**
 * Encodes `val` as signed varint and writes that into `buf` at `offset`. This returns the number
 * of bytes written.
 */
export function encode(val: number, buf: Uint8Array, offset: number): number {
  const val2 = val >= 0 ? val : -val;
  let hi = toUint32(val2 / mathPowTwoThirtyTwo);
  let lo = toUint32(val2);
  // Shift left 1
  // Get the highest n bits of lo
  const carry = lo >>> (32 - 1);
  lo = toUint32(lo << 1);
  hi = (hi << 1) | carry;  // no way that this can turn negative.
  if (val < 0) {
    if (lo !== 0) {
      lo--;
    } else {
      hi--;
      lo = 0xffffffff;
    }
  }

  function append(num, start, end, arr, j) {
    for (let i = start; i < end; i += 7) {
      if (j !== 0) {
        arr[j - 1] |= 0x80;
      }
      arr[j++] = (num & (0x7f << i)) >>> i;
    }
    return j;
  }

  if (hi === 0) {
    const sigbits = 32 - Math.clz32(lo);
    const byteLength = Math.max(1, Math.ceil(sigbits / 7));
    const arr = Bytes.alloc(byteLength);
    append(lo, 0, sigbits, arr, 0);
    Bytes.copy(arr, buf, offset);
    return byteLength;
  } else {
    const sigbits = 64 - Math.clz32(hi);
    const byteLength = Math.ceil(sigbits / 7);
    const arr = Bytes.alloc(byteLength);

    // All lo, bit 0 through 28
    let j = append(lo, 0, 28, arr, 0);

    // Get the 4 remaining from lo and 3 from hi
    arr[j - 1] |= 0x80;
    arr[j++] = lo >>> 32 - 4 | (hi & 0b111) << 4;

    // And the hi bits
    append(hi, 3, sigbits - 32, arr, j);

    Bytes.copy(arr, buf, offset);
    return byteLength;
  }
}

/**
 * Decodes a signed varint from `buf`, starting at `offset`. This returns an array of the number and
 * the number of bytes consumed.
 */
export function decode(buf: Uint8Array, offset: number): [number, number] {
  let hi = 0, lo = 0, shift = 0, count;
  for (let i = offset; i < buf.length; i++) {
    const b = buf[i];
    if (shift < 28) {
      // lo
      lo |= (b & 0x7f) << shift;
    } else if (shift < 35) {
      // overlap
      lo |= (b & 0x7f) << 28;
      hi |= (b & 0x7f) >>> 4;
    } else {
      // hi
      hi |= (b & 0x7f) << shift - 32;
    }

    if (b < 0x80) {  // last one
      count = i - offset + 1;
      break;
    }

    shift += 7;
  }

  if (count === undefined) {
    throw new Error('Invalid number encoding');
  }

  let sign = 2;
  // lo can become negative due to `|`
  lo = toUint32(lo);
  if (lo & 1) {
    sign = -2;
    lo++;
  }

  return [(mathPowTwoThirtyTwo * hi + lo) / sign, count];
}

/**
 * The number of bytes needed to encode `n` as a signed varint.
 */
export function encodingLength(n: number): number {
  if (n <= 0) {
    return 1;
  }
  return Math.floor((Math.log2(n) + 1) / 7) + 1;
}
