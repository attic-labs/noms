// @flow

import Chunk from './chunk.js';
import Ref from './ref.js';
import {invariant} from './assert.js';

const headerSize = 4; // uint32
const bigEndian = false; // Passing false to DataView methods makes them use big-endian byte order.
const sha1Size = 20;
const chunkLengthSize = 4; // uint32
const chunkHeaderSize = sha1Size + chunkLengthSize;

export interface ChunkStreamer {
   register(cb: (chunk: Chunk) => void): void;
   done(): Promise<void>;
}

export class ChunkSerializer {
  _buf: ArrayBuffer;
  _offset: number;
  _streamer: ChunkStreamer;

  constructor(streamer: ChunkStreamer) {
    this._buf = new ArrayBuffer(1024);
    this._offset = 0;
    this._streamer = streamer;
  }

  run(hints: Set<Ref>): Promise<ArrayBuffer> {
    const hintsLen = serializedHintsLength(hints);
    if (this._buf.byteLength < hintsLen) {
      this._buf = new ArrayBuffer(hintsLen * 2); // Leave space for some chunks.
    }
    this._offset = serializeHints(hints, this._buf);
    this._streamer.register(chunk => {
      const chunkLength = serializedChunkLength(chunk);
      if (this._buf.byteLength < chunkLength) {
        let newLen = this._buf.byteLength;
        for (; newLen < chunkLength; newLen *= 2)
          ;
        const newBuf = new ArrayBuffer(newLen);
        new Uint8Array(newBuf).set(new Uint8Array(this._buf));
        this._buf = newBuf;
      }
      this._offset = serializeChunk(chunk, this._buf, this._offset);
    });

    return this._streamer.done().then(() => this._buf.slice(0, this._offset));
  }
}

export function serialize(hints: Set<Ref>, chunks: Array<Chunk>): ArrayBuffer {
  const buffer = new ArrayBuffer(serializedHintsLength(hints) + serializedChunksLength(chunks));

  let offset = serializeHints(hints, buffer);
  for (let i = 0; i < chunks.length; i++) {
    offset = serializeChunk(chunks[i], buffer, offset);
  }

  return buffer;
}

function serializeChunk(chunk: Chunk, buffer: ArrayBuffer, offset: number): number {
  invariant(buffer.byteLength - offset >= serializedChunkLength(chunk),
    'Invalid chunk buffer');

  const refArray = new Uint8Array(buffer, offset, sha1Size);
  refArray.set(chunk.ref.digest);
  offset += sha1Size;

  const chunkLength = chunk.data.length;
  const view = new DataView(buffer, offset, chunkLengthSize);
  view.setUint32(0, chunkLength | 0, bigEndian); // Coerce number to uint32
  offset += chunkLengthSize;

  const dataArray = new Uint8Array(buffer, offset, chunkLength);
  dataArray.set(chunk.data);
  offset += chunkLength;
  return offset;
}

function serializeHints(hints: Set<Ref>, buffer: ArrayBuffer): number {
  let offset = 0;
  const view = new DataView(buffer, offset, headerSize);
  view.setUint32(0, hints.size | 0, bigEndian); // Coerce number to uint32
  offset += headerSize;

  hints.forEach(ref => {
    const refArray = new Uint8Array(buffer, offset, sha1Size);
    refArray.set(ref.digest);
    offset += sha1Size;
  });

  return offset;
}

export function serializedHintsLength(hints: Set<Ref>): number {
  return headerSize + sha1Size * hints.size;
}

function serializedChunksLength(chunks: Array<Chunk>): number {
  let totalSize = 0;
  for (let i = 0; i < chunks.length; i++) {
    totalSize += serializedChunkLength(chunks[i]);
  }
  return totalSize;
}

function serializedChunkLength(chunk: Chunk): number {
  return chunkHeaderSize + chunk.data.length;
}

export function deserialize(buffer: ArrayBuffer): {hints: Array<Ref>, chunks: Array<Chunk>} {
  const {hints, offset} = deserializeHints(buffer);
  return {hints: hints, chunks: deserializeChunks(buffer, offset)};
}

function deserializeHints(buffer: ArrayBuffer): {hints: Array<Ref>, offset: number} {
  const hints:Array<Ref> = [];

  let offset = 0;
  const view = new DataView(buffer, offset, headerSize);
  const numHints = view.getUint32(0, bigEndian);
  offset += headerSize;

  const totalLength = headerSize + (numHints * sha1Size);
  for (; offset < totalLength;) {
    invariant(buffer.byteLength - offset >= sha1Size, 'Invalid hint buffer');

    const refArray = new Uint8Array(buffer, offset, sha1Size);
    const ref = Ref.fromDigest(new Uint8Array(refArray));
    offset += sha1Size;

    hints.push(ref);
  }

  return {hints: hints, offset: offset};
}

export function deserializeChunks(buffer: ArrayBuffer, offset: number = 0): Array<Chunk> {
  const chunks:Array<Chunk> = [];

  const totalLenth = buffer.byteLength;
  for (; offset < totalLenth;) {
    invariant(buffer.byteLength - offset >= chunkHeaderSize, 'Invalid chunk buffer');

    const refArray = new Uint8Array(buffer, offset, sha1Size);
    const ref = Ref.fromDigest(new Uint8Array(refArray));
    offset += sha1Size;

    const view = new DataView(buffer, offset, chunkLengthSize);
    const chunkLength = view.getUint32(0, bigEndian);
    offset += chunkLengthSize;

    invariant(offset + chunkLength <= totalLenth, 'Invalid chunk buffer');

    const dataArray = new Uint8Array(buffer, offset, chunkLength);
    const chunk = new Chunk(new Uint8Array(dataArray)); // Makes a slice (copy) of the byte sequence
                                                        // from buffer.

    invariant(chunk.ref.equals(ref), 'Serialized ref !== computed ref');

    offset += chunkLength;
    chunks.push(chunk);
  }

  return chunks;
}
