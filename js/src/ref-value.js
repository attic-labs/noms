// @flow

import type {ValueReader} from './value-store.js';
import {describeType} from './encode-human-readable.js';
import {getRefOfValue} from './get-ref.js';
import {Kind} from './noms-kind.js';
import type Ref from './ref.js';
import type {Type} from './type.js';
import type {valueOrPrimitive} from './value.js'; // eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {getTypeOfValue, makeRefType} from './type.js';
import {Value} from './value.js';

export function refValueFromValue(val: valueOrPrimitive): RefValue {
  let height = 1;
  if (val instanceof Value) {
    height += val.chunks.reduce((max, c) => Math.max(max, c.height), 0);
  }
  return new RefValue(getRefOfValue(val), height, makeRefType(getTypeOfValue(val)));
}

export default class RefValue<T: valueOrPrimitive> extends Value {
  _type: Type;
  // Ref of the value this points to.
  targetRef: Ref;
  // Height, the length of the longest path of RefValues to find any leaf in the graph.
  // If targetRef isn't a dangling pointer, this must by definition be >= 1.
  height: number;

  constructor(targetRef: Ref, height: number, t: Type) {
    super();
    invariant(t.kind === Kind.Ref, () => `Not a Ref type: ${describeType(t)}`);
    this._type = t;
    this.targetRef = targetRef;
    this.height = height;
  }

  get type(): Type {
    return this._type;
  }

  targetValue(vr: ValueReader): Promise<T> {
    return vr.readValue(this.targetRef);
  }

  less(other: Value): boolean {
    invariant(other instanceof RefValue);
    return this.targetRef.less(other.targetRef);
  }

  get chunks(): Array<RefValue> {
    return [this];
  }
}
