// @flow

// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import sha1 from './sha1.js';

export const sha1Size = 20;
const pattern = /^sha1-[0-9a-f]{40}$/;

const sha1Prefix = 'sha1-';
const sha1PrefixLength = sha1Prefix.length;

function uint8ArrayToSha1(a: Uint8Array): string {
  const sha1 = new Array(1 + sha1Size * 2);
  sha1[0] = [sha1Prefix];
  for (let i = 0; i < a.length; i++) {
    sha1[i + 1] = byteToAscii[a[i]];
  }
  return sha1.join('');
}

function sha1ToUint8Array(s: string): Uint8Array {
  const digest = new Uint8Array(sha1Size);
  for (let i = 0; i < sha1Size; i++) {
    const hc = asciiToBinary(s.charCodeAt(sha1PrefixLength + 2 * i));
    const lc = asciiToBinary(s.charCodeAt(sha1PrefixLength + 2 * i + 1));
    digest[i] = hc << 4 | lc;
  }
  return digest;
}

export default class Hash {
  _digest: Uint8Array;

  constructor(digest: Uint8Array) {
    // Make a copy to prevent holding on to the data that was passed in.
    this._digest = new Uint8Array(digest);
  }

  get digest(): Uint8Array {
    return this._digest;
  }

  isEmpty(): boolean {
    for (let i = 0; i < sha1Size; i++) {
      if (this._digest[i]) {
        return false;
      }
    }
    return true;
  }

  equals(other: Hash): boolean {
    for (let i = 0; i < sha1Size; i++) {
      if (this._digest[i] !== other._digest[i]) {
        return false;
      }
    }
    return true;
  }

  compare(other: Hash): number {
    for (let i = 0; i < sha1Size; i++) {
      const d = this._digest[i] - other._digest[i];
      if (d) {
        return d;
      }
    }
    return 0;
  }

  toString(): string {
    return uint8ArrayToSha1(this._digest);
  }

  static parse(s: string): Hash {
    if (!pattern.test(s)) {
      throw new Error(`Could not parse hash: ${s}`);
    }
    return new Hash(sha1ToUint8Array(s));
  }

  static maybeParse(s: string): ?Hash {
    if (pattern.test(s)) {
      return new Hash(sha1ToUint8Array(s));
    }
    return null;
  }

  static fromData(data: Uint8Array): Hash {
    return new Hash(sha1(data));
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
