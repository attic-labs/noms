/* @flow */

import type {ChunkStore} from './chunk_store.js';
import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {Kind} from './noms_kind.js';
import {MetaTuple, registerMetaValue} from './meta_sequence.js';
import {OrderedSequence, OrderedMetaSequence} from './ordered_sequence.js';

import {Type} from './type.js';

type Entry<K: valueOrPrimitive, V: valueOrPrimitive> = {
  key: K,
  value: V
};

export type NSMap<K: valueOrPrimitive, V: valueOrPrimitive> = {
  first(): Promise<[K, V]>;
  get(key: K): Promise<?V>;
  has(key: K): Promise<boolean>;
  forEach(cb: (v: V, k: K) => void): Promise<void>;
  size: number;
};

export class MapLeaf<K:valueOrPrimitive, V:valueOrPrimitive> extends OrderedSequence<K, Entry<K, V>> {

  constructor(cs: ChunkStore, type: Type, items: Array<Entry<K, V>>) {
    super(cs, type, items);
    invariant(type.kind === Kind.Map);
  }

  getKey(idx: number): K {
    return this.items[idx].key;
  }

  first(): Promise<[K, V]> {
    invariant(this.items.length > 0);
    let entry = this.items[0];
    return Promise.resolve([entry.key, entry.value]);
  }

  get(key: K): Promise<?V> {
    let [idx, found] = this.indexOf(key);
    if (found) {
      return Promise.resolve(this.items[idx].value);
    }

    return Promise.resolve(null);
  }

  forEach(cb: (v: V, k: K) => void): Promise<void> {
    this.items.forEach((entry: Entry<K, V>) => {
      cb(entry.value, entry.key);
    });
    return Promise.resolve();
  }

  get size(): number {
    return this.items.length;
  }
}

export class CompoundMap<K:valueOrPrimitive, V:valueOrPrimitive> extends OrderedMetaSequence<K, MapLeaf> {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    // invariant(items are pre-ordered and k/v pairs are of the correct type);
    super(cs, type, items);
  }

  async first(): Promise<[K, V]> {
    let leaf = (await this.newCursor())[1];
    return leaf.first();
  }

  async get(key: K): Promise<?V> {
    let leaf = (await this.findLeaf(key))[1];
    return leaf.get(key);
  }

  async forEach(cb: (v: V, k: K) => void): Promise<void> {
    let cursor = (await this.newCursor())[0];
    do {
      let entry = cursor.getCurrent();
      invariant(entry instanceof MetaTuple);
      let mapLeaf = await entry.readValue(this.cs);
      await mapLeaf.forEach(cb);
    } while (await cursor.advance());
  }
}

registerMetaValue(Kind.Map, (cs, type, tuples) => new CompoundMap(cs, type, tuples));

