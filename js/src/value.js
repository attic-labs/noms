/* @flow */

import Ref from './ref.js';
import {Type} from './type.js';
import type {primitive} from './primitives.js';
import {invariant} from './assert.js';

export type Value = {
  ref: Ref;
  type: Type;
  equals(other: Value): boolean;
}

export type valueOrPrimitive = Value | primitive;

export function less(v1: valueOrPrimitive, v2: valueOrPrimitive): boolean {
  invariant(v1 !== null && v1 !== undefined && v2 !== null && v2 !== undefined);

  if (v1 instanceof Ref) {
    invariant(v2 instanceof Ref);
    return v1.compare(v2) < 0;
  }

  if (typeof v1 === 'object') {
    invariant(v1.ref instanceof Ref);
    invariant(v2.ref instanceof Ref);
    return v1.ref.compare(v2.ref) < 0;
  }

  if (typeof v1 === 'string') {
    invariant(typeof v2 === 'string');
    return v1 < v2;
  }

  invariant(typeof v1 === 'number');
  invariant(typeof v2 === 'number');
  return v1 < v2;
}

export function equals(v1: valueOrPrimitive, v2: valueOrPrimitive): boolean {
  invariant(v1 !== null && v1 !== undefined && v2 !== null && v2 !== undefined);

  if (typeof v1 === 'object') {
    invariant(typeof v2 === 'object');
    return (v1:Value).equals((v2:Value));
  }

  if (typeof v1 === 'string') {
    invariant(typeof v2 === 'string');
    return v1 === v2;
  }

  invariant(typeof v1 === 'number');
  invariant(typeof v2 === 'number');
  return v1 === v2;
}
