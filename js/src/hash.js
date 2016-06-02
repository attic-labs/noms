// @flow

// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import {hex} from './sha1.js';

export const sha1Size = 20;
const pattern = /^(sha1-[0-9a-f]{40})$/;

const sha1Prefix = 'sha1-';
const sha1PrefixLength = sha1Prefix.length;
const emtpyHashStr = sha1Prefix + '0'.repeat(40);

function uint8ArrayToHex(a: Uint8Array): string {
  const hex = [];
  for (let i = 0; i < a.length; i++) {
    hex[i] = byteToAscii[a[i]];
  }
  return hex.join('');
}

function hexToUint8Array(s: string): Uint8Array {
  const digest = new Uint8Array(sha1Size);
  for (let i = 0; i < sha1Size; i++) {
    const hc = asciiToBinary(s.charCodeAt(sha1PrefixLength + 2 * i));
    const lc = asciiToBinary(s.charCodeAt(sha1PrefixLength + 2 * i + 1));
    digest[i] = hc << 4 | lc;
  }
  return digest;
}

export default class Hash {
  _hashStr: string;
  _digest: Uint8Array;

  constructor(digest: Uint8Array) {
    this._digest = digest;
    this._hashStr = '';
  }

  get digest(): Uint8Array {
    return this._digest;
  }

  isEmpty(): boolean {
    return this.toString() === emtpyHashStr;
  }

  equals(other: Hash): boolean {
    return this.toString() === other.toString();
  }

  compare(other: Hash): number {
    const s1 = this.toString();
    const s2 = other.toString();
    return s1 === s2 ? 0 : s1 < s2 ? -1 : 1;
  }

  toString(): string {
    return this._hashStr || (this._hashStr = sha1Prefix + uint8ArrayToHex(this._digest));
  }

  static parse(s: string): Hash {
    const m = s.match(pattern);
    if (!m) {
      throw Error('Could not parse hash: ' + s);
    }
    return new Hash(hexToUint8Array(s));
  }

  static maybeParse(s: string): ?Hash {
    const m = s.match(pattern);
    return m && new Hash(hexToUint8Array(m[1]));
  }

  static fromData(data: Uint8Array): Hash {
    return new Hash(hex(data));
  }
}

export const emptyHash = new Hash(new Uint8Array(sha1Size));

function asciiToBinary(cc: number): number {
  // This only accepts the char code for '0' - '9', 'a' - 'f'
  return cc - (cc <= 57 ? 48 : 87); // '9', '0', 'a' - 10
}

// Precompute '00' to 'ff'.
const byteToAscii = new Array(256);
for (let i = 0; i < 256; i++) {
  byteToAscii[i] = (i < 0x10 ? '0' : '') + i.toString(16);
}
