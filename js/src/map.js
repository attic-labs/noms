// @flow

import BuzHashBoundaryChecker from './buzhash-boundary-checker.js';
import RefValue from './ref-value.js';
import type DataStore from './data-store.js';
import type {BoundaryChecker, makeChunkFn} from './sequence-chunker.js';
import type {valueOrPrimitive} from './value.js'; // eslint-disable-line no-unused-vars
import type {AsyncIterator} from './async-iterator.js';
import {chunkSequence} from './sequence-chunker.js';
import {Collection} from './collection.js';
import {getCompareFunction, equals} from './compare.js';
import {sha1Size} from './ref.js';
import {getRefOfValueOrPrimitive} from './get-ref.js';
import {invariant} from './assert.js';
import {isPrimitive} from './primitives.js';
import {mapOfValueType, Type} from './type.js';
import {MetaTuple, newOrderedMetaSequenceBoundaryChecker,
  newOrderedMetaSequenceChunkFn} from './meta-sequence.js';
import {OrderedSequence, OrderedSequenceCursor,
  OrderedSequenceIterator} from './ordered-sequence.js';
import diff from './ordered-sequence-diff.js';

export type MapEntry<K: valueOrPrimitive, V: valueOrPrimitive> = {
  key: K,
  value: V,
};

const mapWindowSize = 1;
const mapPattern = ((1 << 6) | 0) - 1;

function newMapLeafChunkFn(t: Type, ds: ?DataStore = null): makeChunkFn {
  return (items: Array<MapEntry>) => {
    const mapLeaf = new MapLeafSequence(ds, t, items);

    let indexValue: ?valueOrPrimitive = null;
    if (items.length > 0) {
      const lastValue = items[items.length - 1];
      if (t.elemTypes[0].ordered) {
        indexValue = lastValue.key;
      } else {
        indexValue = new RefValue(getRefOfValueOrPrimitive(lastValue.key, t.elemTypes[0]));
      }
    }

    const mt = new MetaTuple(mapLeaf, indexValue, items.length);
    return [mt, mapLeaf];
  };
}

function newMapLeafBoundaryChecker(t: Type): BoundaryChecker<MapEntry> {
  return new BuzHashBoundaryChecker(mapWindowSize, sha1Size, mapPattern,
    (entry: MapEntry) => getRefOfValueOrPrimitive(entry.key, t.elemTypes[0]).digest);
}

function buildMapData(t: Type, kvs: Array<any>): Array<MapEntry> {
  // TODO: Assert k & v are of correct type
  const entries = [];
  for (let i = 0; i < kvs.length; i += 2) {
    entries.push({
      key: kvs[i],
      value: kvs[i + 1],
    });
  }
  const compare = getCompareFunction(t.elemTypes[0]);
  entries.sort((v1, v2) => compare(v1.key, v2.key));
  return entries;
}

export function newMap<K: valueOrPrimitive, V: valueOrPrimitive>(kvs: Array<any>,
    type: Type = mapOfValueType): Promise<NomsMap<K, V>> {
  return chunkSequence(null, buildMapData(type, kvs), 0, newMapLeafChunkFn(type),
                       newOrderedMetaSequenceChunkFn(type),
                       newMapLeafBoundaryChecker(type),
                       newOrderedMetaSequenceBoundaryChecker)
  .then((seq: OrderedSequence) => new NomsMap(type, seq));
}

export class NomsMap<K: valueOrPrimitive, V: valueOrPrimitive> extends Collection<OrderedSequence> {
  get chunks(): Array<RefValue> {
    if (this.sequence.isMeta) {
      return super.chunks;
    }

    const chunks = [];
    this.sequence.items.forEach(entry => {
      if (!isPrimitive(entry.key)) {
        chunks.push(...entry.key.chunks);
      }
      if (!isPrimitive(entry.value)) {
        chunks.push(...entry.value.chunks);
      }
    });

    return chunks;
  }

  async has(key: K): Promise<boolean> {
    const cursor = await this.sequence.newCursorAt(key);
    return cursor.valid && equals(cursor.getCurrentKey(), key);
  }


  async _firstOrLast(last: boolean): Promise<?[K, V]> {
    const cursor = await this.sequence.newCursorAt(null, false, last);
    if (!cursor.valid) {
      return undefined;
    }

    const entry = cursor.getCurrent();
    return [entry.key, entry.value];
  }

  first(): Promise<?[K, V]> {
    return this._firstOrLast(false);
  }

  last(): Promise<?[K, V]> {
    return this._firstOrLast(true);
  }

  async get(key: K): Promise<?V> {
    const cursor = await this.sequence.newCursorAt(key);
    if (!cursor.valid) {
      return undefined;
    }

    const entry = cursor.getCurrent();
    return equals(entry.key, key) ? entry.value : undefined;
  }

  async forEach(cb: (v: V, k: K) => void): Promise<void> {
    const cursor = await this.sequence.newCursorAt(null);
    return cursor.iter(entry => {
      cb(entry.value, entry.key);
      return false;
    });
  }

  iterator(): AsyncIterator<MapEntry<K, V>> {
    return new OrderedSequenceIterator(this.sequence.newCursorAt(null));
  }

  iteratorAt(k: K): AsyncIterator<MapEntry<K, V>> {
    return new OrderedSequenceIterator(this.sequence.newCursorAt(k));
  }

  async _splice(cursor: OrderedSequenceCursor, insert: Array<MapEntry>, remove: number):
      Promise<NomsMap<K, V>> {
    const type = this.type;
    const ds = this.sequence.ds;
    const seq = await chunkSequence(cursor, insert, remove, newMapLeafChunkFn(type, ds),
                                    newOrderedMetaSequenceChunkFn(type, ds),
                                    newMapLeafBoundaryChecker(type),
                                    newOrderedMetaSequenceBoundaryChecker);
    invariant(seq instanceof OrderedSequence);
    return new NomsMap(type, seq);
  }

  async set(key: K, value: V): Promise<NomsMap<K, V>> {
    let remove = 0;
    const cursor = await this.sequence.newCursorAt(key, true);
    if (cursor.valid && equals(cursor.getCurrentKey(), key)) {
      const entry = cursor.getCurrent();
      if (equals(entry.value, value)) {
        return this;
      }

      remove = 1;
    }

    return this._splice(cursor, [{key: key, value: value}], remove);
  }

  async remove(key: K): Promise<NomsMap<K, V>> {
    const cursor = await this.sequence.newCursorAt(key);
    if (cursor.valid && equals(cursor.getCurrentKey(), key)) {
      return this._splice(cursor, [], 1);
    }

    return this;
  }

  get size(): number {
    return this.sequence.numLeaves;
  }

  /**
   * Returns a 3-tuple [added, removed, modified] sorted by keys.
   */
  diff(from: NomsMap<K, V>): Promise<[Array<K>, Array<K>, Array<K>]> {
    return diff(from.sequence, this.sequence);
  }
}

export class MapLeafSequence<K: valueOrPrimitive, V: valueOrPrimitive> extends
    OrderedSequence<K, MapEntry<K, V>> {
  getKey(idx: number): K {
    return this.items[idx].key;
  }

  equalsAt(idx: number, other: {key: K, value: V}): boolean {
    const entry = this.items[idx];
    return equals(entry.key, other.key) && equals(entry.value, other.value);
  }
}
