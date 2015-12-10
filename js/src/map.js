/* @flow */

import type {ChunkStore} from './chunk_store.js';
import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {Kind} from './noms_kind.js';
import {less, equals} from './value.js';
import {MetaSequence, MetaSequenceCursor, MetaTuple, registerMetaValue} from './meta_sequence.js';
import {search, Sequence} from './sequence.js';
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
}

export class MapLeaf<K:valueOrPrimitive, V:valueOrPrimitive> extends Sequence<Entry<K, V>> {

  constructor(cs: ChunkStore, type: Type, items: Array<Entry<K, V>>) {
    super(cs, type, items);
    invariant(type.kind === Kind.Map);
  }

  indexOf(key: K): number {
    return search(this.items.length, (i: number) => {
      return !less(this.items[i].key, key);
    });
  }

  _findEntry(key: K): ?Entry<K, V> {
    let idx = this.indexOf(key);
    if (idx < this.items.length) {
      let entry = this.items[idx];
      if (equals(entry.key, key)) {
        return entry;
      }
    }

    return null;
  }

  first(): Promise<[K, V]> {
    invariant(this.items.length > 0);
    let entry = this.items[0];
    return Promise.resolve([entry.key, entry.value]);
  }

  get(key: K): Promise<?V> {
    let entry = this._findEntry(key);
    if (entry) {
      return Promise.resolve(entry.value);
    }

    return Promise.resolve(null);
  }

  has(key: K): Promise<boolean> {
    let entry = this._findEntry(key);
    if (entry) {
      return Promise.resolve(true);
    }

    return Promise.resolve(false);
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

export class CompoundMap<K:valueOrPrimitive, V:valueOrPrimitive> extends MetaSequence<MapLeaf> {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    // invariant(items are pre-ordered and k/v pairs are of the correct type);
    super(cs, type, items);
  }

  async _findLeaf(key: K): Promise<[MetaSequenceCursor, MapLeaf<K, V>]> {
    let [cursor, leaf] = await this.newCursor();
    await cursor.seek((carry: any, mt: MetaTuple) => {
      return !less(mt.value, key);
    }, null, null);

    let mt = cursor.getCurrent();

    if (!mt.ref.equals(leaf.ref)) {
      leaf = await mt.readValue(this.cs);
      invariant(leaf instanceof MapLeaf);
    }

    return [cursor, leaf];
  }

  async first(): Promise<[K, V]> {
    let leaf = (await this.newCursor())[1];
    return leaf.first();
  }

  async get(key: K): Promise<?V> {
    let leaf = (await this._findLeaf(key))[1];
    return leaf.get(key);
  }

  async has(key: K): Promise<boolean> {
    let leaf = (await this._findLeaf(key))[1];
    return leaf.has(key);
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

  get size(): number {
    throw new Error('not implemented');
  }
}

registerMetaValue(Kind.Map, (cs, type, tuples) => new CompoundMap(cs, type, tuples));

