// This file was generated by nomdl/codegen.
// @flow
/* eslint-disable */

import {
  int64Type as _int64Type,
  makeListType as _makeListType,
  newList as _newList,
} from '@attic/noms';
import type {
  NomsList as _NomsList,
  int64 as _int64,
} from '@attic/noms';


export function newListOfInt64(values: Array<_int64>): Promise<_NomsList<_int64>> {
  return _newList(values, _makeListType(_int64Type));
}
