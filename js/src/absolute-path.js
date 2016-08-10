// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import {invariant, notNull} from './assert.js';
import {datasetRe} from './dataset.js';
import Database from './database.js';
import {default as Hash, stringLength} from './hash.js';
import Path from './path.js';
import type Value from './value.js';

const datasetCapturePrefixRe = new RegExp('^(' + datasetRe.source + ')');

/**
 * An AbsolutePath is a Path relative to either a dataset head, or a hash.
 */
export default class AbsolutePath {
  /** The dataset ID that `_path` is in, or `''` if none. */
  dataset: string;

  /** The hash the that `_path` is in, if any. */
  hash: Hash | null;

  /** Path relative to either `_dataset` or `_hash`. */
  path: Path;

  /**
   * Returns `str` parsed as an AbsolutePath if successful, or a null path with an error message
   * if not.
   */
  static parse(str: string): [AbsolutePath | null, string] {
    if (str === '') {
      return [null, 'Empty path'];
    }

    let dataset = '';
    let hash = null;
    let pathStr = '';

    if (str[0] === '#') {
      const tail = str.slice(1);
      if (tail.length < stringLength) {
        return [null, `Invalid hash: ${tail}`];
      }

      const hashStr = tail.slice(0, stringLength);
      hash = Hash.parse(hashStr);
      if (hash === null) {
        return [null, `Invalid hash: ${hashStr}`];
      }

      pathStr = tail.slice(stringLength);
    } else {
      const parts = datasetCapturePrefixRe.exec(str);
      if (!parts) {
        return [null, `Invalid dataset name: ${str}`];
      }

      invariant(parts.length === 2);
      dataset = parts[1];
      pathStr = str.slice(parts[0].length);
    }

    if (pathStr.length === 0) {
      return [new AbsolutePath(dataset, hash, new Path()), ''];
    }

    const [path, err] = Path.parse(pathStr);
    if (err !== '') {
      return [null, err];
    }

    return [new AbsolutePath(dataset, hash, notNull(path)), ''];
  }

  constructor(dataset: string, hash: Hash | null, path: Path) {
    this.dataset = dataset;
    this.hash = hash;
    this.path = path;
  }

  async resolve(db: Database): Promise<Value | null> {
    let val = null;
    if (this.dataset !== '') {
      val = await db.head(this.dataset);
    } else if (this.hash !== null) {
      val = await db.readValue(this.hash);
    } else {
      throw new Error('unreachable');
    }

    if (val === undefined) {
      val = null;
    }

    if (val !== null) {
      return this.path.resolve(val);
    } else {
      return null;
    }
  }

  toString(): string {
    if (this.dataset !== '') {
      return this.dataset + this.path.toString();
    }
    if (this.hash !== null) {
      return '#' + this.hash.toString() + this.path.toString();
    }
    throw new Error('unreachable');
  }
}
