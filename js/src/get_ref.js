// @flow

import Chunk from './chunk.js';
import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';
import {notNull} from './assert.js';
import {Type} from './type.js';

type encodeFn = (v: any, t: Type, cs: ?ChunkStore) => Chunk;
let encodeNomsValue: ?encodeFn = null;

export function getRef(v: any, t: Type): Ref {
  return notNull(encodeNomsValue)(v, t, null).ref;
}

export function ensureRef(r: ?Ref, v: any, t: Type): Ref {
  if (r !== null && r !== undefined && !r.isEmpty()) {
    return r;
  }

  return getRef(v, t);
}

export function setEncodeNomsValue(encode: encodeFn) {
  encodeNomsValue = encode;
}
