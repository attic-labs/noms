/* @flow */

import type {ChunkStore} from './chunk_store.js';
import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {Kind} from './noms_kind.js';
import {less, equals} from './value.js';
import {MetaSequence, MetaSequenceCursor, MetaTuple, registerMetaValue} from './meta_sequence.js';
import {search, Sequence} from './sequence.js';
import {Type} from './type.js';

export type NSSet<T: valueOrPrimitive> = {
  first(): Promise<T>;
  has(key: T): Promise<boolean>;
  forEach(cb: (v: T) => void): Promise<void>;
  size: number;
}

export class SetLeaf<T:valueOrPrimitive> extends Sequence<T> {

  constructor(cs: ChunkStore, type: Type, items: Array<T>) {
    super(cs, type, items);
    invariant(type.kind === Kind.Set);
  }

  indexOf(key: T): number {
    return search(this.items.length, (i: number) => {
      return !less(this.items[i], key);
    });
  }

  first(): Promise<T> {
    invariant(this.items.length > 0);
    return Promise.resolve(this.items[0]);
  }

  has(key: T): Promise<boolean> {
    let idx = this.indexOf(key);
    if (idx < this.items.length) {
      let k = this.items[idx];
      if (equals(k, key)) {
        return Promise.resolve(true);
      }
    }

    return Promise.resolve(false);
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

export class CompoundSet<T:valueOrPrimitive> extends MetaSequence {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    // invariant(items are pre-ordered and k/v pairs are of the correct type);
    super(cs, type, items);
  }

  async _findLeaf(key: T): Promise<[MetaSequenceCursor, SetLeaf<T>]> {
    let [cursor, leaf] = await this.newCursor();
    await cursor.seek((carry: any, mt: MetaTuple) => {
      return !less(mt.value, key);
    }, null, null);

    let mt = cursor.getCurrent();

    if (!mt.ref.equals(leaf.ref)) {
      leaf = await mt.readValue(this.cs);
      invariant(leaf instanceof SetLeaf);
    }

    return [cursor, leaf];
  }

  async first(): Promise<T> {
    let leaf = (await this.newCursor())[1];
    return leaf.first();
  }

  async has(key: T): Promise<boolean> {
    let leaf = (await this._findLeaf(key))[1];
    return leaf.has(key);
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

  get size(): number {
    throw new Error('not implemented');
  }
}

registerMetaValue(Kind.Set, (cs, type, tuples) => new CompoundSet(cs, type, tuples));

