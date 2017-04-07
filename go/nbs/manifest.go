// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"crypto/sha512"
	"fmt"
	"strconv"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

var (
	errOptimisticLockFailedRoot   = fmt.Errorf("Root moved")
	errOptimisticLockFailedTables = fmt.Errorf("Tables changed")
)

type manifest interface {
	// LoadIfExists extracts and returns values from a NomsBlockStore
	// manifest, if one exists. Concrete implementations are responsible for
	// defining how to find and parse the desired manifest, e.g. a
	// particularly-named file in a given directory. Implementations are also
	// responsible for managing whatever concurrency guarantees they require
	// for correctness.
	// If the manifest exists, |exists| is set to true and manifest data is
	// returned, including the version of the Noms data in the store, the root
	// hash.Hash of the store, and a tableSpec describing every table that
	// comprises the store.
	// If the manifest doesn't exist, |exists| is set to false and the other
	// return values are undefined. The |readHook| parameter allows race
	// condition testing. If it is non-nil, it will be invoked while the
	// implementation is guaranteeing exclusive access to the manifest.
	LoadIfExists(readHook func()) (exists bool, vers string, root hash.Hash, tableSpecs []tableSpec)

	// Update optimistically tries to write a new manifest containing
	// |newRoot| and the tables referenced by |specs|. If the currently
	// persisted manifest has not changed since this instance was last loaded
	// or updated, then Update succeeds and subsequent calls to both Update
	// and LoadIfExists will reflect a manifest containing |newRoot| and
	// |tables|. If not, Update fails with an error indicating what caused the
	// optimistic lock failure. Regardless, |actual| and |tableSpecs| will
	// reflect the current state of the world upon return. Upon error, clients
	// should merge any desired new table information with the contents of
	// |tableSpecs| before trying again. Concrete implementations are
	// responsible for ensuring that concurrent Update calls (and LoadIfExists
	// calls) are correct.
	// If writeHook is non-nil, it will be invoked while the implementation is
	// guaranteeing exclusive access to the manifest. This allows for testing
	// of race conditions.
	Update(specs []tableSpec, root, newRoot hash.Hash, writeHook func()) (actual hash.Hash, tableSpecs []tableSpec, err error)
}

type tableSpec struct {
	name       addr
	chunkCount uint32
}

func parseSpecs(tableInfo []string) []tableSpec {
	specs := make([]tableSpec, len(tableInfo)/2)
	for i := range specs {
		specs[i].name = ParseAddr([]byte(tableInfo[2*i]))
		c, err := strconv.ParseUint(tableInfo[2*i+1], 10, 32)
		d.PanicIfError(err)
		specs[i].chunkCount = uint32(c)
	}
	return specs
}

func formatSpecs(specs []tableSpec, tableInfo []string) {
	d.Chk.True(len(tableInfo) == 2*len(specs))
	for i, t := range specs {
		tableInfo[2*i] = t.name.String()
		tableInfo[2*i+1] = strconv.FormatUint(uint64(t.chunkCount), 10)
	}
}

func generateLockHash(root hash.Hash, specs []tableSpec) (lock addr) {
	blockHash := sha512.New()
	blockHash.Write(root[:])
	for _, spec := range specs {
		blockHash.Write(spec.name[:])
	}
	var h []byte
	h = blockHash.Sum(h) // Appends hash to h
	copy(lock[:], h)
	return
}
