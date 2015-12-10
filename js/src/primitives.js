/* @flow */

import Ref from './ref.js';
import {makePrimitiveType, Type} from './type.js';
import {invariant} from './assert.js';
import {ensureRef} from './get_ref.js';
import {Kind} from './noms_kind.js';

export type Value = {
  ref: Ref;
  type: Type;
  equals(other: Value): boolean;
}

class Primitive<T: primitive> {
  val: T;
  type: Type;

  constructor(val: T, type: Type) {
    this.val = val;
    this.type = type;
  }

  get ref(): Ref {
    return ensureRef(null, this.val, this.type);
  }

  equals(other: Value): boolean {
    return this.ref.equals(other.ref);
  }
}

// TODO: What to do about (u)int64 lack of precision?
function isSafeInteger(value): boolean {
  return Number.isInteger(value) && Math.abs(value) <= Number.MAX_SAFE_INTEGER;
}

export function toInt(v: any, min: number, max: number): number {
  invariant(v !== null && v !== undefined);
  switch (typeof v) {
    case 'object':
      invariant(v instanceof Primitive);
      return toInt(v.val, min, max);
    case 'string':
      return toInt(parseInt(v, 10), min, max);
    case 'number':
      invariant(isSafeInteger(v) && v >= min && v < max);
      return v;
    default:
      throw new Error('unexpected primitive type');
  }
}

export type uint8 = number;
let maxUint8Value = Math.pow(2, 8) - 1;
export function toUint8(v: any): uint8 {
  return toInt(v, 0, maxUint8Value);
}
let uint8Type = makePrimitiveType(Kind.Uint8);
export class Uint8 extends Primitive<uint8> {
  constructor(val: Uint8Union) {
    super(toUint8(val), uint8Type);
  }
}
export type Uint8Union = Uint8 | uint8;

export type uint16 = number;
let maxUint16Value = Math.pow(2, 16) - 1;
export function toUint16(v: any): uint16 {
  return toInt(v, 0, maxUint16Value);
}
let uint16Type = makePrimitiveType(Kind.Uint16);
export class Uint16 extends Primitive<uint16> {
  constructor(val: Uint16Union) {
    super(toUint16(val), uint16Type);
  }
}
export type Uint16Union = Uint16 | uint16;

export type uint32 = number;
let maxUint32Value = Math.pow(2, 32) - 1;
export function toUint32(v: any): uint32 {
  return toInt(v, 0, maxUint32Value);
}
let uint32Type = makePrimitiveType(Kind.Uint32);
export class Uint32 extends Primitive<uint32> {
  constructor(val: Uint32Union) {
    super(toUint32(val), uint32Type);
  }
}
export type Uint32Union = Uint32 | uint32;

export type uint64 = number;
let maxUint64Value = Math.pow(2, 64) - 1;
export function toUint64(v: any): uint64 {
  return toInt(v, 0, maxUint64Value);
}
let uint64Type = makePrimitiveType(Kind.Uint64);
export class Uint64 extends Primitive<uint64> {
  constructor(val: Uint64Union) {
    super(toUint64(val), uint64Type);
  }
}
export type Uint64Union = Uint64 | uint64;

export type int8 = number;
let minInt8Value = -Math.pow(2, 7);
let maxInt8Value = Math.pow(2, 7) - 1;
export function toInt8(v: any): int8 {
  return toInt(v, minInt8Value, maxInt8Value);
}
let int8Type = makePrimitiveType(Kind.Int8);
export class Int8 extends Primitive<int8> {
  constructor(val: Int8Union) {
    super(toInt8(val), int8Type);
  }
}
export type Int8Union = Int8 | int8;

export type int16 = number;
let minInt16Value = -Math.pow(2, 15);
let maxInt16Value = Math.pow(2, 15) - 1;
export function toInt16(v: any): int16 {
  return toInt(v, minInt16Value, maxInt16Value);
}
let int16Type = makePrimitiveType(Kind.Int16);
export class Int16 extends Primitive<int16> {
  constructor(val: Int16Union) {
    super(toInt16(val), int16Type);
  }
}
export type Int16Union = Int16 | int16;

export type int32 = number;
let minInt32Value = -Math.pow(2, 31);
let maxInt32Value = Math.pow(2, 31) - 1;
export function toInt32(v: any): int32 {
  return toInt(v, minInt32Value, maxInt32Value);
}
let int32Type = makePrimitiveType(Kind.Int32);
export class Int32 extends Primitive<int32> {
  constructor(val: Int32Union) {
    super(toInt32(val), int32Type);
  }
}
export type Int32Union = Int32 | int32;

export type int64 = number;
let minInt64Value = -Math.pow(2, 63);
let maxInt64Value = Math.pow(2, 63) - 1;
export function toInt64(v: any): int64 {
  return toInt(v, minInt64Value, maxInt64Value);
}
let int64Type = makePrimitiveType(Kind.Int64);
export class Int64 extends Primitive<int64> {
  constructor(val: Int64Union) {
    super(toInt64(val), int64Type);
  }
}
export type Int64Union = Int64 | int64;

// TODO: What to do about float32 overflow?

export function toFloat(v: any): number {
  invariant(v !== null && v !== undefined);
  switch (typeof v) {
    case 'object':
      invariant(v instanceof Primitive);
      return toFloat(v.val);
    case 'string':
      return toFloat(parseFloat(v, 10));
    case 'number':
      return v;
    default:
      throw new Error('unexpected primitive type');
  }
}

export type float32 = number;
export function toFloat32(v: any): float32 {
  return toFloat(v);
}

export type float64 = number;
export function toFloat64(v: any): float64 {
  return toFloat(v);
}

let strType = makePrimitiveType(Kind.String);
export class Str extends Primitive<string> {
  constructor(val: StrUnion) {
    super(toStr(val), strType);
  }
}
export type StrUnion = Str | string;
export function toStr(v: any): string {
  invariant(v !== null && v !== undefined);
  switch (typeof v) {
    case 'object':
      invariant(v instanceof Primitive);
      return toStr(v.val);
    case 'string':
      return v;
    default:
      throw new Error('unexpected primitive type');
  }
}

let boolType = makePrimitiveType(Kind.Bool);
export class Bool extends Primitive<boolean> {
  constructor(val: BoolUnion) {
    super(toBool(val), boolType);
  }
}
export type BoolUnion = Bool | boolean;
export function toBool(v: any): boolean {
  invariant(v !== null && v !== undefined);
  switch (typeof v) {
    case 'object':
      invariant(v instanceof Primitive);
      return toBool(v.val);
    case 'boolean':
      return v;
    default:
      throw new Error('unexpected primitive type');
  }
}

export type primitive = uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | float32 | float64 | string | boolean;


