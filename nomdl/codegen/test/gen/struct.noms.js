// This file was generated by nomdl/codegen.
// @flow
/* eslint-disable */

import {
  Field as _Field,
  Package as _Package,
  boolType as _boolType,
  createStructClass as _createStructClass,
  makeListType as _makeListType,
  makeStructType as _makeStructType,
  makeType as _makeType,
  newList as _newList,
  registerPackage as _registerPackage,
  stringType as _stringType,
} from '@attic/noms';
import type {
  NomsList as _NomsList,
  Struct as _Struct,
} from '@attic/noms';

const _pkg = new _Package([
  _makeStructType('Struct',
    [
      new _Field('s', _stringType, false),
      new _Field('b', _boolType, false),
    ],
    [

    ]
  ),
], [
]);
_registerPackage(_pkg);
const Struct$type = _makeType(_pkg.ref, 0);
const Struct$typeDef = _makeStructType('Struct',
  [
    new _Field('s', _stringType, false),
    new _Field('b', _boolType, false),
  ],
  [

  ]
);


type Struct$Data = {
  s: string;
  b: boolean;
};

interface Struct$Interface extends _Struct {
  constructor(data: Struct$Data): void;
  s: string;  // readonly
  setS(value: string): Struct$Interface;
  b: boolean;  // readonly
  setB(value: boolean): Struct$Interface;
}

export const Struct: Class<Struct$Interface> = _createStructClass(Struct$type, Struct$typeDef);

export function newListOfStruct(values: Array<Struct>): Promise<_NomsList<Struct>> {
  return _newList(values, _makeListType(_makeType(_pkg.ref, 0)));
}
