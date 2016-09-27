// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import Hash from './hash.js';
import Ref from './ref.js';
import Map from './map.js';
import Set from './set.js';
import type Value from './value.js';
import type {RootTracker} from './chunk-store.js';
import ValueStore from './value-store.js';
import type {BatchStore} from './batch-store.js';
import Dataset from './dataset.js';
import Commit from './commit.js';
import {equals} from './compare.js';

export default class Database {
  _vs: ValueStore;
  _rt: RootTracker;
  _datasets: Promise<Map<string, Ref<Commit<any>>>>;

  constructor(bs: BatchStore, cacheSize: number = 0) {
    this._vs = new ValueStore(bs, cacheSize);
    this._rt = bs;
    this._datasets = this._datasetsFromRootRef(bs.getRoot());
  }

  _datasetsFromRootRef(rootRef: Promise<Hash>): Promise<Map<string, Ref<Commit<any>>>> {
    return rootRef.then(rootRef => {
      if (rootRef.isEmpty()) {
        return Promise.resolve(new Map());
      }

      return this.readValue(rootRef);
    });
  }

  datasets(): Promise<Map<string, Ref<Commit<any>>>> {
    return this._datasets;
  }

  getDataset(id: string): Dataset {
    return new Dataset(this, id, this.datasets().then(sets => sets.get(id)));
  }

  // TODO: This should return Promise<Value | null>
  async readValue(hash: Hash): Promise<any> {
    return this._vs.readValue(hash);
  }

  writeValue<T: Value>(v: T): Ref<T> {
    return this._vs.writeValue(v);
  }

  async _descendsFrom(commit: Commit<any>, currentHeadRef: Ref<Commit<any>>): Promise<boolean> {
    let ancestors = commit.parents;
    while (!(await ancestors.has(currentHeadRef))) {
      if (ancestors.isEmpty()) {
        return false;
      }
      ancestors = await getAncestors(ancestors, this);
    }
    return true;
  }

  // Commit updates the commit that ds points at. If parents is provided then the promise
  // is rejected if the commit does not descend from the parents.
  async commit(ds: Dataset, v: Value, parents: ?Array<Ref<Commit<any>>> = undefined):
  Promise<Dataset> {
    if (!parents) {
      const headRef = await ds.headRef();
      parents = headRef ? [headRef] : [];
    }
    const commit = new Commit(v, new Set(parents));
    try {
      const commitRef = await this._doCommit(ds.id, commit);
      return new Dataset(this, ds.id, Promise.resolve(commitRef));
    } finally {
      this._datasets = this._datasetsFromRootRef(this._rt.getRoot());
    }
  }

  async _doCommit(datasetId: string, commit: Commit<any>): Ref<any> {
    const currentRootRefP = this._rt.getRoot();
    const datasetsP = this._datasetsFromRootRef(currentRootRefP);
    let currentDatasets = await (datasetsP:Promise<Map<any, any>>);
    const currentRootRef = await currentRootRefP;
    const commitRef = this.writeValue(commit);

    if (!currentRootRef.isEmpty()) {
      const currentHeadRef = await currentDatasets.get(datasetId);
      if (currentHeadRef) {
        if (equals(commitRef, currentHeadRef)) {
          // $FlowIssue: thinks commitRef is a Promise
          return commitRef;
        }
        if (!await this._descendsFrom(commit, currentHeadRef)) {
          throw new Error('Merge needed');
        }
      }
    }

    currentDatasets = await currentDatasets.set(datasetId, commitRef);
    const newRootRef = this.writeValue(currentDatasets).targetHash;
    if (await this._rt.updateRoot(newRootRef, currentRootRef)) {
      // $FlowIssue: thinks commitRef is a Promise
      return commitRef;
    }

    throw new Error('Optimistic lock failed');
  }

  close(): Promise<void> {
    return this._vs.close();
  }
}

async function getAncestors(commits: Set<Ref<Commit<any>>>, database: Database):
    Promise<Set<Ref<Commit<any>>>> {
  let ancestors = new Set();
  await commits.map(async (commitRef) => {
    const commit = await database.readValue(commitRef.targetHash);
    await commit.parents.map(async (ref) => ancestors = await ancestors.add(ref));
  });
  return ancestors;
}
