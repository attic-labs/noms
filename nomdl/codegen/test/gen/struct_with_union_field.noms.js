// This file was generated by nomdl/codegen.
// @flow
// eslint-disable max-len

import {
  Field as _Field,
  Kind as _Kind,
  Package as _Package,
  blobType as _blobType,
  float32Type as _float32Type,
  float64Type as _float64Type,
  makeCompoundType as _makeCompoundType,
  makeStructType as _makeStructType,
  registerPackage as _registerPackage,
  stringType as _stringType,
  uint8Type as _uint8Type,
  valueType as _valueType,
} from "@attic/noms";
import type {
  Blob as _Blob,
  NomsSet as _NomsSet,
  Struct as _Struct,
  Value as _Value,
  float32 as _float32,
  float64 as _float64,
  uint8 as _uint8,
} from "@attic/noms";

{
  const pkg = new _Package([
    _makeStructType('StructWithUnionField',
      [
        new _Field('a', _float32Type, false),
      ],
      [
        new _Field('b', _float64Type, false),
        new _Field('c', _stringType, false),
        new _Field('d', _blobType, false),
        new _Field('e', _valueType, false),
        new _Field('f', _makeCompoundType(_Kind.Set, _uint8Type), false),
      ]
    ),
  ], [
  ]);
  _registerPackage(pkg);
}


export interface StructWithUnionField extends _Struct {
  a: _float32;  // readonly
  setA(value: _float32): StructWithUnionField;
  b: ?_float64;  // readonly
  setB(value: _float64): StructWithUnionField;
  c: ?string;  // readonly
  setC(value: string): StructWithUnionField;
  d: ?_Blob;  // readonly
  setD(value: _Blob): StructWithUnionField;
  e: ?_Value;  // readonly
  setE(value: _Value): StructWithUnionField;
  f: ?_NomsSet<_uint8>;  // readonly
  setF(value: _NomsSet<_uint8>): StructWithUnionField;
}
