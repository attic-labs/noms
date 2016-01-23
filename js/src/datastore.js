// @flow

import Chunk from './chunk.js';
import Ref from './ref.js';
import type {ChunkStore} from './chunk_store.js';

export default class DataStore {
  _cs: ChunkStore;

  constructor(cs: ChunkStore) {
    this._cs = cs;
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
}
