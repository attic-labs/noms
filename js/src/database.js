// @flow

import Ref from './ref.js';
import RefValue from './ref-value.js';
import Map from './map.js';
import Set from './set.js';
import type {valueOrPrimitive} from './value.js';
import type {RootTracker} from './chunk-store.js';
import ValueStore from './value-store.js';
import BatchStore from './batch-store.js';
import {
  makeRefType,
  makeStructType,
  makeSetType,
  makeMapType,
  Type,
  stringType,
  valueType,
} from './type.js';
import Commit from './commit.js';
import {equals} from './compare.js';

type DatasTypes = {
  commitType: Type,
  commitSetType: Type,
  refOfCommitType: Type,
  commitMapType: Type,
};

let datasTypes: DatasTypes;
export function getDatasTypes(): DatasTypes {
  if (!datasTypes) {
    // struct Commit {
    //   value: Value
    //   parents: Set<Ref<Commit>>
    // }
    const commitType = makeStructType('Commit', {
      'value': valueType,
      'parents': valueType, // placeholder
    });
    const refOfCommitType = makeRefType(commitType);
    const commitSetType = makeSetType(refOfCommitType);
    commitType.desc.fields['parents'] = commitSetType;
    const commitMapType = makeMapType(stringType, refOfCommitType);
    datasTypes = {
      commitType,
      refOfCommitType,
      commitSetType,
      commitMapType,
    };
  }

  return datasTypes;
}

export default class Database {
  _vs: ValueStore;
  _rt: RootTracker;
  _datasets: Promise<Map<string, RefValue<Commit>>>;

  constructor(bs: BatchStore, cacheSize: number = 0) {
    this._vs = new ValueStore(bs, cacheSize);
    this._rt = bs;
    this._datasets = this._datasetsFromRootRef(bs.getRoot());
  }

  _clone(vs: ValueStore, rt: RootTracker): Database {
    const ds = Object.create(Database.prototype);
    ds._vs = vs;
    ds._rt = rt;
    ds._datasets = this._datasetsFromRootRef(rt.getRoot());
    return ds;
  }

  _datasetsFromRootRef(rootRef: Promise<Ref>): Promise<Map<string, RefValue<Commit>>> {
    return rootRef.then(rootRef => {
      if (rootRef.isEmpty()) {
        return Promise.resolve(new Map());
      }

      return this.readValue(rootRef);
    });
  }

  headRef(datasetID: string): Promise<?RefValue<Commit>> {
    return this._datasets.then(datasets => datasets.get(datasetID));
  }

  head(datasetID: string): Promise<?Commit> {
    return this.headRef(datasetID).then(hr => hr ? this.readValue(hr.targetRef) : null);
  }

  datasets(): Promise<Map<string, RefValue<Commit>>> {
    return this._datasets;
  }

  // TODO: This should return Promise<?valueOrPrimitive>
  async readValue(ref: Ref): Promise<any> {
    return this._vs.readValue(ref);
  }


  writeValue<T: valueOrPrimitive>(v: T): RefValue<T> {
    return this._vs.writeValue(v);
  }

  async _descendsFrom(commit: Commit, currentHeadRef: RefValue<Commit>): Promise<boolean> {
    let ancestors = commit.parents;
    while (!(await ancestors.has(currentHeadRef))) {
      if (ancestors.isEmpty()) {
        return false;
      }
      ancestors = await getAncestors(ancestors, this);
    }
    return true;
  }

  async commit(datasetId: string, commit: Commit): Promise<Database> {
    const currentRootRefP = this._rt.getRoot();
    const datasetsP = this._datasetsFromRootRef(currentRootRefP);
    let currentDatasets = await (datasetsP:Promise<Map>);
    const currentRootRef = await currentRootRefP;
    const commitRef = this.writeValue(commit);

    if (!currentRootRef.isEmpty()) {
      const currentHeadRef = await currentDatasets.get(datasetId);
      if (currentHeadRef) {
        if (equals(commitRef, currentHeadRef)) {
          return this;
        }
        if (!await this._descendsFrom(commit, currentHeadRef)) {
          throw new Error('Merge needed');
        }
      }
    }

    currentDatasets = await currentDatasets.set(datasetId, commitRef);
    const newRootRef = this.writeValue(currentDatasets).targetRef;
    if (await this._rt.updateRoot(newRootRef, currentRootRef)) {
      return this._clone(this._vs, this._rt);
    }

    throw new Error('Optimistic lock failed');
  }

  close(): Promise<void> {
    return this._vs.close();
  }
}

async function getAncestors(commits: Set<RefValue<Commit>>, store: Database):
    Promise<Set<RefValue<Commit>>> {
  let ancestors = new Set();
  await commits.map(async (commitRef) => {
    const commit = await store.readValue(commitRef.targetRef);
    await commit.parents.map(async (ref) => ancestors = await ancestors.insert(ref));
  });
  return ancestors;
}
