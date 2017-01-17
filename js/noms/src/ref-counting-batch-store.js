// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import type {BatchStore} from './batch-store.js';
import Chunk from './chunk.js';
import Hash from './hash.js';

export default class RefCountingBatchStore {
  _bs: BatchStore;
  _cb: () => any;
  _rc: number;

  /**
   * Ref counts `bs`, and calls `cb` when `bs` is actually closed (when the ref
   * count reaches 0).
   *
   * Ref count starts at 1, and can be increased by calling `addRef`, or
   * decremented by calling `close`. Once the ref count reaches 0, no methods
   * can be called
   */
  constructor(bs: BatchStore, cb: () => any) {
    this._bs = bs;
    this._cb = cb;
    this._rc = 1;
  }

  addRef(): void {
    this._assertIsOpen();
    this._rc++;
  }

  close(): Promise<void> {
    this._assertIsOpen();
    if (--this._rc === 0) {
      return this._bs.close().then(r => {
        this._cb();
        return r;
      });
    }
    return Promise.resolve(undefined);
  }

  _assertIsOpen() {
    if (this._rc <= 0) {
      throw new Error('already closed');
    }
  }

  get(hash: Hash) {
    this._assertIsOpen();
    return this._bs.get(hash);
  }
  schedulePut(c: Chunk, hints: Set<Hash>): void {
    this._assertIsOpen();
    return this._bs.schedulePut(c, hints);
  }
  flush(): Promise<void> {
    this._assertIsOpen();
    return this._bs.flush();
  }
  getRoot(): Promise<Hash> {
    this._assertIsOpen();
    return this._bs.getRoot();
  }
  updateRoot(current: Hash, last: Hash): Promise<boolean> {
    this._assertIsOpen();
    return this._bs.updateRoot(current, last);
  }
}
