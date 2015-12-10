/* @flow */

import type {valueOrPrimitive} from './value.js'; //eslint-disable-line no-unused-vars
import {notNull} from './assert.js';
import {less, equals} from './value.js';
import {search, Sequence} from './sequence.js';

export class OrderedSequence<K:valueOrPrimitive, T> extends Sequence<T> {

  getKey(idx: number): K {
    notNull(idx);
    throw new Error('override');
  }

  indexOf(key: K): [number, boolean] {
    let idx = search(this.items.length, (i: number) => {
      return !less(this.getKey(i), key);
    });

    if (idx < this.items.length) {
      return [idx, equals(this.getKey(idx), key)];
    }

    return [idx, false];
  }

  has(key: K): Promise<boolean> {
    let found = this.indexOf(key)[1];
    if (found) {
      return Promise.resolve(true);
    }

    return Promise.resolve(false);
  }
}
