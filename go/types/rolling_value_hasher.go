// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sync"

	"github.com/attic-labs/noms/go/sloppy"

	"github.com/kch42/buzhash"
)

const (
	defaultChunkPattern = uint32(1<<12 - 1) // Avg Chunk Size of 4k

	// The choice of hash window here is a trade off between:
	//   -When mutating a prolly tree, the likelihood that a change within a node
	//    "cascades" by affecting the chunk boundary and causing a following node
	//    to change.
	//   -The chance that repeated sequences of bytes which happen to be longer
	//    than the window do not chunk and cause average chunk size to be way
	//    larger than the target
	// The most likely source of repeated sequences larger than the chunk window
	// is structs, all of whose field names are encoded together. A window size
	// of 256 gives a roughly 6% chance that a change within a chunk will cascade
	// into the next chunk while being sufficiently large to be larger than the
	// field encodings of a "reasonable" struct.
	defaultChunkWindow = uint32(256)

	defaultCompressionWindow = uint16(1<<12) - 1 // 4k
)

// Only set by tests
var (
	chunkPattern  = defaultChunkPattern
	chunkWindow   = defaultChunkWindow
	chunkConfigMu = &sync.Mutex{}
)

func chunkingConfig() (pattern, window uint32) {
	chunkConfigMu.Lock()
	defer chunkConfigMu.Unlock()
	return chunkPattern, chunkWindow
}

func smallTestChunks() {
	chunkConfigMu.Lock()
	defer chunkConfigMu.Unlock()
	chunkPattern = uint32(1<<8 - 1) // Avg Chunk Size of 256 bytes
	chunkWindow = uint32(64)
}

func normalProductionChunks() {
	chunkConfigMu.Lock()
	defer chunkConfigMu.Unlock()
	chunkPattern = defaultChunkPattern
	chunkWindow = defaultChunkWindow
}

type rollingValueHasher struct {
	bw              *binaryNomsWriter
	bz              *buzhash.BuzHash
	enc             *valueEncoder
	crossedBoundary bool
	pattern, window uint32
	salt            byte
	sl              *sloppy.Sloppy
}

func hashValueBytes(item sequenceItem, rv *rollingValueHasher) {
	rv.HashValue(item.(Value))
}

func hashValueByte(item sequenceItem, rv *rollingValueHasher) {
	rv.HashByte(item.(byte))
}

func newRollingValueHasher(salt byte) *rollingValueHasher {
	pattern, window := chunkingConfig()
	bw := newBinaryNomsWriter()
	enc := newValueEncoder(bw, nil)

	rv := &rollingValueHasher{
		bw:      bw,
		enc:     enc,
		bz:      buzhash.NewBuzHash(window),
		pattern: pattern,
		window:  window,
		salt:    salt,
	}

	rv.sl = sloppy.New(func(b byte) bool {
		return rv.HashByte(b)
	}, defaultCompressionWindow)

	return rv
}

func (rv *rollingValueHasher) HashByte(b byte) bool {
	if !rv.crossedBoundary {
		rv.bz.HashByte(b ^ rv.salt)
		rv.crossedBoundary = (rv.bz.Sum32()&rv.pattern == rv.pattern)
	}
	return rv.crossedBoundary
}

func (rv *rollingValueHasher) Reset() {
	rv.crossedBoundary = false
	rv.bz = buzhash.NewBuzHash(rv.window)
	rv.bw.reset()
	rv.sl.Reset()
}

func (rv *rollingValueHasher) HashValue(v Value) {
	rv.enc.writeValue(v)
	rv.sl.Update(rv.bw.data())
}
