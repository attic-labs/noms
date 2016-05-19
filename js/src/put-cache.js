// @flow

import tingodb from 'tingodb';
import type {tcoll as Collection} from 'tingodb';
import fs from 'fs';
import {default as Chunk, emptyChunk} from './chunk.js';
import {invariant} from './assert.js';

const __tingodb = tingodb();

const tDb = __tingodb.Db;
const Binary = __tingodb.Binary;

type ChunkStream = (cb: (chunk: Chunk) => void) => Promise<void>
type ChunkItem = {hash: string, refHeight: number, data: Uint8Array, gen: number};
type DbRecord = {hash: string, refHeight: number, data: Binary, gen: number};

declare class CursorStream {
  pause(): void;
  resume(): void;
  on(event: 'data', cb: (record: DbRecord) => void): void;
  on(event: 'end', cb: () => void): void;
}

type ChunkIndex = Map<string, number>;

export default class OrderedPutCache {
  _chunkIndex: ChunkIndex;
  _db: Promise<Db>;
  _coll: Promise<DbCollection>;
  _appends: Set<Promise<void>>;
  gen: number;

  constructor() {
    this._chunkIndex = new Map();
    this._db = makeTempDir().then(folder => new Db(folder));
    this._coll = this._db
      .then(db => db.collection('puts'))
      .then(coll =>
        coll.ensureIndex({hash: 1}, {unique: true})
          .then(() => coll.ensureIndex({refHeight: 1}))
          .then(() => coll.ensureIndex({gen: 1}))
          .then(() => coll)
        );

    this._appends = new Set();
    this.gen = 0;
  }

  insert(c: Chunk, refHeight: number): boolean {
    const hash = c.ref.toString();
    if (this._chunkIndex.has(hash)) {
      return false;
    }
    this._chunkIndex.set(hash, -1);
    const currentGen = this.gen;
    const p = this._coll
      .then(coll => coll.insert({hash: hash, refHeight: refHeight, data: c.data, gen: currentGen}))
      .then((gen) => this._chunkIndex.set(hash, gen))
      .then(() => { this._appends.delete(p); });
    this._appends.add(p);
    return true;
  }

  get(hash: string): ?Promise<Chunk> {
    if (!this._chunkIndex.has(hash)) {
      return null;
    }
    //$FlowIssue
    return Promise.all(this._appends)
      .then(() => this._coll)
      .then(coll => coll.findOne(hash))
      .then(item => {
        if (item) {
          return new Chunk(item.data);
        }
        return emptyChunk;
      });
  }

  dropGeneration(gen: number): Promise<void> {
    //$FlowIssue
    return Promise.all(this._appends).then(() => this._coll).then(coll => {
      let count = 0;
      for (const [hash, chunkGen] of this._chunkIndex) {
        if (chunkGen === gen) {
          count++;
          this._chunkIndex.delete(hash);
        }
      }
      return coll.dropGeneration(gen).then(dropped => invariant(dropped === count));
    });
  }

  extractChunks(gen: number): Promise<ChunkStream> {
    //$FlowIssue
    return Promise.all(this._appends)
      .then(() => this._coll)
      .then(coll => coll.findGen(gen));
  }

  destroy(): Promise<void> {
    return this._db.then(db => db.destroy());
  }
}

function createChunkStream(stream: CursorStream): ChunkStream {
  return function(cb: (chunk: Chunk) => void): Promise<void> {
    return new Promise(fulfill => {
      stream.on('data', (record: DbRecord) => {
        const item = recordToItem(record);
        cb(new Chunk(item.data));
      });

      stream.resume();
      stream.on('end', fulfill);
    });
  };
}

class Db {
  _db: tDb;
  _folder: string;

  constructor(folder: string) {
    this._folder = folder;
    this._db = new tDb(folder, {});
  }

  collection(name: string): Promise<DbCollection> {
    return new DbCollection(this._db.collection(name));
  }

  destroy(): Promise<void> {
    return this._close().then(() => removeDir(this._folder));
  }

  _close(): Promise<void> {
    return new Promise((resolve, reject) => {
      this._db.close(err => {
        if (err) {
          reject(err);
        } else {
          resolve();
        }
      });
    });
  }
}

class DbCollection {
  _coll: Collection;

  constructor(coll: Collection) {
    this._coll = coll;
  }

  ensureIndex(obj: Object, options: Object = {}): Promise<void> {
    return new Promise((resolve, reject) => {
      options.w = 1;
      this._coll.ensureIndex(obj, options, (err) => {
        if (err) {
          reject(err);
        } else {
          resolve();
        }
      });
    });
  }

  insert(item: ChunkItem, options: Object = {}): Promise<number> {
    return new Promise((resolve, reject) => {
      options.w = 1;
      this._coll.insert(itemToRecord(item), options, err => {
        if (err) {
          reject(err);
        } else {
          resolve(item.gen);
        }
      });
    });
  }

  findOne(hash: string, options: Object = {}): Promise<ChunkItem> {
    return new Promise((resolve, reject) => {
      options.w = 1;
      this._coll.findOne({hash: hash}, options, (err, record) => {
        if (err) {
          reject(err);
        } else {
          resolve(recordToItem(record));
        }
      });
    });
  }

  findGen(gen: number, options: Object = {}): ChunkStream {
    options.w = 1;
    options.hint = {gen: 1, refHeight: 1};
    options.sort = {refHeight: 1};
    const stream = this._coll.find({gen: gen}, options).stream();
    stream.pause();
    return createChunkStream(stream);
  }

  dropGeneration(gen: number, options: Object = {}): Promise<number> {
    return new Promise((resolve, reject) => {
      options.w = 1;
      this._coll.remove({gen: gen}, options, (err, numRemovedDocs) => {
        if (err) {
          reject(err);
        } else {
          resolve(numRemovedDocs);
        }
      });
    });
  }
}

function recordToItem(rec: DbRecord): ChunkItem {
  return {
    hash: rec.hash,
    refHeight: rec.refHeight,
    data: new Uint8Array(rec.data.buffer),
    gen: rec.gen,
  };
}

function itemToRecord(item: ChunkItem): DbRecord {
  return {
    hash: item.hash,
    refHeight: item.refHeight,
    //$FlowIssue
    data: new Binary(new Buffer(item.data.buffer)),
    gen: item.gen,
  };
}

function makeTempDir(): Promise<string> {
  return new Promise((resolve, reject) => {
    //$FlowIssue
    fs.mkdtemp('/tmp/put-cache-', (err, folder) => {
      if (err) {
        reject(err);
      } else {
        resolve(folder);
      }
    });
  });
}

async function removeDir(dir: string): Promise<void> {
  await access(dir);
  const files = await readdir(dir);
  for (const file of files) {
    await unlink(dir + '/' + file);
  }
  return rmdir(dir);
}

function access(path: string, mode = fs.F_OK): Promise<void> {
  return new Promise((resolve, reject) => {
    fs.access(path, mode, (err) => {
      if (err) {
        reject(err);
      } else {
        resolve();
      }
    });
  });
}

function readdir(path: string): Promise<Array<string>> {
  return new Promise((resolve, reject) => {
    fs.readdir(path, (err, files) => {
      if (err) {
        reject(err);
      } else {
        resolve(files);
      }
    });
  });
}

function rmdir(path: string): Promise<void> {
  return new Promise((resolve, reject) => {
    fs.rmdir(path, (err) => {
      if (err) {
        reject(err);
      } else {
        resolve();
      }
    });
  });
}

function unlink(path: string): Promise<void> {
  return new Promise((resolve, reject) => {
    fs.unlink(path, (err) => {
      if (err) {
        reject(err);
      } else {
        resolve();
      }
    });
  });
}
