// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/attic-labs/noms/go/d"
)

func newFSTablePersister(dir string, fc *fdCache, indexCache *indexCache) tablePersister {
	d.PanicIfTrue(fc == nil)
	return &fsTablePersister{dir, fc, indexCache}
}

type fsTablePersister struct {
	dir        string
	fc         *fdCache
	indexCache *indexCache
}

func (ftp *fsTablePersister) Open(spec tableSpec) chunkSource {
	return newMmapTableReader(ftp.dir, spec.name, spec.chunkCount, ftp.indexCache, ftp.fc)
}

func (ftp *fsTablePersister) Persist(spec tableSpec, data []byte) chunkSource {
	tempName := func() string {
		temp, err := ioutil.TempFile(ftp.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)
		io.Copy(temp, bytes.NewReader(data))
		index := parseTableIndex(data)
		if ftp.indexCache != nil {
			ftp.indexCache.put(spec.name, index)
		}
		return temp.Name()
	}()
	err := os.Rename(tempName, filepath.Join(ftp.dir, spec.name.String()))
	d.PanicIfError(err)
	return ftp.Open(spec)
}

func (ftp *fsTablePersister) ConjoinAll(spec tableSpec, sources chunkSources, index []byte) chunkSource {
	tempName := func() string {
		temp, err := ioutil.TempFile(ftp.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)

		for _, source := range sources {
			r := source.reader()
			n, err := io.CopyN(temp, r, int64(source.chunkDataLen()))
			d.PanicIfError(err)
			d.PanicIfFalse(uint64(n) == source.chunkDataLen())
		}
		_, err = temp.Write(index)
		d.PanicIfError(err)

		index := parseTableIndex(index)
		if ftp.indexCache != nil {
			ftp.indexCache.put(spec.name, index)
		}
		return temp.Name()
	}()

	err := os.Rename(tempName, filepath.Join(ftp.dir, spec.name.String()))
	d.PanicIfError(err)

	return ftp.Open(spec)
}
