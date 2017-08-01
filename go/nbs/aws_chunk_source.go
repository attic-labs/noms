// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"sync"

	"github.com/attic-labs/noms/go/d"
)

type awsChunkSource struct {
	tableReader
	accessor tableAccessor
	name     addr
}

func newAWSChunkSource(accessor tableAccessor, name addr, indexCache *indexCache, stats *Stats) chunkSource {
	source := &awsChunkSource{accessor: accessor, name: name}
	var index tableIndex
	found := false
	if indexCache != nil {
		indexCache.lockEntry(name)
		defer indexCache.unlockEntry(name)
		index, found = indexCache.get(name)
	}

	if !found {
		index = source.accessor.getTableIndex(stats)
		if indexCache != nil {
			indexCache.put(name, index)
		}
	}

	source.tableReader = newTableReader(index, source, s3BlockSize)
	return source
}

func (acs *awsChunkSource) hash() addr {
	return acs.name
}

func (acs *awsChunkSource) ReadAtWithStats(p []byte, off int64, stats *Stats) (n int, err error) {
	return acs.accessor.getTableReaderAt(stats).ReadAtWithStats(p, off, stats)
}

type tableAccessor interface {
	getTableIndex(stats *Stats) tableIndex
	getTableReaderAt(stats *Stats) tableReaderAt
}

type trivialTableAccessor struct {
	tra tableReaderAt
}

func (tta trivialTableAccessor) getTableIndex(stats *Stats) tableIndex {
	panic("Not Reached")
}

func (tta trivialTableAccessor) getTableReaderAt(stats *Stats) tableReaderAt {
	return tta.tra
}

type oneShotTableQuery struct {
	once  sync.Once
	index tableIndex
	tra   tableReaderAt

	al  awsLimits
	ddb *ddbTableStore
	s3  *s3ObjectReader

	name       addr
	chunkCount uint32
}

func (o *oneShotTableQuery) getTableIndex(stats *Stats) tableIndex {
	o.once.Do(func() { o.resolve(stats) })
	return o.index
}

func (o *oneShotTableQuery) getTableReaderAt(stats *Stats) tableReaderAt {
	o.once.Do(func() { o.resolve(stats) })
	return o.tra
}

func (o *oneShotTableQuery) resolve(stats *Stats) {
	if o.al.tableMayBeInDynamo(o.chunkCount) {
		data, err := o.ddb.ReadTable(o.name, stats)
		if data != nil {
			o.index = parseTableIndex(data)
			o.tra = &dynamoTableReaderAt{ddb: o.ddb, h: o.name}
			return
		}
		d.PanicIfTrue(err == nil) // There MUST be either data or an error
		d.PanicIfNotType(err, tableNotInDynamoErr{})
	}

	size := indexSize(o.chunkCount) + footerSize
	buff := make([]byte, size)

	n, err := o.s3.ReadFromEnd(o.name, buff, stats)
	d.PanicIfError(err)
	d.PanicIfFalse(size == uint64(n))
	o.index = parseTableIndex(buff)
	o.tra = &s3TableReaderAt{s3: o.s3, h: o.name}
}
