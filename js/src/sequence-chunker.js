// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import type Sequence from './sequence.js'; // eslint-disable-line no-unused-vars
import type {MetaSequence, OrderedKey} from './meta-sequence.js';
import {MetaTuple} from './meta-sequence.js';
import type {SequenceCursor} from './sequence.js';
import type {ValueReader, ValueWriter} from './value-store.js';
import {invariant, notNull} from './assert.js';
import type Collection from './collection.js';

import Ref from './ref.js';

export type BoundaryChecker<T> = {
  write: (item: T) => boolean;
  windowSize: number;
};

export type NewBoundaryCheckerFn = () => BoundaryChecker<MetaTuple>;

export type makeChunkFn<T, S: Sequence> = (items: Array<T>) => [Collection<S>, OrderedKey, number];

export async function chunkSequence<T, S: Sequence<T>>(
    cursor: SequenceCursor,
    insert: Array<T>,
    remove: number,
    makeChunk: makeChunkFn<T, S>,
    parentMakeChunk: makeChunkFn<MetaTuple, MetaSequence>,
    boundaryChecker: BoundaryChecker<T>,
    newBoundaryChecker: NewBoundaryCheckerFn): Promise<Sequence> {

  const chunker = new SequenceChunker(cursor, null, makeChunk, parentMakeChunk, boundaryChecker,
                                      newBoundaryChecker);
  if (cursor) {
    await chunker.resume();
  }

  if (remove > 0) {
    invariant(cursor);
    for (let i = 0; i < remove; i++) {
      await chunker.skip();
    }
  }

  insert.forEach(i => chunker.append(i));

  return chunker.done(null);
}

// Like |chunkSequence|, but without an existing cursor (implying this is a new collection), so it
// can be synchronous. Necessary for constructing collections without a Promises or async/await.
// There is no equivalent in the Go code because Go is already synchronous.
export function chunkSequenceSync<T, S: Sequence<T>>(
    insert: Array<T>,
    makeChunk: makeChunkFn<T, S>,
    parentMakeChunk: makeChunkFn<MetaTuple, MetaSequence>,
    boundaryChecker: BoundaryChecker<T>,
    newBoundaryChecker: NewBoundaryCheckerFn): Sequence {

  const chunker = new SequenceChunker(null, null, makeChunk, parentMakeChunk, boundaryChecker,
                                      newBoundaryChecker);

  insert.forEach(i => chunker.append(i));

  return chunker.doneSync();
}

export default class SequenceChunker<T, S: Sequence<T>> {
  _cursor: ?SequenceCursor<T, S>;
  _vw: ?ValueWriter;
  _parent: ?SequenceChunker<MetaTuple, MetaSequence>;
  _current: Array<T>;
  _isLeaf: boolean;
  _lastSeq: ?S;
  _makeChunk: makeChunkFn<T, S>;
  _parentMakeChunk: makeChunkFn<MetaTuple, MetaSequence>;
  _boundaryChecker: BoundaryChecker<T>;
  _newBoundaryChecker: NewBoundaryCheckerFn;
  _done: boolean;

  constructor(cursor: ?SequenceCursor, vw: ?ValueWriter, makeChunk: makeChunkFn,
              parentMakeChunk: makeChunkFn,
              boundaryChecker: BoundaryChecker<T>,
              newBoundaryChecker: NewBoundaryCheckerFn) {
    this._cursor = cursor;
    this._vw = vw;
    this._parent = null;
    this._current = [];
    this._isLeaf = true;
    this._lastSeq = null;
    this._makeChunk = makeChunk;
    this._parentMakeChunk = parentMakeChunk;
    this._boundaryChecker = boundaryChecker;
    this._newBoundaryChecker = newBoundaryChecker;
    this._done = false;
  }

  async resume(): Promise<void> {
    const cursor = notNull(this._cursor);
    if (cursor.parent) {
      this.createParent();
      await notNull(this._parent).resume();
    }

    // Number of previous items which must be hashed into the boundary checker.
    let primeHashWindow = this._boundaryChecker.windowSize - 1;

    const retreater = cursor.clone();
    let appendCount = 0;
    let primeHashCount = 0;

    // If the cursor is beyond the final position in the sequence, the preceeding item may have been
    // a chunk boundary. In that case, we must test at least the preceeding item.
    const appendPenultimate = cursor.idx === cursor.length;
    if (appendPenultimate && await retreater._retreatMaybeAllowBeforeStart(false)) {
      // In that case, we prime enough items *prior* to the penultimate item to be correct.
      appendCount++;
      primeHashCount++;
    }

    // Walk backwards to the start of the existing chunk
    while (retreater.indexInChunk > 0 && await retreater._retreatMaybeAllowBeforeStart(false)) {
      appendCount++;
      if (primeHashWindow > 0) {
        primeHashCount++;
        primeHashWindow--;
      }
    }

    // If the hash window won't be filled by the preceeding items in the current chunk, walk further
    // back until they will.
    while (primeHashWindow > 0 && await retreater._retreatMaybeAllowBeforeStart(false)) {
      primeHashCount++;
      primeHashWindow--;
    }

    while (primeHashCount > 0 || appendCount > 0) {
      const item = retreater.getCurrent();
      if (primeHashCount > appendCount) {
        // Before the start of the current chunk: just hash value bytes into window
        this._boundaryChecker.write(item);
        primeHashCount--;
      } else if (appendCount > primeHashCount) {
        // In current chunk, but before window: just append item
        this._current.push(item);
        appendCount--;
      } else {
        // Within current chunk and hash window: append item & hash value bytes into window.
        if (appendPenultimate && appendCount === 1) {
          // It's ONLY correct Append immediately preceeding the cursor position because only after
          // its insertion into the hash will the window be filled.
          this.append(item);
        } else {
          this._boundaryChecker.write(item);
          this._current.push(item);
        }
        appendCount--;
        primeHashCount--;
      }

      await retreater.advance();
    }
  }

  append(item: T) {
    this._current.push(item);
    if (this._boundaryChecker.write(item)) {
      this.handleChunkBoundary();
    }
  }

  async skip(): Promise<void> {
    const cursor = notNull(this._cursor);

    if (await cursor.advance() && cursor.indexInChunk === 0) {
      await this.skipParentIfExists();
    }
  }

  async skipParentIfExists(): Promise<void> {
    if (this._parent && this._parent._cursor) {
      await this._parent.skip();
    }
  }

  createParent() {
    invariant(!this._parent);
    this._parent = new SequenceChunker(
        this._cursor && this._cursor.parent ? this._cursor.parent.clone() : null,
        this._vw,
        this._parentMakeChunk,
        this._parentMakeChunk,
        this._newBoundaryChecker(),
        this._newBoundaryChecker);
    this._parent._isLeaf = false;
  }

  createSequence(): [Sequence, MetaTuple] {
    let [col, key, numLeaves] = this._makeChunk(this._current); // eslint-disable-line prefer-const
    const seq = col.sequence;
    let ref: Ref;
    if (this._vw) {
      ref = this._vw.writeValue(col);
      col = null;
    } else {
      ref = new Ref(col);
    }
    const mt = new MetaTuple(ref, key, numLeaves, col);
    this._current = [];
    return [seq, mt];
  }

  handleChunkBoundary() {
    invariant(this._current.length > 0);
    const mt = this.createSequence()[1];
    if (!this._parent) {
      this.createParent();
    }

    notNull(this._parent).append(mt);
  }

  anyPending(): boolean {
    if (this._current.length > 0) {
      return true;
    }

    if (this._parent) {
      return this._parent.anyPending();
    }

    return false;
  }

  // Returns the root sequence of the resulting tree.
  async done(vr: ?ValueReader): Promise<Sequence> {
    invariant(!vr === !this._vw);
    invariant(!this._done);
    this._done = true;

    if (this._cursor) {
      await this.finalizeCursor();
    }

    if (!this._parent || !this._parent.anyPending()) {
      if (this._isLeaf) {
        // Return the (possibly empty) sequence which never chunked
        return this.createSequence()[0];
      }

      if (this._current.length === 1) {
        // Walk down until we find either a leaf sequence or meta sequence with more than one
        // metaTuple.
        const mt = this._current[0];
        invariant(mt instanceof MetaTuple);
        let seq = await mt.getChildSequence(vr);

        while (seq.isMeta && seq.length === 1) {
          seq = await seq.getChildSequence(0);
        }

        return seq;
      }
    }

    if (this._current.length > 0) {
      this.handleChunkBoundary();
    }

    return notNull(this._parent).done(vr);
  }

  // Like |done|, but assumes there is no cursor, so it can be synchronous. Necessary for
  // constructing collections without Promises or async/await. There is no equivalent in the Go
  // code because Go is already synchronous.
  doneSync(): Sequence {
    invariant(!this._vw);
    invariant(!this._cursor);
    invariant(!this._done);
    this._done = true;

    if (!this._parent || !this._parent.anyPending()) {
      if (this._isLeaf) {
        // Return the (possibly empty) sequence which never chunked
        return this.createSequence()[0];
      }

      if (this._current.length === 1) {
        // Walk down until we find either a leaf sequence or meta sequence with more than one
        // metaTuple.
        const mt = this._current[0];
        invariant(mt instanceof MetaTuple);
        let seq = mt.getChildSequenceSync();

        while (seq.isMeta && seq.length === 1) {
          seq = seq.getChildSequenceSync(0);
        }

        return seq;
      }
    }

    if (this._current.length > 0) {
      this.handleChunkBoundary();
    }

    return notNull(this._parent).doneSync();
  }

  async finalizeCursor(): Promise<void> {
    const cursor = notNull(this._cursor);
    if (!cursor.valid) {
      // The cursor is past the end, and due to the way cursors work, the parent cursor will
      // actually point to its last chunk. We need to force it to point past the end so that our
      // parent's Done() method doesn't add the last chunk twice.
      await this.skipParentIfExists();
      return;
    }

    // Append the rest of the values in the sequence, up to the window size, plus the rest of that
    // chunk. It needs to be the full window size because anything that was appended/skipped between
    // chunker construction and finalization will have changed the hash state.
    let hashWindow = this._boundaryChecker.windowSize;
    const fzr = cursor.clone();

    let i = 0;
    for (; hashWindow > 0 || fzr.indexInChunk > 0; i++) {
      if (i === 0 || fzr.indexInChunk === 0) {
        await this.skipParentIfExists();
      }
      const item = fzr.getCurrent();
      const didAdvance = await fzr.advance();

      if (hashWindow > 0) {
        // While we are within the hash window, append items (which explicit checks the hash value
        // for chunk boundaries)
        this.append(item);
        hashWindow--;
      } else {
        // Once we are beyond the hash window, we know that boundaries can only occur in the same
        // place they did within the existing sequence
        this._current.push(item);
        if (didAdvance && fzr.indexInChunk === 0) {
          this.handleChunkBoundary();
        }
      }
      if (!didAdvance) {
        break;
      }
    }
  }
}
