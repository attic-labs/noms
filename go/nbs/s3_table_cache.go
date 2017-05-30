// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/sizecache"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3TableCache struct {
	dir    string
	cache  *sizecache.SizeCache
	fd     *fdCache
	s3     s3svc
	bucket string
}

func newS3TableCache(dir string, cacheSize uint64, maxOpenFds int, s3 s3svc, bucket string) *s3TableCache {
	stc := &s3TableCache{dir: dir, fd: newFDCache(maxOpenFds), s3: s3, bucket: bucket}
	stc.cache = sizecache.NewWithExpireCallback(cacheSize, func(elm interface{}) {
		stc.expire(elm.(addr))
	})

	stc.init()
	return stc
}

func (stc *s3TableCache) init() {
	fmt.Println("TODO: Open dir, load existing files and prime cache")
}

func (stc *s3TableCache) checkout(h addr) io.ReaderAt {
	_, ok := stc.cache.Get(h)
	if !ok {
		return nil
	}

	fd, err := stc.fd.RefFile(filepath.Join(stc.dir, h.String()))
	if err != nil {
		return nil
	}

	return fd
}

func (stc *s3TableCache) checkin(h addr) {
	stc.fd.UnrefFile(filepath.Join(stc.dir, h.String()))
}

func (stc *s3TableCache) store(h addr, data []byte) {
	path := filepath.Join(stc.dir, h.String())
	tempName := func() string {
		temp, err := ioutil.TempFile(stc.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)
		io.Copy(temp, bytes.NewReader(data))
		return temp.Name()
	}()

	err := os.Rename(tempName, path)
	d.PanicIfError(err)

	stc.insert(h, uint64(len(data)))

	stc.fd.RefFile(path) // Prime the file in the fd cache
	stc.fd.UnrefFile(path)
}

func (stc *s3TableCache) load(h addr) {
	path := filepath.Join(stc.dir, h.String())

	tempName, length := func() (string, uint64) {
		input := &s3.GetObjectInput{
			Bucket: aws.String(stc.bucket),
			Key:    aws.String(h.String()),
		}
		result, err := stc.s3.GetObject(input)
		d.PanicIfError(err)

		temp, err := ioutil.TempFile(stc.dir, "nbs_table_")
		d.PanicIfError(err)
		defer checkClose(temp)
		io.Copy(temp, result.Body)
		return temp.Name(), uint64(*result.ContentLength)
	}()

	err := os.Rename(tempName, path)
	d.PanicIfError(err)

	stc.insert(h, length)

	stc.fd.RefFile(path) // Prime the file in the fd cache
	stc.fd.UnrefFile(path)
}

func (stc *s3TableCache) insert(h addr, size uint64) {
	stc.cache.Add(h, size, true)
}

func (stc *s3TableCache) expire(h addr) {
	err := os.Remove(filepath.Join(stc.dir, h.String()))
	d.PanicIfError(err)
}
