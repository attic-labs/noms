// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import Blob from './blob.js';
import List from './list.js';
import Set from './set.js';
import Map from './map.js';
import Ref from './ref.js';
import Struct, {StructMirror} from './struct.js';
import {CompoundDesc, StructDesc, Type} from './type.js';
import {Kind} from './noms-kind.js';
import {invariant} from './assert.js';

import type Database from './database.js';
import type Value from './value.js';

type walkCb = (v: Value) => ?boolean | Promise<?boolean>;

/**
 * Invokes `cb` once for `v` and each of its descendants. The returned `Promise` is resolved when
 * all invocations to `cb` have been resolved.
 *
 * The return value of `cb` indicates whether to recurse further into the tree. Return false or
 * `Promise.resolve(false)` to continue recursing. Return `true` or `Promise.resolve(true)` to skip
 * this node's children.
 *
 * If `cb` returns undefined or `Promise.resolve()`, the default is to continue recursing (`false`).
 */
export default async function walk(v: Value, ds: Database, cb: walkCb): Promise<void> {
  let skip = cb(v);
  if (skip && skip !== true) {
    // Might be a Promise, but we can't check instanceof: https://phabricator.babeljs.io/T7340.
    skip = await skip;
  }

  if (skip) {
    return;
  }

  switch (typeof v) {
    case 'boolean':
    case 'number':
    case 'string':
      return;
  }

  if (v instanceof Blob) {
    return;
  }

  if (v instanceof Type) {
    switch (v.kind) {
      case Kind.Set:
      case Kind.Union:
      case Kind.Map:
      case Kind.List:
      case Kind.Ref: {
        const {desc} = v;
        invariant(desc instanceof CompoundDesc);
        await Promise.all(desc.elemTypes.map(t => walk(t, ds, cb)));
        return;
      }
      case Kind.Struct: {
        const {desc} = v;
        invariant(desc instanceof StructDesc);
        await Promise.all(desc.fields.map(f => walk(f.type, ds, cb)));
        return;
      }
      case Kind.Blob:
      case Kind.Bool:
      case Kind.Cycle:
      case Kind.Number:
      case Kind.String:
      case Kind.Type:
      case Kind.Value:
        return;
      default:
        throw new Error('unreachable');
    }
  }

  if (v instanceof Ref) {
    return walk(await v.targetValue(ds), ds, cb);
  }

  const p = [];
  if (v instanceof List || v instanceof Set) {
    await v.forEach(cv => void(p.push(walk(cv, ds, cb))));
  } else if (v instanceof Map) {
    await v.forEach((cv, k) => {
      p.push(walk(k, ds, cb));
      p.push(walk(cv, ds, cb));
    });
  } else if (v instanceof Struct) {
    new StructMirror(v).forEachField(f => {
      p.push(walk(f.value, ds, cb));
    });
  } else {
    throw new Error('not reached');
  }

  return Promise.all(p).then();
}
