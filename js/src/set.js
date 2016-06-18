// @flow

// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import BuzHashBoundaryChecker from './buzhash-boundary-checker.js';
import Ref from './ref.js';
import type {ValueReader} from './value-store.js';
import type {BoundaryChecker, makeChunkFn} from './sequence-chunker.js';
import type Value from './value.js'; // eslint-disable-line no-unused-vars
import {ValueBase} from './value.js';
import {AsyncIterator} from './async-iterator.js';
import {chunkSequence, chunkSequenceSync} from './sequence-chunker.js';
import Collection from './collection.js';
import {compare, equals} from './compare.js';
import {getHashOfValue} from './get-hash.js';
import {invariant} from './assert.js';
import {MetaTuple, newOrderedMetaSequenceBoundaryChecker, newOrderedMetaSequenceChunkFn} from
  './meta-sequence.js';
import {OrderedSequence, OrderedSequenceCursor, OrderedSequenceIterator} from
  './ordered-sequence.js';
import diff from './ordered-sequence-diff.js';
import {makeSetType, makeUnionType, getTypeOfValue} from './type.js';
import {sha1Size} from './hash.js';
import {removeDuplicateFromOrdered} from './map.js';
import {getValueChunks} from './sequence.js';
import {Kind} from './noms-kind.js';
import type {EqualsFn} from './edit-distance.js';

const setWindowSize = 1;
const setPattern = ((1 << 6) | 0) - 1;

function newSetLeafChunkFn<T:Value>(vr: ?ValueReader): makeChunkFn {
  return (items: Array<T>) => {
    let indexValue: ?(T | Ref) = null;
    if (items.length > 0) {
      indexValue = items[items.length - 1];
      if (indexValue instanceof ValueBase) {
        indexValue = new Ref(indexValue);
      }
    }

    const seq = newSetLeafSequence(vr, items);
    const ns = Set.fromSequence(seq);
    const mt = new MetaTuple(new Ref(ns), indexValue, items.length, ns);
    return [mt, seq];
  };
}

function newSetLeafBoundaryChecker<T:Value>(): BoundaryChecker<T> {
  return new BuzHashBoundaryChecker(setWindowSize, sha1Size, setPattern, (v: T) => {
    const hash = getHashOfValue(v);
    return hash.digest;
  });
}

function buildSetData<T: Value>(values: Array<any>): Array<T> {
  values = values.slice();
  values.sort(compare);
  return removeDuplicateFromOrdered(values, compare);
}

export function newSetLeafSequence<K: Value>(
    vr: ?ValueReader, items: K[]): SetLeafSequence {
  const t = makeSetType(makeUnionType(items.map(getTypeOfValue)));
  return new SetLeafSequence(vr, t, items);
}

export default class Set<T: Value> extends Collection<OrderedSequence> {
  constructor(values: Array<T> = []) {
    const seq = chunkSequenceSync(
        buildSetData(values),
        newSetLeafChunkFn(null),
        newOrderedMetaSequenceChunkFn(Kind.Set, null),
        newSetLeafBoundaryChecker(),
        newOrderedMetaSequenceBoundaryChecker);
    invariant(seq instanceof OrderedSequence);
    super(seq);
  }

  async has(key: T): Promise<boolean> {
    const cursor = await this.sequence.newCursorAt(key);
    return cursor.valid && equals(cursor.getCurrentKey(), key);
  }

  async _firstOrLast(last: boolean): Promise<?T> {
    const cursor = await this.sequence.newCursorAt(null, false, last);
    return cursor.valid ? cursor.getCurrent() : null;
  }

  first(): Promise<?T> {
    return this._firstOrLast(false);
  }

  last(): Promise<?T> {
    return this._firstOrLast(true);
  }

  async forEach(cb: (v: T) => ?Promise<void>): Promise<void> {
    const cursor = await this.sequence.newCursorAt(null);
    const promises = [];
    return cursor.iter(v => {
      promises.push(cb(v));
      return false;
    }).then(() => Promise.all(promises)).then(() => void 0);
  }

  iterator(): AsyncIterator<T> {
    return new OrderedSequenceIterator(this.sequence.newCursorAt(null));
  }

  iteratorAt(v: T): AsyncIterator<T> {
    return new OrderedSequenceIterator(this.sequence.newCursorAt(v));
  }

  _splice(cursor: OrderedSequenceCursor, insert: Array<T>, remove: number):
      Promise<Set<T>> {
    const vr = this.sequence.vr;
    return chunkSequence(cursor, insert, remove, newSetLeafChunkFn(vr),
                         newOrderedMetaSequenceChunkFn(Kind.Set, vr),
                         newSetLeafBoundaryChecker(),
                         newOrderedMetaSequenceBoundaryChecker).then(s => Set.fromSequence(s));
  }

  async add(value: T): Promise<Set<T>> {
    const cursor = await this.sequence.newCursorAt(value, true);
    if (cursor.valid && equals(cursor.getCurrentKey(), value)) {
      return this;
    }

    return this._splice(cursor, [value], 0);
  }

  async delete(value: T): Promise<Set<T>> {
    const cursor = await this.sequence.newCursorAt(value);
    if (cursor.valid && equals(cursor.getCurrentKey(), value)) {
      return this._splice(cursor, [], 1);
    }

    return this;
  }

  // TODO: Find some way to return a Set.
  async map<S>(cb: (v: T) => (Promise<S> | S)): Promise<Array<S>> {
    const cursor = await this.sequence.newCursorAt(null);
    const values = [];
    await cursor.iter(v => {
      values.push(cb(v));
      return false;
    });

    return Promise.all(values);
  }

  get size(): number {
    return this.sequence.numLeaves;
  }

  async intersect(...sets: Array<Set<T>>): Promise<Set<T>> {
    if (sets.length === 0) {
      return this;
    }

    let cursor = await this.sequence.newCursorAt(null);
    if (!cursor.valid) {
      return this;
    }

    const values: Array<T> = [];

    for (let i = 0; cursor.valid && i < sets.length; i++) {
      const first = cursor.getCurrent();
      const next = await sets[i].sequence.newCursorAt(first);
      if (!next.valid) {
        break;
      }

      cursor = new SetIntersectionCursor(cursor, next);
      await cursor.align();
    }

    while (cursor.valid) {
      values.push(cursor.getCurrent());
      await cursor.advance();
    }

    return new Set(values);
  }

  /**
   * Returns a 2-tuple [added, removed] sorted values.
   */
  diff(from: Set<T>): Promise<[Array<T> /* added */, Array<T> /* removed */]> {
    return diff(from.sequence, this.sequence).then(([added, removed, modified]) => {
      invariant(modified.length === 0);
      return [added, removed];
    });
  }
}

export class SetLeafSequence<K: Value> extends OrderedSequence<K, K> {
  getKey(idx: number): K {
    return this.items[idx];
  }

  getCompareFn(other: OrderedSequence): EqualsFn {
    return (idx: number, otherIdx: number) =>
      equals(this.items[idx], other.items[otherIdx]);
  }

  get chunks(): Array<Ref> {
    return getValueChunks(this.items);
  }
}

type OrderedCursor<K: Value> = {
  valid: boolean;
  getCurrent(): K;
  advanceTo(key: K): Promise<boolean>;
  advance(): Promise<boolean>;
}

class SetIntersectionCursor<K: Value> {
  s1: OrderedCursor<K>;
  s2: OrderedCursor<K>;
  valid: boolean;

  constructor(s1: OrderedCursor<K>, s2: OrderedCursor<K>) {
    invariant(s1.valid && s2.valid);
    this.s1 = s1;
    this.s2 = s2;
    this.valid = true;
  }

  getCurrent(): K {
    invariant(this.valid);
    return this.s1.getCurrent();
  }

  async align(): Promise<boolean> {
    let v1 = this.s1.getCurrent();
    let v2 = this.s2.getCurrent();

    let i;
    while ((i = compare(v1, v2)) !== 0) {
      if (i < 0) {
        if (!await this.s1.advanceTo(v2)) {
          return this.valid = false;
        }

        v1 = this.s1.getCurrent();
        continue;
      }

      if (!await this.s2.advanceTo(v1)) {
        return this.valid = false;
      }

      v2 = this.s2.getCurrent();
    }

    return this.valid = true;
  }

  async advanceTo(key: K): Promise<boolean> {
    invariant(this.valid);
    return this.valid = await this.s1.advanceTo(key) && await this.align();
  }

  async advance(): Promise<boolean> {
    invariant(this.valid);
    return this.valid = await this.s1.advance() && await this.align();
  }
}
