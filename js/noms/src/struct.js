// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import assertSubtype from './assert-type.js';
import type Ref from './ref.js';
import type {Type, StructDesc, Field} from './type.js';
import type Value from './value.js';
import {Kind} from './noms-kind.js';
import {ValueBase, init as initValue} from './value.js';
import {equals} from './compare.js';
import {getTypeOfValue, makeStructType, findFieldIndex} from './type.js';
import {invariant} from './assert.js';
import {isPrimitive} from './primitives.js';
import * as Bytes from './bytes.js';
import {isSubtype} from './assert-type.js';
import walk from './walk.js';
import type {WalkCallback} from './walk.js';
import type {ValueReader} from './value-store.js';

type StructData = {[key: string]: Value};

/** Matches the first valid field name in a string. */
export const fieldNameComponentRe = /^[a-zA-Z][a-zA-Z0-9_]*/;

/** Matches if an entire string is a valid field name. */
export const fieldNameRe = new RegExp(fieldNameComponentRe.source + '$');

/**
 * Base class for all Noms structs. The decoder creates sub classes of this for Noms struct.
 * These have the form of:
 *
 * ```noms
 * struct MyStruct {
 *   x: Number
 *   s: string
 * }
 * ```
 *
 * ```js
 * interface MyStruct extends Struct {
 *   get x(): number;
 *   setX(value: number): MyStruct;
 *   get s(): string;
 *   setS(value: string): MyStruct;
 * }
 * ```
 *
 * With one major exception: if the field name conflicts with any of the properties in ValueBase (or
 * Object), such as `chunks`, `hash` or `type` (or `toString`, `hasOwnProperty` etc.), then these
 * are not reflected directly on the struct instance.
 *
 * To reflect over structs you can create a new `StructMirror`. This is also the only way to get the
 * value of fields that conflict with `ValueBase` (`chunks`, `hash` and `type`).
 */
export default class Struct extends ValueBase {
  _type: Type<any>;
  _values: Value[];

  constructor(type: Type<StructDesc>, values: Value[]) {
    super();
    invariant(type.kind === Kind.Struct);
    init(this, type, values);
  }

  walkValues(vr: ValueReader, cb: WalkCallback): Promise<void> {
    return Promise.all(this._values.map(v => walk(v, vr, cb))).then();
  }

  get type(): Type<any> {
    return this._type;
  }

  get chunks(): Array<Ref<any>> {
    const mirror = new StructMirror(this);
    const chunks = [];

    const add = field => {
      const {value} = field;
      if (!isPrimitive(value)) {
        invariant(value instanceof ValueBase);
        chunks.push(...value.chunks);
      }
    };

    mirror.forEachField(add);
    return chunks;
  }
}

function validate(type: Type<any>, values: Value[]): void {
  let i = 0;
  type.desc.forEachField((name: string, type: Type<any>) => {
    const value = values[i];
    assertSubtype(type, value);
    i++;
  });
}

/**
 * StructFieldMirror represents a field in a struct and it used by StructMirror.
 */
export class StructFieldMirror {
  value: Value;
  name: string;
  type: Type<any>;

  constructor(value: Value, name: string, type: Type<any>) {
    this.value = value;
    this.name = name;
    this.type = type;
  }
}

/**
 * A StructMirror allows reflection of a Noms struct.
 * This allows you to get, set and remove a field by its name. Normally a Noms Struct will have
 * properties `foo` and method `setFoo(v)` to get and set a struct field but if the field name
 * conflicts with one of the properties provided by ValueBase then the only way to get and set them
 * is by using a StructMirror.
 */
export class StructMirror<T: Struct> {
  _values: Value[];
  _s: T;

  constructor(s: T) {
    this._s = s;
    this._values = s._values;
  }

  /**
   * The type of the struct this mirror is representing.
   */
  get type(): Type<StructDesc> {
    return this._s.type;
  }

  /**
   * The StructDesc describing the struct type.
   */
  get desc(): StructDesc {
    return this.type.desc;
  }

  /**
   * Iterates over all the fields in the struct and calls `cb`.
   */
  forEachField(cb: (f: StructFieldMirror) => void) {
    this.desc.fields.forEach((f, i) => {
      cb(new StructFieldMirror(this._values[i], f.name, f.type));
    });
  }

  /**
   * The name of the struct type.
   */
  get name(): string {
    return this.desc.name;
  }

  /**
   * Gets the value of a field in the struct. If the struct does not a have a field with the name
   * `name` then this returns `undefined`.
   */
  get(name: string): ?Value {
    const i = findFieldIndex(name, this.desc.fields);
    return i !== -1 ? this._values[i] : undefined;
  }

  /**
   * Whether the struct has a field with the name `name`.
   */
  has(name: string): boolean {
    return findFieldIndex(name, this.desc.fields) !== -1;
  }

  /**
   * Returns a new struct where the field `name` has been set to `value`. If `name` is not an
   * existing field in the struct or the type of `value` is different from the old value of the
   * struct field a new struct type is created.
   */
  set(name: string, value: Value): Struct {
    const fields = this.desc.fields;
    const i = findFieldIndex(name, fields);
    if (i === -1 || !isSubtype(fields[i].type, getTypeOfValue(value))) {
      // New/change field
      const data = Object.create(null);
      for (let i = 0; i < fields.length; i++) {
        data[fields[i].name] = this._values[i];
      }
      data[name] = value;
      return newStruct(this.name, data);
    }

    const newValues = this._values.concat();  // shallow clone
    newValues[i] = value;
    return newStructWithType(this.type, newValues);
  }

  /**
   * Returns a new struct where the field `name` has been removed.
   * If `name` is not an existing field in the struct then the current struct is returned.
   */
  delete(name: string): Struct {
    const oldFields = this.desc.fields;
    const idx = findFieldIndex(name, oldFields);
    if (idx === -1) {
      return this._s;
    }

    // New/change field
    const values = this._values.concat();  // clone
    values.splice(idx, 1);
    const type = removeFieldFromType(this.type, idx);
    return newStructWithValues(type, values);
  }
}

function removeFieldFromType(type: Type<StructDesc>, idx: number): Type<StructDesc> {
  const {desc} = type;
  const fieldMap = Object.create(null);
  let i = 0;
  desc.forEachField((n, t) => {
    if (i++ !== idx) {
      fieldMap[n] = t;
    }
  });
  return makeStructType(desc.name, fieldMap);
}

const cache: {[key: string]: Class<any>} = Object.create(null);

function makeName(prefix, name) {
  return `${prefix}${name[0].toUpperCase()}${name.slice(1)}`;
}

/**
 * Creates a class (function) that can be used to create new instances of the class.
 */
export function createStructClass<T: Struct>(type: Type<StructDesc>): Class<T> {
  const k = type.hash.toString();
  if (cache[k]) {
    return cache[k];
  }

  const c: any = class extends Struct {
    constructor(data: StructData) {
      const {fields} = type.desc;
      const values = new Array(fields.length);
      for (let i = 0; i < fields.length; i++) {
        values[i] = data[fields[i].name];
      }

      validate(type, values);
      super(type, values);
    }
  };

  type.desc.fields.forEach((f: Field, i: number) => {
    if (f.name in Struct.prototype) {  // Don't shadow things in {Struct, Object}.prototype.
      return;
    }
    Object.defineProperty(c.prototype, f.name, {
      configurable: true,
      enumerable: false,
      get: function() {
        return this._values[i];
      },
    });
    Object.defineProperty(c.prototype, makeName('set', f.name), {
      configurable: true,
      enumerable: false,
      value: getSetter(i),
      writable: true,
    });
    Object.defineProperty(c.prototype, makeName('delete', f.name), {
      configurable: true,
      enumerable: false,
      value: getDeleter(i),
      writable: true,
    });
  });

  return cache[k] = c;
}

function getSetter(i: number) {
  return function(value) {
    const values = this._values.concat();  // clone
    values[i] = value;
    return newStructWithType(this.type, values);
  };
}

function getDeleter(i: number) {
  return function() {
    const values = this._values.concat();  // clone
    values.splice(i, 1);
    const type = removeFieldFromType(this.type, i);
    return newStructWithType(type, values);
  };
}

/**
 * Creates a new instance of a struct, computing the type based on the `name` and `data`.
 */
export function newStruct<T: Struct>(name: string, data: StructData): T {
  const type = computeTypeForStruct(name, data);
  // Skip validation since there is no way the type and data can mismatch.
  return new (createStructClass(type))(data);
}

/**
 * Creates a new instance of a struct with a predetermined type. The `values` must come in the right
 * order (same order as the field names which are always in alphabetic order) and have the correct
 * type.
 */
export function newStructWithType<T: Struct>(type: Type<StructDesc>, values: Value[]): T {
  validate(type, values);
  return newStructWithValues(type, values);
}

function init<T: Struct>(s: T, type: Type<any>, values: Value[]) {
  s._type = type;
  s._values = values;
}

/**
 * Creates a new instance of a struct with a predetermined type. The `values` must come in the right
 * order (same order as the field names which are always in alphabetic order). This function does
 * not type check its values and should be used with care.
 */
export function newStructWithValues<T: Struct>(type: Type<any>, values: Value[]): T {
  const c = createStructClass(type);
  const s = Object.create(c.prototype);
  invariant(s instanceof c);
  initValue(s);
  init(s, type, values);
  return s;
}

function computeTypeForStruct(name: string, data: StructData): Type<StructDesc> {
  const fields = {};
  for (const k in data) {
    fields[k] = getTypeOfValue(data[k]);
  }
  return makeStructType(name, fields);
}

/**
 * Computes the diff between two structs of the same struct type. If the types are not equal an
 * exception is thrown.
 * Returns the field names which have different values in the respective structs.
 */
export function structDiff(s1: Struct, s2: Struct): string[] {
  invariant(equals(s1.type, s2.type));
  const desc1: StructDesc = s1.type.desc;

  const changed = [];
  desc1.fields.forEach((f: Field, i: number) => {
    const v1 = s1._values[i];
    const v2 = s2._values[i];
    if (!equals(v1, v2)) {
      changed.push(f.name);
    }
  });

  return changed;
}

const escapeChar = 'Q';
const escapedHeadRe = /[a-zA-PR-Z]/;
const escapedTailRe = /[a-zA-PR-Z0-9_]/;
const escapedCompleteRe = new RegExp('^' + escapedHeadRe.source + escapedTailRe.source + '*$');

/**
 * Escapes names for use as noms structs. Disallow characters are encoded as
 * 'Q<hex-encoded-utf8-bytes>'. Note that Q itself is also escaped since it is
 * the escape character.
 */
export function escapeStructField(input: string): string {
  if (escapedCompleteRe.test(input)) {
    return input;
  }

  if (input.length === 0) {
    throw new Error('cannot escape empty field name');
  }

  const encode = (c: string, p: RegExp) => {
    if (p.test(c) && p !== escapeChar) {
      return c;
    }

    let out = escapeChar;
    Bytes.fromString(c).forEach(b => {
      const hex = b.toString(16).toUpperCase();
      if (hex.length === 1) {
        out += '0';
      }
      out += hex;
    });
    return out;
  };

  let output = '';
  let pattern = escapedHeadRe;
  for (const c of input) {
    output += encode(c, pattern);
    pattern = escapedTailRe;
  }

  return output;
}
