// This file was generated by nomdl/codegen.
// @flow
// eslint-disable max-len

import {
  Field as _Field,
  Package as _Package,
  blobType as _blobType,
  boolType as _boolType,
  float32Type as _float32Type,
  float64Type as _float64Type,
  int16Type as _int16Type,
  int32Type as _int32Type,
  int64Type as _int64Type,
  int8Type as _int8Type,
  makeStructType as _makeStructType,
  registerPackage as _registerPackage,
  stringType as _stringType,
  uint16Type as _uint16Type,
  uint32Type as _uint32Type,
  uint64Type as _uint64Type,
  uint8Type as _uint8Type,
  valueType as _valueType,
} from "@attic/noms";
import type {
  Blob as _Blob,
  Struct as _Struct,
  Value as _Value,
  float32 as _float32,
  float64 as _float64,
  int16 as _int16,
  int32 as _int32,
  int64 as _int64,
  int8 as _int8,
  uint16 as _uint16,
  uint32 as _uint32,
  uint64 as _uint64,
  uint8 as _uint8,
} from "@attic/noms";

{
  const pkg = new _Package([
    _makeStructType('StructPrimitives',
      [
        new _Field('uint64', _uint64Type, false),
        new _Field('uint32', _uint32Type, false),
        new _Field('uint16', _uint16Type, false),
        new _Field('uint8', _uint8Type, false),
        new _Field('int64', _int64Type, false),
        new _Field('int32', _int32Type, false),
        new _Field('int16', _int16Type, false),
        new _Field('int8', _int8Type, false),
        new _Field('float64', _float64Type, false),
        new _Field('float32', _float32Type, false),
        new _Field('bool', _boolType, false),
        new _Field('string', _stringType, false),
        new _Field('blob', _blobType, false),
        new _Field('value', _valueType, false),
      ],
      [

      ]
    ),
  ], [
  ]);
  _registerPackage(pkg);
}


export interface StructPrimitives extends _Struct {
  uint64: _uint64;  // readonly
  setUint64(value: _uint64): StructPrimitives;
  uint32: _uint32;  // readonly
  setUint32(value: _uint32): StructPrimitives;
  uint16: _uint16;  // readonly
  setUint16(value: _uint16): StructPrimitives;
  uint8: _uint8;  // readonly
  setUint8(value: _uint8): StructPrimitives;
  int64: _int64;  // readonly
  setInt64(value: _int64): StructPrimitives;
  int32: _int32;  // readonly
  setInt32(value: _int32): StructPrimitives;
  int16: _int16;  // readonly
  setInt16(value: _int16): StructPrimitives;
  int8: _int8;  // readonly
  setInt8(value: _int8): StructPrimitives;
  float64: _float64;  // readonly
  setFloat64(value: _float64): StructPrimitives;
  float32: _float32;  // readonly
  setFloat32(value: _float32): StructPrimitives;
  bool: boolean;  // readonly
  setBool(value: boolean): StructPrimitives;
  string: string;  // readonly
  setString(value: string): StructPrimitives;
  blob: _Blob;  // readonly
  setBlob(value: _Blob): StructPrimitives;
  value: _Value;  // readonly
  setValue(value: _Value): StructPrimitives;
}
