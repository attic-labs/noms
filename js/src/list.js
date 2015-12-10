/* @flow */

import type {ChunkStore} from './chunk_store.js';
import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {invariant, notNull} from './assert.js';
import {Kind} from './noms_kind.js';
import {MetaSequence, MetaSequenceCursor, MetaTuple, registerMetaValue} from './meta_sequence.js';
import {Sequence} from './sequence.js';
import {Type} from './type.js';

export type NSList<T: valueOrPrimitive> = {
  get(idx: number): T;
  forEach(cb: (v: T, i: number) => void): Promise<void>;
  length: number;
}

export class ListLeaf<T:valueOrPrimitive> extends Sequence<T> {

  constructor(cs: ChunkStore, type: Type, items: Array<T>) {
    super(cs, type, items);
    invariant(type.kind === Kind.List);
  }

  async get(idx: number): Promise<T> {
    return Promise.resolve(this.items[idx]);
  }

  forEach(cb: (v: T, i: number) => void): Promise<void> {
    this.items.forEach((v: T, i: number) => {
      cb(v, i);
    });
    return Promise.resolve();
  }

  get length(): number {
    return this.items.length;
  }
}

export class CompoundList<T:valueOrPrimitive> extends MetaSequence {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    // invariant(items are pre-ordered and k/v pairs are of the correct type);
    super(cs, type, items);
  }

  async _cursorAt(idx: number): Promise<[MetaSequenceCursor, ListLeaf<T>, number]> {
    let [cursor, leaf] = await this.newCursor();
    let chunkStart = await cursor.seek((carry: number, mt: MetaTuple) => {
      return idx < carry + mt.value;
    }, (carry: number, prev: ?MetaTuple) => {
      let pv = prev ? prev.value : 0;
      return carry + pv;
    }, 0);

    let mt = cursor.getCurrent();

    if (!mt.ref.equals(leaf.ref)) {
      leaf = await mt.readValue(this.cs);
      invariant(leaf instanceof ListLeaf);
    }

    return [cursor, leaf, chunkStart];
  }

  async get(idx: number): Promise<T> {
    let [cursor, leaf, start] = await this._cursorAt(idx);
    notNull(cursor);
    return leaf.get(idx - start);
  }

  async forEach(cb: (v: T, i: number) => void): Promise<void> {
    let start = 0;
    let cursor = (await this.newCursor())[0];
    do {
      let entry = cursor.getCurrent();
      invariant(entry instanceof MetaTuple);
      let listLeaf = await entry.readValue(this.cs);
      listLeaf.items.forEach((v: T, i: number) => {
        cb(v, start + i);
      });

      start += listLeaf.length;
    } while (await cursor.advance());
  }

  get length(): number {
    throw new Error('not implemented');
  }
}

registerMetaValue(Kind.List, (cs, type, tuples) => new CompoundList(cs, type, tuples));

