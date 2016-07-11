// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import Ref from './ref.js';
import type {NomsKind} from './noms-kind.js';
import {invariant} from './assert.js';
import {isPrimitiveKind, Kind} from './noms-kind.js';
import {ValueBase} from './value.js';
import type Value from './value.js';
import {equals} from './compare.js';
import {describeType} from './encode-human-readable.js';
import search from './binary-search.js';
import {staticTypeCache} from './type-cache.js';

export interface TypeDesc {
  kind: NomsKind;
  equals(other: TypeDesc): boolean;
  hasUnresolvedCycle(visited: Type[]): boolean;
}

export class PrimitiveDesc {
  kind: NomsKind;

  constructor(kind: NomsKind) {
    this.kind = kind;
  }

  equals(other: TypeDesc): boolean {
    return other instanceof PrimitiveDesc && other.kind === this.kind;
  }

  hasUnresolvedCycle(visited: Type[]): boolean { // eslint-disable-line no-unused-vars
    return false;
  }
}

export class CompoundDesc {
  kind: NomsKind;
  elemTypes: Array<Type>;

  constructor(kind: NomsKind, elemTypes: Array<Type>) {
    this.kind = kind;
    this.elemTypes = elemTypes;
  }

  equals(other: TypeDesc): boolean {
    if (other instanceof CompoundDesc) {
      if (this.kind !== other.kind || this.elemTypes.length !== other.elemTypes.length) {
        return false;
      }

      for (let i = 0; i < this.elemTypes.length; i++) {
        if (!equals(this.elemTypes[i], other.elemTypes[i])) {
          return false;
        }
      }

      return true;
    }

    return false;
  }

  hasUnresolvedCycle(visited: Type[]): boolean {
    return this.elemTypes.some(t => t.hasUnresolvedCycle(visited));
  }
}

export type Field = {
  name: string;
  type: Type;
};

export class StructDesc {
  name: string;
  fields: Field[];

  constructor(name: string, fields: Field[]) {
    this.name = name;
    this.fields = fields;
  }

  get fieldCount(): number {
    return this.fields.length;
  }

  get kind(): NomsKind {
    return Kind.Struct;
  }

  equals(other: TypeDesc): boolean {
    if (this === other) {
      return true;
    }

    if (other.kind !== this.kind) {
      return false;
    }
    invariant(other instanceof StructDesc);

    const fields = this.fields;
    const otherFields = other.fields;

    if (fields.length !== otherFields.length) {
      return false;
    }

    for (let i = 0; i < fields.length; i++) {
      if (fields[i].name !== otherFields[i].name || !equals(fields[i].type, otherFields[i].type)) {
        return false;
      }
    }

    return true;
  }

  hasUnresolvedCycle(visited: Type[]): boolean {
    return this.fields.some(f => f.type.hasUnresolvedCycle(visited));
  }

  forEachField(cb: (name: string, type: Type) => void) {
    const fields = this.fields;
    for (let i = 0; i < fields.length; i++) {
      cb(fields[i].name, fields[i].type);
    }
  }

  getField(name: string): ?Type {
    const f = findField(name, this.fields);
    return f && f.type;
  }
}

function findField(name: string, fields: Field[]): ?Field {
  const i = findFieldIndex(name, fields);
  return i !== -1 ? fields[i] : undefined;
}

/**
 * Finds the index of the `Field` or `-1` if not found.
 */
export function findFieldIndex(name: string, fields: Field[]): number {
  const i = search(fields.length, i => {
    const n = fields[i].name;
    return n === name ? 0 : n > name ? 1 : -1;
  });
  return i === fields.length || fields[i].name !== name ? -1 : i;
}

export class CycleDesc {
  level: number;

  constructor(level: number) {
    this.level = level;
  }

  get kind(): NomsKind {
    return Kind.Cycle;
  }

  equals(other: TypeDesc): boolean {
    return other instanceof CycleDesc && other.level === this.level;
  }

  hasUnresolvedCycle(visited: Type[]): boolean { // eslint-disable-line no-unused-vars
    return true;
  }
}

export class Type<T: TypeDesc> extends ValueBase {
  _desc: T;
  id: number;
  serialization: ?Uint8Array;

  constructor(desc: T, id: number) {
    super();
    this._desc = desc;
    this.id = id;
    this.serialization = null;
  }

  get type(): Type {
    return typeType;
  }

  get chunks(): Array<Ref> {
    return [];
  }

  get kind(): NomsKind {
    return this._desc.kind;
  }

  get desc(): T {
    return this._desc;
  }

  get name(): string {
    invariant(this._desc instanceof StructDesc);
    return this._desc.name;
  }

  hasUnresolvedCycle(visited: Type[] = []): boolean {
    if (visited.indexOf(this) >= 0) {
      return false;
    }

    visited.push(this);
    return this._desc.hasUnresolvedCycle(visited);
  }

  get elemTypes(): Array<Type> {
    invariant(this._desc instanceof CompoundDesc);
    return this._desc.elemTypes;
  }

  describe(): string {
    return describeType(this);
  }
}

function makePrimitiveType(k: NomsKind): Type<PrimitiveDesc> {
  return new Type(new PrimitiveDesc(k), k);
}

export function makeListType(elemType: Type): Type<CompoundDesc> {
  return staticTypeCache.getCompoundType(Kind.List, elemType);
}

export function makeSetType(elemType: Type): Type<CompoundDesc> {
  return staticTypeCache.getCompoundType(Kind.Set, elemType);
}

export function makeMapType(keyType: Type, valueType: Type): Type<CompoundDesc> {
  return staticTypeCache.getCompoundType(Kind.Map, keyType, valueType);
}

export function makeRefType(elemType: Type): Type<CompoundDesc> {
  return staticTypeCache.getCompoundType(Kind.Ref, elemType);
}

export function makeStructType(name: string, fieldNames: string[], fieldTypes: Type[]):
    Type<StructDesc> {
  return staticTypeCache.makeStructType(name, fieldNames, fieldTypes);
}

export function makeUnionType(types: Type[]): Type {
  return staticTypeCache.makeUnionType(types);
}

export function makeCycleType(level: number): Type {
  return staticTypeCache.getCycleType(level);
}

/**
 * Gives the existing primitive Type value for a NomsKind.
 */
export function getPrimitiveType(k: NomsKind): Type {
  invariant(isPrimitiveKind(k));
  switch (k) {
    case Kind.Bool:
      return boolType;
    case Kind.Number:
      return numberType;
    case Kind.String:
      return stringType;
    case Kind.Blob:
      return blobType;
    case Kind.Type:
      return typeType;
    case Kind.Value:
      return valueType;
    default:
      invariant(false, 'not reachable');
  }
}

// Returns the Noms type of any value. This will throw if you pass in an object that cannot be
// represented by noms.
export function getTypeOfValue(v: Value): Type {
  if (v instanceof ValueBase) {
    return v.type;
  }

  switch (typeof v) {
    case 'string':
      return stringType;
    case 'boolean':
      return boolType;
    case 'number':
      return numberType;
    default:
      throw new Error('Unknown type');
  }
}

export const boolType = makePrimitiveType(Kind.Bool);
export const numberType = makePrimitiveType(Kind.Number);
export const stringType = makePrimitiveType(Kind.String);
export const blobType = makePrimitiveType(Kind.Blob);
export const typeType = makePrimitiveType(Kind.Type);
export const valueType = makePrimitiveType(Kind.Value);
