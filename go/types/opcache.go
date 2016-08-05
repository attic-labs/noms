// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"encoding/binary"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	collectionId = uint32(0)
	ldb          *opCacheDb
	activeOpcCnt = int32(0)
	once         = sync.Once{}
	activeMutex  = sync.Mutex{}
)

const prefixByteSize = 4

type opCacheDb struct {
	ops   *leveldb.DB
	dbDir string
}

func newLevelDb() *opCacheDb {
	dir, err := ioutil.TempDir("", "")
	d.Chk.NoError(err)
	db, err := leveldb.OpenFile(dir, &opt.Options{
		Compression:            opt.NoCompression,
		Comparer:               opCacheComparer{},
		OpenFilesCacheCapacity: 24,
		NoSync:                 true,    // We don't need this data to be durable. LDB is acting as temporary storage that can be larger than main memory.
		WriteBuffer:            1 << 27, // 128MiB
	})
	d.Chk.NoError(err, "opening put cache in %s", dir)
	return &opCacheDb{ops: db, dbDir: dir}
}

func newOpCache(vrw ValueReadWriter) *opCache {
	incrementActiveColCount()
	prefix := [prefixByteSize]byte{}
	colId := atomic.AddUint32(&collectionId, 1)
	binary.LittleEndian.PutUint32(prefix[:], colId)
	return &opCache{vrw: vrw, colId: colId, prefix: prefix[:]}
}

type opCache struct {
	vrw              ValueReadWriter
	ldbKeyScratch    [1 + hash.ByteLen]byte
	keyScratch       [initialBufferSize]byte
	prefixKeyScratch [initialBufferSize]byte
	valScratch       [initialBufferSize]byte
	colId            uint32
	prefix           []byte
}

type opCacheIterator struct {
	opc  *opCache
	iter iterator.Iterator
	vr   ValueReader
}

var uint32Size = binary.Size(uint32(0))

func incrementActiveColCount() {
	activeMutex.Lock()
	defer activeMutex.Unlock()

	if activeOpcCnt == 0 {
		ldb = newLevelDb()
	}
	activeOpcCnt++
}

func decrementActiveColCount() (err error) {
	activeMutex.Lock()
	defer activeMutex.Unlock()

	if activeOpcCnt == 1 {
		d.Chk.NoError(ldb.ops.Close())
		err = os.RemoveAll(ldb.dbDir)
		ldb = nil
	}
	activeOpcCnt--
	return
}

func (p *opCache) PutOp(key, val []byte) error {
	copy(p.prefixKeyScratch[:], p.prefix)
	copy(p.prefixKeyScratch[prefixByteSize:], key)
	prefixedKey := p.prefixKeyScratch[:prefixByteSize+len(key)]
	return ldb.ops.Put(prefixedKey, val, nil)
}

// Set can be called from any goroutine
func (p *opCache) Set(mapKey Value, mapVal Value) {
	switch mapKey.Type().Kind() {
	default:
		// This is the complicated case. For non-primitives, we want the ldb key to be the hash of mapKey, but we obviously need to get both mapKey and mapVal into ldb somehow. The simplest thing is just to do this:
		//
		//     uint32 (4 bytes)             bytes                 bytes
		// +-----------------------+---------------------+----------------------+
		// | key serialization len |    serialized key   |   serialized value   |
		// +-----------------------+---------------------+----------------------+

		// Note that, if mapKey and/or mapVal are prolly trees, any in-memory child chunks will be written to vrw at this time.
		p.ldbKeyScratch[0] = byte(mapKey.Type().Kind())
		copy(p.ldbKeyScratch[1:], mapKey.Hash().DigestSlice())
		mapKeyData := encToSlice(mapKey, p.keyScratch[:], p.vrw)
		mapValData := encToSlice(mapVal, p.valScratch[:], p.vrw)

		mapKeyByteLen := len(mapKeyData)
		data := make([]byte, uint32Size+mapKeyByteLen+len(mapValData))
		binary.LittleEndian.PutUint32(data, uint32(mapKeyByteLen))
		copy(data[uint32Size:], mapKeyData)
		copy(data[uint32Size+mapKeyByteLen:], mapValData)

		// TODO: Will manually batching these help?
		err := p.PutOp(p.ldbKeyScratch[:], data)
		d.Chk.NoError(err)

	case BoolKind, NumberKind, StringKind:
		// In this case, we can just serialize mapKey and use it as the ldb key, so we can also just serialize mapVal and dump that into the DB.
		keyData := encToSlice(mapKey, p.keyScratch[:], p.vrw)
		valData := encToSlice(mapVal, p.valScratch[:], p.vrw)
		// TODO: Will manually batching these help?
		err := p.PutOp(keyData, valData)
		d.Chk.NoError(err)
	}
}

func encToSlice(v Value, initBuf []byte, vw ValueWriter) []byte {
	// TODO: Are there enough calls to this that it's worth re-using a nomsWriter and valueEncoder?
	w := &binaryNomsWriter{initBuf, 0}
	enc := newValueEncoder(w, vw)
	enc.writeValue(v)
	return w.data()
}

func (p *opCache) NewIterator() *opCacheIterator {
	return &opCacheIterator{opc: p, iter: ldb.ops.NewIterator(util.BytesPrefix(p.prefix), nil), vr: p.vrw}
}

func (p *opCache) Destroy() error {
	return decrementActiveColCount()
}

func (i *opCacheIterator) Next() bool {
	return i.iter.Next()
}

func (i *opCacheIterator) Op() sequenceItem {
	entry := mapEntry{}
	prefixedLdbKey := i.iter.Key()
	ldbKey := prefixedLdbKey[prefixByteSize:]
	data := i.iter.Value()
	dataOffset := 0
	switch NomsKind(ldbKey[0]) {
	case BoolKind, NumberKind, StringKind:
		entry.key = DecodeFromBytes(ldbKey, i.vr, staticTypeCache)
	default:
		keyBytesLen := int(binary.LittleEndian.Uint32(data))
		entry.key = DecodeFromBytes(data[uint32Size:uint32Size+keyBytesLen], i.vr, staticTypeCache)
		dataOffset = uint32Size + keyBytesLen
	}

	entry.value = DecodeFromBytes(data[dataOffset:], i.vr, staticTypeCache)
	return entry
}

// Insert can be called from any goroutine
func (p *opCache) Insert(val Value) {
	switch val.Type().Kind() {
	default:
		// This is the complicated case. For non-primitives, we want the ldb key to be the hash of mapKey, but we obviously need to get both mapKey and mapVal into ldb somehow. The simplest thing is just to do this:
		//
		//     uint32 (4 bytes)             bytes
		// +-----------------------+----------------------+
		// | key serialization len |   serialized value   |
		// +-----------------------+----------------------+

		// Note that, if val is a prolly trees, any in-memory child chunks will be written to vrw at this time.
		p.ldbKeyScratch[0] = byte(val.Type().Kind())
		copy(p.ldbKeyScratch[1:], val.Hash().DigestSlice())
		mapValData := encToSlice(val, p.valScratch[:], p.vrw)

		data := make([]byte, len(mapValData))
		copy(data, mapValData)

		// TODO: Will manually batching these help?
		err := p.PutOp(p.ldbKeyScratch[:], data)
		d.Chk.NoError(err)

	case BoolKind, NumberKind, StringKind:
		// In this case, we can just serialize mapKey and use it as the ldb key, so we can also just serialize mapVal and dump that into the DB.
		keyData := encToSlice(val, p.keyScratch[:], p.vrw)
		valData := encToSlice(val, p.valScratch[:], p.vrw)
		// TODO: Will manually batching these help?
		err := p.PutOp(keyData, valData)
		d.Chk.NoError(err)
	}
}

func (i *opCacheIterator) SetOp() sequenceItem {
	data := i.iter.Value()
	val := DecodeFromBytes(data, i.vr, staticTypeCache)
	return val
}

func (i *opCacheIterator) Release() {
	i.iter.Release()
}
