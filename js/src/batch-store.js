// @flow

import Chunk from './chunk.js';
import Ref from './ref.js';
import OrderedPutCache from './put-cache.js';
import type {ChunkStream} from './chunk-serializer.js';
import {notNull} from './assert.js';

type PendingReadMap = { [key: string]: Promise<Chunk> };
export type UnsentReadMap = { [key: string]: (c: Chunk) => void };

export type WriteRequest = {
  hash: Ref;
  hints: Set<Ref>;
}

interface Delegate {
  readBatch(reqs: UnsentReadMap): Promise<void>;
  writeBatch(hints: Set<Ref>, chunkStream: ChunkStream): Promise<void>;
  getRoot(): Promise<Ref>;
  updateRoot(current: Ref, last: Ref): Promise<boolean>;
}

export default class BatchStore {
  _pendingReads: PendingReadMap;
  _unsentReads: ?UnsentReadMap;
  _readScheduled: boolean;
  _activeReads: number;
  _maxReads: number;

  _pendingWrites: OrderedPutCache;
  _unsentWrites: ?Array<WriteRequest>;
  _delegate: Delegate;

  constructor(maxReads: number, delegate: Delegate) {
    this._pendingReads = Object.create(null);
    this._unsentReads = null;
    this._readScheduled = false;
    this._activeReads = 0;
    this._maxReads = maxReads;

    this._pendingWrites = new OrderedPutCache();
    this._unsentWrites = null;
    this._delegate = delegate;
  }

  get(ref: Ref): Promise<Chunk> {
    const refStr = ref.toString();
    let p = this._pendingReads[refStr];
    if (p) {
      return p;
    }
    p = this._pendingWrites.get(refStr);
    if (p) {
      return p;
    }

    return this._pendingReads[refStr] = new Promise(resolve => {
      if (!this._unsentReads) {
        this._unsentReads = Object.create(null);
      }

      notNull(this._unsentReads)[refStr] = resolve;
      this._maybeStartRead();
    });
  }

  _maybeStartRead() {
    if (!this._readScheduled && this._unsentReads && this._activeReads < this._maxReads) {
      this._readScheduled = true;
      setTimeout(() => {
        this._read();
      }, 0);
    }
  }

  async _read(): Promise<void> {
    this._activeReads++;

    const reqs = notNull(this._unsentReads);
    this._unsentReads = null;
    this._readScheduled = false;

    await this._delegate.readBatch(reqs);

    const self = this; // TODO: Remove this when babel bug is fixed.
    Object.keys(reqs).forEach(refStr => {
      delete self._pendingReads[refStr];
    });

    this._activeReads--;
    this._maybeStartRead();
  }

  schedulePut(c: Chunk, hints: Set<Ref>): void {
    if (!this._pendingWrites.append(c)) {
      return; // Already in flight.
    }

    if (!this._unsentWrites) {
      this._unsentWrites = [];
    }
    this._unsentWrites.push({hash: c.ref, hints: hints});
  }

  async flush(): Promise<void> {
    if (!this._unsentWrites) {
      return;
    }

    const reqs = notNull(this._unsentWrites);
    this._unsentWrites = null;

    const first = reqs[0].hash;
    let last = first;
    let hints = new Set();
    for (const req of reqs) {
      req.hints.forEach(hint => { hints = hints.add(hint); });
      last = req.hash;
    }
    // TODO: Deal with backpressure
    const chunkStream = await this._pendingWrites.extractChunks(first.toString(), last.toString());
    await this._delegate.writeBatch(hints, chunkStream);

    return this._pendingWrites.dropUntil(last.toString());
  }

  async getRoot(): Promise<Ref> {
    return this._delegate.getRoot();
  }

  async updateRoot(current: Ref, last: Ref): Promise<boolean> {
    await this.flush();
    if (current.equals(last)) {
      return true;
    }

    return this._delegate.updateRoot(current, last);
  }

  // TODO: Should close() call flush() and block until it's done? Maybe closing with outstanding
  // requests should be an error on both sides. TBD.
  close() {}
}
