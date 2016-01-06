// @flow

import Ref from './ref.js';
import {TextEncoder, TextDecoder} from './text_encoding.js';

const decoder = new TextDecoder();
const encoder = new TextEncoder();

export default class Chunk {
  data: Uint8Array;
  _ref: ?Ref;

  constructor(data: Uint8Array = new Uint8Array(0), ref: ?Ref) {
    this.data = data;
    this._ref = ref;
  }

  get ref(): Ref {
    // Work around Babel bug: https://phabricator.babeljs.io/T6926
    if (!this._ref) {
      this._ref = Ref.fromData(this.data);
    }
    return this._ref;
  }

  isEmpty(): boolean {
    return this.data.length === 0;
  }

  toString(): string {
    return decoder.decode(this.data);
  }

  static emptyChunk: Chunk;

  static fromString(s: string, ref: ?Ref): Chunk {
    return new Chunk(encoder.encode(s), ref);
  }
}

Chunk.emptyChunk = new Chunk();
