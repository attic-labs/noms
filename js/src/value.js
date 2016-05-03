// @flow

import type Ref from './ref.js';
import type {primitive} from './primitives.js';
import {ensureRef} from './get-ref.js';
import type {Type} from './type.js';
import type RefValue from './ref-value.js';

export class Value {
  _ref: ?Ref;

  constructor() {
    this._ref = null;
  }

  get type(): Type {
    throw new Error('abstract');
  }

  get ref(): Ref {
    return this._ref = ensureRef(this._ref, this);
  }

  equals(other: Value): boolean {
    return this === other || this.ref.equals(other.ref);
  }

  less(other: Value): boolean {
    return this.ref.less(other.ref);
  }

  get chunks(): Array<RefValue> {
    return [];
  }
}

export type valueOrPrimitive = primitive | Value;

export function getChunksOfValue(v: valueOrPrimitive): Array<RefValue> {
  if (v instanceof Value) {
    return v.chunks;
  }

  return [];
}
