// @flow

import Chunk from './chunk.js';
import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';
import Struct from './struct.js';
import {Package, registerPackage} from './package.js';
import {Kind} from './noms_kind.js';
import {Field, makeCompoundType, makePrimitiveType, makeStructType, makeType} from './type.js';
import type {valueOrPrimitive} from './value.js';
import {NomsMap, MapLeafSequence} from './map.js';
import {NomsSet, SetLeafSequence} from './map.js';
import Ref from './ref.js';
import {readValue} from './readValue.js';
import {writeValue} from './encode.js';

export class DataStore {
  _cs: ChunkStore;
  datasets: Promise<NomsMap<string, Ref>>;

  constructor(cs: ChunkStore) {
    this._cs = cs;
    this.datasets = this._datasetsFromRootRef();
  }

  async getRoot(): Promise<Ref> {
    return this._cs.getRoot();
  }

  async updateRoot(current: Ref, last: Ref): Promise<boolean> {
    return this._cs.updateRoot(current, last);
  }

  async get(ref: Ref): Promise<Chunk> {
    return this._cs.get(ref);
  }

  async has(ref: Ref): Promise<boolean> {
    return this._cs.has(ref);
  }

  put(c: Chunk) {
    this._cs.put(c);
  }

  close() {}

  _datasetsFromRootRef(): Promise<NomsMap<string, Ref>> {
    return this.getRoot().then(
      rootRef => rootRef ? readValue(rootRef, this._cs) : newEmptyCommitMap());
  }

  head(datasetID: string): Promise<?Commit> {
    return this.datasets.then(datasets => datasets.get(datasetID).then(
      commitRef => commitRef ? readValue(commitRef, this._cs) : null));
  }

  async commit(datasetID: string, commit: Commit): Promise<DataStore> {
    const currentRootRef = await this.getRoot();
    let currentDatasets: NomsMap<string, Ref> = await this.datasets;
    if (!currentRootRef.empty && !currentDatasets.ref.equal(currentRootRef)) {
      currentDatasets = await this._datasetsFromRootRef();
    }

    if (!currentRootRef.empty) {
      const currentHeadRef = await currentDatasets.get(datasetID);
      if (currentHeadRef) {
        if (commitRef.equals(currentHeadRef)) {
          return this;
        }

        if (await this.descendsFrom(commit, currentHeadRef)) {
          throw new MergeNeededError(this);
        }
      }
    }

    // TODO: This Commit will be orphaned if the UpdateRoot below fails
    const commitRef = writeValue(commit, commit.type, this);


  }
}

const datasPackage = new Package([makeStructType('S1', [
  new Field('value', makePrimitiveType(Kind.Value), false),
  new Field('parents',
              makeCompoundType(Kind.Set, makeCompoundType(Kind.Ref, makeType(new Ref(), 0))), true),
], [])], []);
const packageRef = datasPackage.ref;
const commitType = makeType(packageRef, 0);
const commitTypeDef = datasPackage.types[0];
const commitMapType = makeCompoundType(Kind.Map, makePrimitiveType(Kind.String),
                                       makeCompoundType(Kind.Ref, commitType));
const commitSetType = makeCompoundType(Kind.Set, makeCompoundType(Kind.Ref, commitType));
registerPackage(datasPackage);

function newEmptyCommitMap(cs: ChunkStore): NomsMap<string, Ref> {
  return new NomsMap(cs, commitMapType, new MapLeafSequence(commitMapType, []));
}

function newParentSet(cs: ChunkStore, commitRefs: Array<Ref>): NomsSet<Ref> {
  return new NomsSet(cs, commitSetType, new SetLeafSequence(commitSetType, commitRefs));
}

export class Commit extends Struct {
  constructor(cs: ChunkStore, parents: Array<Ref>, value: valueOrPrimitive) {
    super(commitType, commitTypeDef, {parents: newParentSet(cs, parents), value: value});
  }
}
