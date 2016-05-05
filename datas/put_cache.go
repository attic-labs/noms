package datas

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func newUnwrittenPutCache() *unwrittenPutCache {
	dir, err := ioutil.TempDir("", "")
	d.Exp.NoError(err)
	db, err := leveldb.OpenFile(dir, &opt.Options{
		Compression:            opt.NoCompression,
		Filter:                 filter.NewBloomFilter(10), // 10 bits/key
		OpenFilesCacheCapacity: 24,
		WriteBuffer:            1 << 24, // 16MiB,
	})
	d.Chk.NoError(err, "opening put cache in %s", dir)
	return &unwrittenPutCache{
		orderedChunks: db,
		chunkIndex:    map[ref.Ref][]byte{},
		dbDir:         dir,
		mu:            &sync.Mutex{},
	}
}

type unwrittenPutCache struct {
	orderedChunks *leveldb.DB
	chunkIndex    map[ref.Ref][]byte
	dbDir         string
	mu            *sync.Mutex
	next          uint64
}

func (p *unwrittenPutCache) Add(c chunks.Chunk) bool {
	hash := c.Ref()
	p.mu.Lock()
	// Don't use defer p.mu.Unlock() here, because I want writing to orderedChunks NOT to be guarded by the lock. LevelDB handles its own goroutine-safety.
	if _, ok := p.chunkIndex[hash]; !ok {
		key := toKey(p.next)
		p.next++
		p.chunkIndex[hash] = key
		p.mu.Unlock()

		buf := &bytes.Buffer{}
		gw := gzip.NewWriter(buf)
		sz := chunks.NewSerializer(gw)
		sz.Put(c)
		sz.Close()
		gw.Close()
		d.Chk.NoError(p.orderedChunks.Put(key, buf.Bytes(), nil))
		return true
	}
	p.mu.Unlock()
	return false
}

func (p *unwrittenPutCache) Has(hash ref.Ref) (has bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, has = p.chunkIndex[hash]
	return
}

func (p *unwrittenPutCache) Get(hash ref.Ref) chunks.Chunk {
	p.mu.Lock()
	// Don't use defer p.mu.Unlock() here, because I want reading from orderedChunks NOT to be guarded by the lock. LevelDB handles its own goroutine-safety.
	if key, ok := p.chunkIndex[hash]; ok {
		p.mu.Unlock()

		data, err := p.orderedChunks.Get(key, nil)
		d.Chk.NoError(err)
		reader, err := gzip.NewReader(bytes.NewReader(data))
		d.Chk.NoError(err)
		defer reader.Close()
		chunkChan := make(chan chunks.Chunk)
		go chunks.DeserializeToChan(reader, chunkChan)
		return <-chunkChan
	}
	p.mu.Unlock()
	return chunks.EmptyChunk
}

func (p *unwrittenPutCache) Clear(hashes ref.RefSlice) {
	toDelete := make([][]byte, len(hashes))
	p.mu.Lock()
	for i, hash := range hashes {
		toDelete[i] = p.chunkIndex[hash]
		delete(p.chunkIndex, hash)
	}
	p.mu.Unlock()
	for _, key := range toDelete {
		d.Chk.NoError(p.orderedChunks.Delete(key, nil))
	}
	return
}

func toKey(idx uint64) []byte {
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, idx)
	d.Chk.NoError(err)
	return buf.Bytes()
}

func fromKey(key []byte) (idx uint64) {
	err := binary.Read(bytes.NewReader(key), binary.BigEndian, &idx)
	d.Chk.NoError(err)
	return
}

type resettableReadCloser interface {
	io.ReadCloser
	Reset()
}

type putCacheReader struct {
	iterator.Iterator
	remaining []byte
}

func (p *unwrittenPutCache) GzipReader(start, end ref.Ref) resettableReadCloser {
	p.mu.Lock()
	defer p.mu.Unlock()
	iterRange := &util.Range{Start: p.chunkIndex[start], Limit: toKey(fromKey(p.chunkIndex[end]) + 1)}
	return &putCacheReader{Iterator: p.orderedChunks.NewIterator(iterRange, nil)}
}

func (p *unwrittenPutCache) Destroy() error {
	d.Chk.NoError(p.orderedChunks.Close())
	return os.RemoveAll(p.dbDir)
}

func (r *putCacheReader) Reset() {
	r.First()
	// First() sets r back to the first item in the iteration. In order to avoid having to detect this case in Read(), set r back to BEFORE the first item. This way, Read() can just always call Next() when it wants another item.
	r.Prev()
}

func (r *putCacheReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	// if anything in r.remaining, Copy up to len(p) bytes into p, reslice r.remaining and return.
	if len(r.remaining) > 0 {
		n = copy(p, r.remaining)
		r.remaining = r.remaining[n:]
		return
	}

	// if not, get the next thing, return up to len(p), and put the rest in r.remaining
	if r.Next() {
		d := r.Value()
		n = copy(p, d)
		r.remaining = d[n:]
		return
	}

	return 0, io.EOF
}

func (r *putCacheReader) Close() error {
	r.Release()
	return r.Error()
}
