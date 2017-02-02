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

type fsTablePersister struct {
	dir        string
	indexCache *indexCache
}

func (ftp fsTablePersister) Compact(mt *memTable, haver chunkReader) chunkSource {
	return ftp.toReader(mt.write(haver))
}

func (ftp fsTablePersister) toReader(name addr, data []byte, chunkCount uint32) chunkSource {
	if chunkCount == 0 {
		return emptyChunkSource{}
	}
	tempName := func() string {
		temp, err := ioutil.TempFile(ftp.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)
		io.Copy(temp, bytes.NewReader(data))
		return temp.Name()
	}()
	err := os.Rename(tempName, filepath.Join(ftp.dir, name.String()))
	d.PanicIfError(err)
	return ftp.Open(name, chunkCount)
}

func (ftp fsTablePersister) CompactAll(sources chunkSources) chunkSource {
	rl := make(chan struct{}, 32)
	defer close(rl)
	return ftp.toReader(compactSourcesToBuffer(sources, rl))
}

func (ftp fsTablePersister) Open(name addr, chunkCount uint32) chunkSource {
	return newMmapTableReader(ftp.dir, name, chunkCount, ftp.indexCache)
}
