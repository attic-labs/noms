// @flow

import Hash, {sha1Size} from './hash.js';
import {encode, decode} from './utf8.js';
import {invariant} from './assert.js';

const maxUInt32 = Math.pow(2, 32);

export interface NomsReader {
  readBytes(): Uint8Array;
  readUint8(): number;
  readUint32(): number;
  readUint64(): number;
  readFloat64(): number;
  readBool(): boolean;
  readString(): string;
  readHash(): Hash;
}

export interface NomsWriter {
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
  buff: ArrayBuffer;
  dv: DataView;
  offset: number;
  length: number;

  constructor(buff: ArrayBuffer) {
    this.buff = buff;
    this.dv = new DataView(this.buff, 0);
    this.offset = 0;
    this.length = this.buff.byteLength;
  }

  checkCanRead(n: number) {
    invariant(this.offset + n <= this.length);
  }

  readBytes(): Uint8Array {
    const size = this.readUint32();
    this.checkCanRead(size);
    // Make a copy of the buffer to return
    const v = new Uint8Array(new Uint8Array(this.buff, this.offset, size));
    this.offset += size;
    return v;
  }

  readUint8(): number {
    this.checkCanRead(1);
    const v = this.dv.getUint8(this.offset);
    this.offset++;
    return v;
  }

  readUint32(): number {
    this.checkCanRead(4);
    const v = this.dv.getUint32(this.offset, true);
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
    this.checkCanRead(8);
    const v = this.dv.getFloat64(this.offset, true);
    this.offset += 8;
    return v;
  }

  readBool(): boolean {
    const v = this.readUint8();
    invariant(v === 0 || v === 1);
    return v === 1;
  }

  readString(): string {
    const size = this.readUint32();
    this.checkCanRead(size);
    const v = new Uint8Array(this.buff, this.offset, size);
    this.offset += size;
    return decode(v);
  }

  readHash(): Hash {
    this.checkCanRead(sha1Size);
    const digest = new Uint8Array(this.buff, this.offset, sha1Size);
    this.offset += sha1Size;
    // fromDigest doesn't take ownership of the memory, so it's safe to pass a view.
    return Hash.fromDigest(digest);
  }
}

const initialBufferSize = 1 << 11;

export class BinaryNomsWriter {
  buff: ArrayBuffer;
  dv: DataView;
  offset: number;
  length: number;

  constructor() {
    this.buff = new ArrayBuffer(initialBufferSize);
    this.dv = new DataView(this.buff, 0);
    this.offset = 0;
    this.length = this.buff.byteLength;
  }

  get data(): Uint8Array {
    // Callers now owns the copied data.
    return new Uint8Array(new Uint8Array(this.buff, 0, this.offset));
  }

  ensureCapacity(n: number): void {
    if (this.offset + n <= this.length) {
      return;
    }

    const oldData = new Uint8Array(this.buff);

    while (this.offset + n > this.length) {
      this.length *= 2;
    }
    this.buff = new ArrayBuffer(this.length);
    this.dv = new DataView(this.buff, 0);

    const a = new Uint8Array(this.buff);
    a.set(oldData);
  }

  writeBytes(v: Uint8Array): void {
    const size = v.byteLength;
    this.writeUint32(size);

    this.ensureCapacity(size);
    const a = new Uint8Array(this.buff, this.offset, size);
    a.set(v);
    this.offset += size;
  }

  writeUint8(v: number): void {
    this.ensureCapacity(1);
    this.dv.setUint8(this.offset, v);
    this.offset++;
  }

  writeUint32(v: number): void {
    this.ensureCapacity(4);
    this.dv.setUint32(this.offset, v, true);
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
    this.dv.setFloat64(this.offset, v, true);
    this.offset += 8;
  }

  writeBool(v:boolean): void {
    this.writeUint8(v ? 1 : 0);
  }

  writeString(v: string): void {
    this.writeBytes(encode(v));
  }

  writeHash(h: Hash): void {
    this.ensureCapacity(sha1Size);
    const a = new Uint8Array(this.buff, this.offset, sha1Size);
    a.set(h.digest);
    this.offset += sha1Size;
  }
}
