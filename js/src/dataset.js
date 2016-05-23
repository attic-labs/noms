// @flow

import Commit from './commit.js';
import type Value from './value.js';
import type Database from './database.js';
import Ref from './ref.js';
import Set from './set.js';

const idRe = /^[a-zA-Z0-9\-_/]+$/;

export default class Dataset {
  _store: Database;
  _id: string;

  constructor(store: Database, id: string) {
    if (!idRe.test(id)) {
      throw new TypeError(`Invalid dataset ID: ${id}`);
    }
    this._store = store;
    this._id = id;
  }

  get store(): Database {
    return this._store;
  }

  get id(): string {
    return this._id;
  }

  headRef(): Promise<?Ref<Commit>> {
    return this._store.headRef(this._id);
  }

  head(): Promise<?Commit> {
    return this._store.head(this._id);
  }

  // Commit updates the commit that a dataset points at. If parents is provided then an the promise
  // is rejected if the commit does not descend from the parents.
  async commit(v: Value,
               parents: ?Array<Ref<Commit>> = undefined): Promise<Dataset> {
    if (!parents) {
      const headRef = await this.headRef();
      parents = headRef ? [headRef] : [];
    }
    const commit = new Commit(v, new Set(parents));
    const store = await this._store.commit(this._id, commit);
    return new Dataset(store, this._id);
  }
}
