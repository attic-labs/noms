// @flow

import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';
import type {Type} from './type.js';
import type {valueOrPrimitive} from './value.js'; // eslint-disable-line no-unused-vars
import {invariant} from './assert.js';
import {Kind} from './noms_kind.js';
import {readValue} from './read_value.js';
import {Value} from './value.js';
import {writeValue} from './encode.js';

export default class RefValue<T: valueOrPrimitive> extends Value {
  _target: Ref;

  constructor(target: Ref, type: Type) {
    invariant(type.kind === Kind.Ref);
    super(type);
    this._target = target;
  }

  targetRef(): Ref {
    return this._target;
  }

  async targetValue(cs: ChunkStore): Promise<T> {
    let v = await readValue(this._target, cs);
    // TODO: Would be nice to have a runtime assert here.
    return v;
  }

  setTargetValue(v: T, cs: ChunkStore): RefValue<T> {
    let ref = writeValue(v, this.type.elemTypes[0], cs);
    return new RefValue(ref, this.type);
  }
}
