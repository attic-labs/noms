// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

func serializeHashes(w io.Writer, hashes hash.HashSlice) {
	err := binary.Write(w, binary.BigEndian, uint32(len(hashes))) // 4 billion hashes is probably absurd. Maybe this should be smaller?
	d.Chk.NoError(err)
	for _, r := range hashes {
		serializeHash(w, r)
	}
}

func serializeHash(w io.Writer, h hash.Hash) {
	n, err := io.Copy(w, bytes.NewReader(h[:]))
	d.Chk.NoError(err)
	d.PanicIfFalse(int64(hash.ByteLen) == n)
}

func deserializeHashes(reader io.Reader) hash.HashSlice {
	numRefs := uint32(0)
	err := binary.Read(reader, binary.BigEndian, &numRefs)
	d.Chk.NoError(err)

	hashes := make(hash.HashSlice, numRefs)
	for i := range hashes {
		hashes[i] = deserializeHash(reader)
	}
	return hashes
}

func deserializeHash(reader io.Reader) hash.Hash {
	h := hash.Hash{}
	n, err := io.ReadFull(reader, h[:])
	d.Chk.NoError(err)
	d.PanicIfFalse(int(hash.ByteLen) == n)
	return h
}
