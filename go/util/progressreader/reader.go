// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package progressreader provides an io.Reader that can be queried for progress. Intended to be used with go/util/status.
package progressreader

import (
	"io"
)

type Reader struct {
	inner io.Reader
	seen  uint64
}

func New(inner io.Reader) *Reader {
	return &Reader{inner, 0}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.inner.Read(p)
	r.seen += uint64(n)
	return
}

func (r *Reader) Seen() uint64 {
	return r.seen
}
