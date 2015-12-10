// @flow

import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';
import type {NomsKind} from './noms_kind.js';
import {CompoundDesc, makeCompoundType, makePrimitiveType, Type} from './type.js';
import {invariant, notNull} from './assert.js';
import {Kind} from './noms_kind.js';
import {readValue} from './read_value.js';
import {Sequence, SequenceCursor} from './sequence.js';

export class MetaSequence<T:Sequence> extends Sequence<MetaTuple> {
  constructor(cs: ChunkStore, type: Type, items: Array<MetaTuple>) {
    super(cs, type, items);
  }

  async newCursor(): Promise<[MetaSequenceCursor, T]> {
    let level = 0;
    let cursors: Array<MetaSequenceCursor> = [new MetaSequenceCursor(null, this, 0, level)];

    let done = false;
    while (!done) {
      let cursor = notNull(cursors[cursors.length - 1]);
      let mt = cursor.getCurrent();
      let sequence = await mt.readValue(this.cs);
      if (sequence instanceof MetaSequence) {
        level++;
        cursors.push(new MetaSequenceCursor(cursor, sequence, 0, level));
      } else {
        invariant(sequence instanceof Sequence);
        return [cursor, sequence];
      }
    }

    throw new Error('not reached');
  }
}

export class MetaTuple {
  ref: Ref;
  value: any;

  constructor(ref: Ref, value: any) {
    this.ref = ref;
    this.value = value;
  }

  readValue(cs: ChunkStore): Promise<any> {
    return readValue(this.ref, cs);
  }
}

export type metaBuilderFn = (cs: ChunkStore, t: Type, tuples: Array<MetaTuple>) => MetaSequence;

let metaFuncMap: Map<NomsKind, metaBuilderFn> = new Map();

export function newMetaSequenceFromData(cs: ChunkStore, t: Type, data: Array<MetaTuple>): MetaSequence {
  let ctor = notNull(metaFuncMap.get(t.kind));
  return ctor(cs, t, data);
}

export function registerMetaValue(k: NomsKind, bf: metaBuilderFn) {
  metaFuncMap.set(k, bf);
}

export class MetaSequenceCursor extends SequenceCursor<MetaTuple, MetaSequence> {

  constructor(parent: ?MetaSequenceCursor, sequence: MetaSequence, idx: number) {
    super(parent, sequence, idx);
  }

  async readSequence(): Promise<MetaSequence> {
    let mt: ?MetaTuple = null;
    if (this.parent !== null && this.parent !== undefined) {
      mt = this.parent.getCurrent();
      let ms = await mt.readValue(this.sequence.cs);
      invariant(ms instanceof MetaSequence);
      return ms;
    }
    throw new Error('fixme');
  }

  copy(): MetaSequenceCursor {
    let parent: ?SequenceCursor = this.parent ? this.parent.copy() : null;
    invariant(parent === null || parent instanceof MetaSequenceCursor);
    return new MetaSequenceCursor(parent, this.sequence, this.idx);
  }
}

export function indexTypeForMetaSequence(t: Type): Type {
  switch (t.kind) {
    case Kind.Map:
    case Kind.Set: {
      let desc = t.desc;
      invariant(desc instanceof CompoundDesc);
      let elemType = desc.elemTypes[0];
      if (elemType.ordered) {
        return elemType;
      } else {
        return makeCompoundType(Kind.Ref, makePrimitiveType(Kind.Value));
      }
    }
    case Kind.Blob:
    case Kind.List:
      return makePrimitiveType(Kind.Uint64);
  }

  throw new Error('Not reached');
}
