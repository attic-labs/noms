// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// This is based on ECMA 262 encodeURIComponent and the V8 implementation.
// https://chromium.googlesource.com/v8/v8/+/4.3.49/src/uri.js?autodive=0%2F%2F

// Copyright 2006-2008 the V8 project authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export function byteLength(str: string): number {
  const strLen = str.length;
  let length = 0;
  for (let k = 0; k < strLen; k++) {
    const cc1 = str.charCodeAt(k);
    if (cc1 >= 0xdc00 && cc1 <= 0xdfff) throw new Error('Invalid string');
    if (cc1 < 0xd800 || cc1 > 0xdbff) {
      length = singleLength(cc1, length);
    } else {
      k++;
      if (k === strLen) throw new Error('Invalid string');
      const cc2 = str.charCodeAt(k);
      if (cc2 < 0xdc00 || cc2 > 0xdfff) throw new Error('Invalid string');
      length += 4;
    }
  }
  return length;
}

function singleLength(cc: number, index: number): number {
  if (cc <= 0x007f) {
    index++;
  } else if (cc <= 0x07ff) {
    index += 2;
  } else {
    index += 3;
  }
  return index;
}
