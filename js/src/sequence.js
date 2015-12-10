/* @flow */

import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';
import {ensureRef} from './get_ref.js';
import {invariant, notNull} from './assert.js';
import {Type} from './type.js';

export class Sequence<T> {
  items: Array<T>;
  _ref: ?Ref;
  cs: ChunkStore;
  type: Type;

  constructor(cs: ChunkStore, type: Type, items: Array<T>) {
    this.cs = cs;
    this.items = items;
    this._ref = null;

    this.type = type;
  }

  get ref(): Ref {
    return this._ref = ensureRef(this._ref, this, this.type);
  }

  equals(other: Sequence<T>): boolean {
    return this.ref.equals(other.ref);
  }
}

export type seekPredicateFn<T, I> = (carry: I, value: T) => boolean;

export type seekStepFn<T, I> = (carry: I, prev: ?T, value: T) => I;

export class SequenceCursor<T, S: Sequence> {
  parent: ?SequenceCursor;
  sequence: S;
  idx: number;
  length: number;

  constructor(parent: ?SequenceCursor, sequence: S, idx: number) {
    this.parent = parent;
    this.sequence = sequence;
    this.idx = idx;
    this.length = sequence.items.length;
  }

  getItem(idx: number): T {
    return this.sequence.items[idx];
  }

  readSequence(): Promise<S> {
    throw new Error('override');
  }

  async _sync(): Promise<void> {
    this.sequence = await this.readSequence();
    this.length = this.sequence.items.length;
  }

  getCurrent(): T {
    invariant(this.idx >= 0 && this.idx < this.length);
    return this.getItem(this.idx);
  }

  get indexInChunk(): number {
    return this.idx;
  }

  advance(): Promise<boolean> {
    return this._advanceMaybeAllowPastEnd(true);
  }

  async _advanceMaybeAllowPastEnd(allowPastEnd: boolean): Promise<boolean> {
    if (this.idx < this.length - 1) {
      this.idx++;
      return true;
    }

    if (this.idx === this.length) {
      return false;
    }

    if (this.parent !== null && (await notNull(this.parent)._advanceMaybeAllowPastEnd(false))) {
      await this._sync();
      this.idx = 0;
      return true;
    }
    if (allowPastEnd) {
      this.idx++;
    }

    return false;
  }

  retreat(): Promise<boolean> {
    return this._retreatMaybeAllowBeforeStart(true);
  }

  async _retreatMaybeAllowBeforeStart(allowBeforeStart: boolean): Promise<boolean> {
    if (this.idx > 0) {
      this.idx--;
      return true;
    }
    if (this.idx === -1) {
      return false;
    }
    invariant(this.idx === 0);
    if (this.parent !== null && notNull(this.parent)._retreatMaybeAllowBeforeStart(false)) {
      await this._sync();
      this.idx = this.length - 1;
      return true;
    }

    if (allowBeforeStart) {
      this.idx--;
    }

    return false;
  }

  copy(): SequenceCursor {
    throw new Error('override');
  }

  async seek<I>(predicate: seekPredicateFn<T, I>, step: ?seekStepFn<T, I>, carry: I): Promise<I> {
    if (this.parent !== null) {
      carry = await notNull(this.parent).seek(predicate, step, carry);
      await this._sync();
    }

    let self = this; // TODO
    this.idx = search(this.length, (i: number) => {
      return predicate(carry, self.getItem(i));
    });

    if (this.idx === this.length) {
      this.idx = this.length - 1;
    }

    if (step === null) {
      return carry;
    }

    let prev = this.idx > 0 ? this.getItem(this.idx - 1) : null;
    return notNull(step)(carry, prev, this.getItem(this.idx));
  }

  maxNPrevItems(n: number): Array<T> {
    let prev: Array<T> = [];

    let retreater = this.copy();
    for (let i = 0; i < n && retreater.retreat(); i++) {
      let v = retreater.getCurrent();
      prev.unshift(v);
    }

    return prev;
  }
}

// Translated from golang source (https://golang.org/src/sort/search.go?s=2249:2289#L49)
export function search(n: number, f: (i: number) => boolean): number {
  // Define f(-1) == false and f(n) == true.
  // Invariant: f(i-1) == false, f(j) == true.
  let i = 0;
  let j = n;
  while (i < j) {
    let h = i + Math.floor((j - i) / 2); // avoid overflow when computing h
    // i ≤ h < j
    if (!f(h)) {
      i = h + 1; // preserves f(i-1) == false
    } else {
      j = h; // preserves f(j) == true
    }
  }

  // i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
  return i;
}
