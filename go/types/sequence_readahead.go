// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"github.com/attic-labs/noms/go/hash"
)

// sequenceReadAhead implements read-ahead by mapping a hash to a channel returning
// the corresponding sequence.
//
// It reads ahead by firing off a set of short-lived go routines. Each go
// routine (1) reads a sequence, (2) inserts it into a channel, (3) adds the
// channel to the map keyed by sequence hash, and (4) exits.
//
// The caller retrieves the sequence by looking up the channel by hash and
// reading from it.
//
// It maintains parallelism |p| by initially firing off |p| go routines
// to read the next |p| sequences. When a sequence is retreived from the cache,
// It fires off a new go-routine to read the next sequence. This ensures that
// there are always |p| outstanding channels to read from the cache.
//
// This approach has one major advantage over a channel based approach:
// there are no go-routines to shutdown when finished with the cursor. This
// avoids requiring caller to call a Close() method.
type sequenceReadAhead struct {
	cursor      *sequenceCursor
	cache       map[raKey]chan sequence
	parallelism int
	outstanding int
	getCount    float32
	hitCount    float32
}

// raKey is the future key. Rather than simply use the hash, we combines it
// with the local chunk offset. This increases the likelihood that repeat values
// in the sequence will get unique entries in the map.
type raKey struct {
	idx int
	hash hash.Hash
}

func newSequenceReadAhead(cursor *sequenceCursor, parallelism int) *sequenceReadAhead {
	m := map[raKey]chan sequence{}
	return &sequenceReadAhead{cursor.clone(), m, parallelism, 0, 0.0, 0.0}
}

func (ram *sequenceReadAhead) get(idx int, h hash.Hash) (sequence, bool) {
	ram.readAhead()
	key := raKey{idx,  h}
	ram.getCount += 1
	if future, ok := ram.cache[key]; ok {
		ram.outstanding -= 1
		result := <-future
		ram.hitCount += 1
		delete(ram.cache, key)
		return result, true
	}
	return nil, false
}

// readAhead (called when read-ahead is enabled) primes the next entries in the
// read-ahead cache. It ensures that go routines have been allocated for reading
// the next n entries in the current sequence. N is either readAheadParallelism
// or the number of entries left in the sequence if smaller.
func (ram *sequenceReadAhead) readAhead() {
	// the next position to be primed
	count := ram.parallelism - ram.outstanding
	for i := 0; i < count; i += 1 {
		ram.cursor.advance()
		if !ram.cursor.valid() {
			break
		}
		future := make(chan sequence, 1)
		key := raKey{
			ram.cursor.idx,
			ram.cursor.current().(metaTuple).ref.target,
		}
		ram.cache[key] = future
		seq := ram.cursor.seq
		idx := ram.cursor.idx
		go func() {
			defer close(future)
			val := seq.getChildSequence(idx)
			future <- val

		}()
		ram.outstanding += 1
	}
}

func (rc *sequenceReadAhead) hitRate() float32 {
	return rc.hitCount/rc.getCount
}



