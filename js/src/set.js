/* @flow */

import type {ChunkStore} from './chunk_store.js';
import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {Kind} from './noms_kind.js';
import {MetaTuple, registerMetaValue} from './meta_sequence.js';
import {OrderedSequence, OrderedMetaSequence} from './ordered_sequence.js';
import {Type} from './type.js';

export type NSSet<T: valueOrPrimitive> = {
  first(): Promise<T>;
  has(key: T): Promise<boolean>;
  forEach(cb: (v: T) => void): Promise<void>;
  size: number;
};

export class SetLeaf<T:valueOrPrimitive> extends OrderedSequence<T, T> {

  constructor(cs: ChunkStore, type: Type, items: Array<T>) {
    super(cs, type, items);
    invariant(type.kind === Kind.Set);
  }

  getKey(idx: number): T {
    return this.items[idx];
  }

  first(): Promise<T> {
    invariant(this.items.length > 0);
    return Promise.resolve(this.items[0]);
  }

  forEach(cb: (v: T) => void): Promise<void> {
    this.items.forEach((k: T) => {
      cb(k);
    });
    return Promise.resolve();
  }

  get size(): number {
    return this.items.length;
  }
}

export class CompoundSet<T:valueOrPrimitive> extends OrderedMetaSequence<T, SetLeaf> {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    // invariant(items are pre-ordered and k/v pairs are of the correct type);
    super(cs, type, items);
  }

  async first(): Promise<T> {
    let leaf = (await this.newCursor())[1];
    return leaf.first();
  }

  async forEach(cb: (k: T) => void): Promise<void> {
    let cursor = (await this.newCursor())[0];
    do {
      let entry = cursor.getCurrent();
      invariant(entry instanceof MetaTuple);
      let setLeaf = await entry.readValue(this.cs);
      await setLeaf.forEach(cb);
    } while (await cursor.advance());
  }
}

registerMetaValue(Kind.Set, (cs, type, tuples) => new CompoundSet(cs, type, tuples));

