// @flow

// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import Chunk from './chunk.js';
import Hash, {sha1Size} from './hash.js';
import ValueDecoder from './value-decoder.js';
import ValueEncoder from './value-encoder.js';
import {invariant} from './assert.js';
import {setEncodeValue} from './get-hash.js';
import {setHash, ValueBase} from './value.js';
import type Value from './value.js';
import type {ValueReader, ValueWriter} from './value-store.js';
import Bytes from './bytes.js';

export function encodeValue(v: Value, vw: ?ValueWriter): Chunk {
  const w = new BinaryNomsWriter();
  const enc = new ValueEncoder(w, vw);
  enc.writeValue(v);
  const chunk = new Chunk(w.data);
  if (v instanceof ValueBase) {
    setHash(v, chunk.hash);
  }

  return chunk;
}

setEncodeValue(encodeValue);

export function decodeValue(chunk: Chunk, vr: ValueReader): Value {
  const data = chunk.data;
  const dec = new ValueDecoder(new BinaryNomsReader(data), vr);
  const v = dec.readValue();

  if (v instanceof ValueBase) {
    setHash(v, chunk.hash);
  }

  return v;
}

const maxUInt32 = Math.pow(2, 32);
const littleEndian = true;

export interface NomsReader {
  pos(): number;
  seek(idx: number): void;
  sliceFrom(idx: number): Uint8Array;
  readBytes(): Uint8Array;
  readUint8(): number;
  readUint32(): number;
  readUint64(): number;
  readFloat64(): number;
  readBool(): boolean;
  readString(): string;
  scanString(): void;
  readHash(): Hash;
}

export interface NomsWriter {
  pos(): number;
  sliceFrom(idx: number): Uint8Array;
  append(data: Uint8Array): void;
  writeBytes(v: Uint8Array): void;
  writeUint8(v: number): void;
  writeUint32(v: number): void;
  writeUint64(v: number): void;
  writeFloat64(v: number): void;
  writeBool(v:boolean): void;
  writeString(v: string): void;
  writeHash(h: Hash): void;
}

export class BinaryNomsReader {
  buff: Uint8Array;
  dv: DataView;
  offset: number;

  constructor(buff: Uint8Array) {
    this.buff = buff;
    this.dv = new DataView(buff.buffer, buff.byteOffset, buff.byteLength);
    this.offset = 0;
  }

  pos(): number {
    return this.offset;
  }

  seek(pos: number) {
    this.offset = pos;
  }

  sliceFrom(idx: number): Uint8Array {
    return new Uint8Array(this.buff, idx, this.offset - idx);
  }

  readBytes(): Uint8Array {
    const size = this.readUint32();
    // Make a copy of the buffer to return
    const v = Bytes.slice(this.buff, this.offset, this.offset + size);
    this.offset += size;
    return v;
  }

  readUint8(): number {
    const v = this.dv.getUint8(this.offset);
    this.offset++;
    return v;
  }

  readUint32(): number {
    const v = this.dv.getUint32(this.offset, littleEndian);
    this.offset += 4;
    return v;
  }

  readUint64(): number {
    const lsi = this.readUint32();
    const msi = this.readUint32();
    const v = msi * maxUInt32 + lsi;
    invariant(v <= Number.MAX_SAFE_INTEGER);
    return v;
  }

  readFloat64(): number {
    const v = this.dv.getFloat64(this.offset, littleEndian);
    this.offset += 8;
    return v;
  }

  readBool(): boolean {
    const v = this.readUint8();
    invariant(v === 0 || v === 1);
    return v === 1;
  }

  scanString() {
    const size = this.readUint32();
    this.offset += size;
  }

  readString(): string {
    const size = this.readUint32();
    const str = Bytes.readUtf8(this.buff, this.offset, this.offset + size);
    this.offset += size;
    return str;
  }

  readHash(): Hash {
    // Make a copy of the data.
    const digest = Bytes.slice(this.buff, this.offset, this.offset + sha1Size);
    this.offset += sha1Size;
    return new Hash(digest);
  }
}

const initialBufferSize = 16;

export class BinaryNomsWriter {
  buff: Uint8Array;
  dv: DataView;
  offset: number;

  constructor() {
    this.buff = Bytes.alloc(initialBufferSize);
    this.dv = new DataView(this.buff.buffer, 0);
    this.offset = 0;
  }

  pos(): number {
    return this.offset;
  }

  sliceFrom(idx: number): Uint8Array {
    return new Uint8Array(this.buff, idx, this.offset - idx);
  }

  append(data: Uint8Array) {
    const size = data.byteLength;
    this.ensureCapacity(size);
    const a = new Uint8Array(this.buff, this.offset, size);
    a.set(data);
    this.offset += size;
  }

  get data(): Uint8Array {
    // Callers now owns the copied data.
    return Bytes.slice(this.buff, 0, this.offset);
  }

  ensureCapacity(n: number): void {
    let length = this.buff.byteLength;
    if (this.offset + n <= length) {
      return;
    }

    while (this.offset + n > length) {
      length *= 2;
    }

    this.buff = Bytes.grow(this.buff, length);
    this.dv = new DataView(this.buff.buffer);
  }

  writeBytes(v: Uint8Array): void {
    const size = v.byteLength;
    this.writeUint32(size);

    this.ensureCapacity(size);
    Bytes.copy(v, this.buff, this.offset);
    this.offset += size;
  }

  writeUint8(v: number): void {
    this.ensureCapacity(1);
    this.dv.setUint8(this.offset, v);
    this.offset++;
  }

  writeUint32(v: number): void {
    this.ensureCapacity(4);
    this.dv.setUint32(this.offset, v, littleEndian);
    this.offset += 4;
  }

  writeUint64(v: number): void {
    invariant(v <= Number.MAX_SAFE_INTEGER);
    const v2 = (v / maxUInt32) | 0;
    const v1 = v % maxUInt32;
    this.writeUint32(v1);
    this.writeUint32(v2);
  }

  writeFloat64(v: number): void {
    this.ensureCapacity(8);
    this.dv.setFloat64(this.offset, v, littleEndian);
    this.offset += 8;
  }

  writeBool(v:boolean): void {
    this.writeUint8(v ? 1 : 0);
  }

  writeString(v: string): void {
    // TODO: This is a bummer. Ensure even the largest UTF8 string will fit.
    this.ensureCapacity(4 + v.length * 4);
    this.offset = Bytes.encodeUtf8(v, this.buff, this.dv, this.offset);
  }

  writeHash(h: Hash): void {
    this.ensureCapacity(sha1Size);
    Bytes.copy(h.digest, this.buff, this.offset);
    this.offset += sha1Size;
  }
}
