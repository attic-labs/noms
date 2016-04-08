// This file was generated by nomdl/codegen.
// @flow
/* eslint-disable */

import {
  Field as _Field,
  Package as _Package,
  Ref as _Ref,
  createStructClass as _createStructClass,
  float64Type as _float64Type,
  makeStructType as _makeStructType,
  makeType as _makeType,
  registerPackage as _registerPackage,
  stringType as _stringType,
} from '@attic/noms';
import type {
  Struct as _Struct,
  float64 as _float64,
} from '@attic/noms';

const _pkg = new _Package([
  _makeStructType('StructWithUnions',
    [
      new _Field('a', _makeType(new _Ref(), 1), false),
      new _Field('d', _makeType(new _Ref(), 2), false),
    ],
    [

    ]
  ),
  _makeStructType('',
    [

    ],
    [
      new _Field('b', _float64Type, false),
      new _Field('c', _stringType, false),
    ]
  ),
  _makeStructType('',
    [

    ],
    [
      new _Field('e', _float64Type, false),
      new _Field('f', _stringType, false),
    ]
  ),
], [
]);
_registerPackage(_pkg);
const StructWithUnions$type = _makeType(_pkg.ref, 0);
const StructWithUnions$typeDef = _makeStructType('StructWithUnions',
  [
    new _Field('a', _makeType(_pkg.ref, 1), false),
    new _Field('d', _makeType(_pkg.ref, 2), false),
  ],
  [

  ]
);
const __unionOfBOfFloat64AndCOfString$type = _makeType(_pkg.ref, 1);
const __unionOfBOfFloat64AndCOfString$typeDef = _makeStructType('',
  [

  ],
  [
    new _Field('b', _float64Type, false),
    new _Field('c', _stringType, false),
  ]
);
const __unionOfEOfFloat64AndFOfString$type = _makeType(_pkg.ref, 2);
const __unionOfEOfFloat64AndFOfString$typeDef = _makeStructType('',
  [

  ],
  [
    new _Field('e', _float64Type, false),
    new _Field('f', _stringType, false),
  ]
);


type StructWithUnions$Data = {
  a: __unionOfBOfFloat64AndCOfString;
  d: __unionOfEOfFloat64AndFOfString;
};

interface StructWithUnions$Interface extends _Struct {
  constructor(data: StructWithUnions$Data): void;
  a: __unionOfBOfFloat64AndCOfString;  // readonly
  setA(value: __unionOfBOfFloat64AndCOfString): StructWithUnions$Interface;
  d: __unionOfEOfFloat64AndFOfString;  // readonly
  setD(value: __unionOfEOfFloat64AndFOfString): StructWithUnions$Interface;
}

export const StructWithUnions: Class<StructWithUnions$Interface> = _createStructClass(StructWithUnions$type, StructWithUnions$typeDef);

type __unionOfBOfFloat64AndCOfString$Data = {
};

interface __unionOfBOfFloat64AndCOfString$Interface extends _Struct {
  constructor(data: __unionOfBOfFloat64AndCOfString$Data): void;
  b: ?_float64;  // readonly
  setB(value: _float64): __unionOfBOfFloat64AndCOfString$Interface;
  c: ?string;  // readonly
  setC(value: string): __unionOfBOfFloat64AndCOfString$Interface;
}

export const __unionOfBOfFloat64AndCOfString: Class<__unionOfBOfFloat64AndCOfString$Interface> = _createStructClass(__unionOfBOfFloat64AndCOfString$type, __unionOfBOfFloat64AndCOfString$typeDef);

type __unionOfEOfFloat64AndFOfString$Data = {
};

interface __unionOfEOfFloat64AndFOfString$Interface extends _Struct {
  constructor(data: __unionOfEOfFloat64AndFOfString$Data): void;
  e: ?_float64;  // readonly
  setE(value: _float64): __unionOfEOfFloat64AndFOfString$Interface;
  f: ?string;  // readonly
  setF(value: string): __unionOfEOfFloat64AndFOfString$Interface;
}

export const __unionOfEOfFloat64AndFOfString: Class<__unionOfEOfFloat64AndFOfString$Interface> = _createStructClass(__unionOfEOfFloat64AndFOfString$type, __unionOfEOfFloat64AndFOfString$typeDef);
