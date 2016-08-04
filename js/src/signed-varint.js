// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

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

  const sigbits = hi !== 0 ? 64 - Math.clz32(hi) : lo === 0 ? 1 : 32 - Math.clz32(lo);
  const byteLength = Math.ceil(sigbits / 7);
  let j = offset;
  for (let i = 0; i < sigbits; i += 7) {
    if (i !== 0) {
      buf[j - 1] |= 0x80;
    }
    if (i < 28) {
      buf[j++] = (lo & (0x7f << i)) >>> i;
    } else if (i < 35) {
      buf[j++] = lo >>> 32 - 4 | (hi & 0b111) << 4;
    } else {
      buf[j++] = (hi & (0x7f << i)) >>> i;
    }
  }
  return byteLength;
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
  // TODO: Clean this up. Remember to not overflow though!
  if (n === 0) {
    return 1;
  }
  let negative = false;
  if (n < 0) {
    negative = true;
    n = -n;
  }

  const l2 = Math.log2(n);
  const bits = Math.ceil(l2);

  let rv = Math.floor((l2 + 1) / 7) + 1;
  // If negative and an exact power of 2 and 1 bit over a multiple of 7 the +1 reduces the result
  // by one.
  if (negative && l2 === bits && l2 % 7 === 6) {
    rv--;
  }

  return rv;
}
