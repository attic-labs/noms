// @flow

import {invariant} from './assert.js';
import {getDatasTypes} from './database.js';
import Struct, {StructMirror} from './struct.js';
import type {valueOrPrimitive} from './value.js';
import type RefValue from './ref-value.js';
import Set from './set.js';

export default class Commit<T: valueOrPrimitive> extends Struct {
  // Hold a reference to the struct 'parents' and 'value' fields so that it can be the correct type
  // when returned from the parents/value getters.
  _parents: Set<RefValue<Commit>>;
  _value: T;

  constructor(value: T, parentsArr: Array<RefValue<Commit>> = []) {
    const parents = new Set(parentsArr);
    const types = getDatasTypes();
    super(types.commitType, {value, parents});
    this._parents = parents;
    this._value = value;
  }

  get parents(): Set<RefValue<Commit>> {
    invariant(new StructMirror(this).get('parents') === this._parents);
    return this._parents;
  }

  get value(): T {
    invariant(new StructMirror(this).get('value') === this._value);
    return this._value;
  }
}
