package chunks

import (
	"bytes"
	"flag"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	rootKey = "/root"
)

type LevelDBStore struct {
	db *leveldb.DB
	m  *sync.Mutex
}

func NewLevelDBStore(dir string) LevelDBStore {
	d.Chk.NotEmpty(dir)
	d.Chk.NoError(os.MkdirAll(dir, 0700))
	db, err := leveldb.OpenFile(dir, &opt.Options{
		Compression: opt.NoCompression,
		Filter:      filter.NewBloomFilter(10), // 10 bits/key
		WriteBuffer: 1 << 24,                   // 16MiB
	})
	d.Chk.NoError(err)
	return LevelDBStore{db, &sync.Mutex{}}
}

func (l LevelDBStore) Root() ref.Ref {
	val, err := l.db.Get([]byte(rootKey), nil)
	if err == errors.ErrNotFound {
		return ref.Ref{}
	}
	d.Chk.NoError(err)

	return ref.MustParse(string(val))
}

func (l LevelDBStore) UpdateRoot(current, last ref.Ref) bool {
	l.m.Lock()
	defer l.m.Unlock()
	if last != l.Root() {
		return false
	}

	// Sync: true write option should fsync memtable data to disk
	err := l.db.Put([]byte(rootKey), []byte(current.String()), &opt.WriteOptions{Sync: true})
	d.Chk.NoError(err)
	return true
}

func (l LevelDBStore) Get(ref ref.Ref) (io.ReadCloser, error) {
	key := []byte(ref.String())
	chunk, err := l.db.Get(key, nil)
	if err == errors.ErrNotFound {
		return nil, nil
	}
	d.Chk.NoError(err)

	buff := bytes.NewBuffer(chunk)
	return ioutil.NopCloser(buff), nil
}

func (l LevelDBStore) Put() ChunkWriter {
	b := &bytes.Buffer{}
	h := ref.NewHash()
	return &ldbChunkWriter{
		db:     l.db,
		buffer: b,
		writer: io.MultiWriter(b, h),
		hash:   h,
	}
}

type ldbChunkWriter struct {
	db     *leveldb.DB
	buffer *bytes.Buffer
	writer io.Writer
	hash   hash.Hash
}

func (w *ldbChunkWriter) Write(data []byte) (int, error) {
	d.Chk.NotNil(w.buffer, "Write() cannot be called after Ref() or Close().")
	size, err := w.writer.Write(data)
	d.Chk.NoError(err)
	return size, nil
}

func (w *ldbChunkWriter) Ref() (ref.Ref, error) {
	d.Chk.NoError(w.Close())
	return ref.FromHash(w.hash), nil
}

func (w *ldbChunkWriter) Close() error {
	if w.buffer == nil {
		return nil
	}

	key := []byte(ref.FromHash(w.hash).String())

	exists, err := w.db.Has(key, &opt.ReadOptions{DontFillCache: true})
	d.Chk.NoError(err)
	if exists {
		return nil
	}

	err = w.db.Put(key, w.buffer.Bytes(), nil)
	d.Chk.NoError(err)
	w.buffer = nil
	return nil
}

type ldbStoreFlags struct {
	dir *string
}

func levelDBFlags(prefix string) ldbStoreFlags {
	return ldbStoreFlags{
		flag.String(prefix+"db", "", "directory to use for a LevelDB-ba chunkstore"),
	}
}

func (f ldbStoreFlags) createStore() ChunkStore {
	if *f.dir == "" {
		return nil
	} else {
		fs := NewLevelDBStore(*f.dir)
		return &fs
	}
}
