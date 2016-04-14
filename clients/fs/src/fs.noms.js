// This file was generated by nomdl/codegen.
// @flow
/* eslint-disable */

import {
  Field as _Field,
  Kind as _Kind,
  Package as _Package,
  blobType as _blobType,
  createStructClass as _createStructClass,
  emptyRef as _emptyRef,
  makeCompoundType as _makeCompoundType,
  makeMapType as _makeMapType,
  makeStructType as _makeStructType,
  makeType as _makeType,
  newMap as _newMap,
  registerPackage as _registerPackage,
  stringType as _stringType,
} from '@attic/noms';
import type {
  Blob as _Blob,
  NomsMap as _NomsMap,
  RefValue as _RefValue,
  Struct as _Struct,
} from '@attic/noms';

const _pkg = new _Package([
  _makeStructType('Directory',
    [
      new _Field('entries', _makeCompoundType(_Kind.Map, _stringType, _makeCompoundType(_Kind.Ref, _makeType(_emptyRef, 2))), false),
    ],
    [

    ]
  ),
  _makeStructType('File',
    [
      new _Field('content', _makeCompoundType(_Kind.Ref, _blobType), false),
    ],
    [

    ]
  ),
  _makeStructType('DirectoryEntry',
    [

    ],
    [
      new _Field('file', _makeType(_emptyRef, 1), false),
      new _Field('directory', _makeType(_emptyRef, 0), false),
    ]
  ),
], [
]);
_registerPackage(_pkg);
const Directory$type = _makeType(_pkg.ref, 0);
const Directory$typeDef = _pkg.types[0];
const File$type = _makeType(_pkg.ref, 1);
const File$typeDef = _pkg.types[1];
const DirectoryEntry$type = _makeType(_pkg.ref, 2);
const DirectoryEntry$typeDef = _pkg.types[2];


type Directory$Data = {
  entries: _NomsMap<string, _RefValue<DirectoryEntry>>;
};

interface Directory$Interface extends _Struct {
  constructor(data: Directory$Data): void;
  entries: _NomsMap<string, _RefValue<DirectoryEntry>>;  // readonly
  setEntries(value: _NomsMap<string, _RefValue<DirectoryEntry>>): Directory$Interface;
}

export const Directory: Class<Directory$Interface> = _createStructClass(Directory$type, Directory$typeDef);

type File$Data = {
  content: _RefValue<_Blob>;
};

interface File$Interface extends _Struct {
  constructor(data: File$Data): void;
  content: _RefValue<_Blob>;  // readonly
  setContent(value: _RefValue<_Blob>): File$Interface;
}

export const File: Class<File$Interface> = _createStructClass(File$type, File$typeDef);

type DirectoryEntry$Data = {
};

interface DirectoryEntry$Interface extends _Struct {
  constructor(data: DirectoryEntry$Data): void;
  file: ?File;  // readonly
  setFile(value: File): DirectoryEntry$Interface;
  directory: ?Directory;  // readonly
  setDirectory(value: Directory): DirectoryEntry$Interface;
}

export const DirectoryEntry: Class<DirectoryEntry$Interface> = _createStructClass(DirectoryEntry$type, DirectoryEntry$typeDef);

export function newMapOfStringToRefOfDirectoryEntry(values: Array<any>): Promise<_NomsMap<string, _RefValue<DirectoryEntry>>> {
  return _newMap(values, _makeMapType(_stringType, _makeCompoundType(_Kind.Ref, _makeType(_pkg.ref, 2))));
}
