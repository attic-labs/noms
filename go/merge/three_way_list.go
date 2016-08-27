// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"fmt"

	"github.com/attic-labs/noms/go/types"
)

func threeWayListMerge(a, b, parent types.List) (merged types.List, err error) {
	aSpliceChan, bSpliceChan := make(chan types.Splice), make(chan types.Splice)
	aStopChan, bStopChan := make(chan struct{}, 1), make(chan struct{}, 1)

	go func() {
		a.Diff(parent, aSpliceChan, aStopChan)
		close(aSpliceChan)
	}()
	go func() {
		b.Diff(parent, bSpliceChan, bStopChan)
		close(bSpliceChan)
	}()

	stopAndDrain := func(stop chan<- struct{}, drain <-chan types.Splice) {
		close(stop)
		for range drain {
		}
	}

	defer stopAndDrain(aStopChan, aSpliceChan)
	defer stopAndDrain(bStopChan, bSpliceChan)

	zeroSplice := types.Splice{}
	zeroToEmpty := func(sp types.Splice) types.Splice {
		if sp == zeroSplice {
			return types.Splice{types.SPLICE_UNASSIGNED, types.SPLICE_UNASSIGNED, types.SPLICE_UNASSIGNED, types.SPLICE_UNASSIGNED}
		}
		return sp
	}

	merged = parent
	offset := uint64(0)
	emptySplice := zeroToEmpty(types.Splice{})
	aSplice, bSplice := emptySplice, emptySplice
	for {
		// Get the next splice from both a and b. If either diff(a, parent) or diff(b, parent) is complete, aSplice or bSplice will get an empty types.Splice. Generally, though, this allows us to proceed through both diffs in (key) order, considering the "current" splice from both diffs at the same time.
		if aSplice == emptySplice {
			aSplice = zeroToEmpty(<-aSpliceChan)
		}
		if bSplice == emptySplice {
			bSplice = zeroToEmpty(<-bSpliceChan)
		}
		// Both channels are producing zero values, so we're done.
		if aSplice == emptySplice && bSplice == emptySplice {
			break
		}
		if overlap(aSplice, bSplice) {
			if canMerge(a, b, aSplice, bSplice) {
				splice := merge(aSplice, bSplice)
				merged = apply(a, merged, offset, splice)
				offset += splice.SpAdded - splice.SpRemoved
				aSplice, bSplice = emptySplice, emptySplice
				continue
			}
			return parent, newMergeConflict("Overlapping splices: %s vs %s", describeSplice(aSplice), describeSplice(bSplice))
		}
		if aSplice.SpAt < bSplice.SpAt {
			merged = apply(a, merged, offset, aSplice)
			offset += aSplice.SpAdded - aSplice.SpRemoved
			aSplice = emptySplice
			continue
		}
		merged = apply(b, merged, offset, bSplice)
		offset += bSplice.SpAdded - bSplice.SpRemoved
		bSplice = emptySplice
	}

	return merged, nil
}

func overlap(s1, s2 types.Splice) bool {
	earlier, later := s1, s2
	if s2.SpAt < s1.SpAt {
		earlier, later = s2, s1
	}
	return s1.SpAt == s2.SpAt || earlier.SpAt+earlier.SpRemoved > later.SpAt
}

func canMerge(a, b types.List, aSplice, bSplice types.Splice) bool {
	if aSplice != bSplice {
		return false
	}
	// TODO: Add List.IterFrom() and use that to compare added values.
	for i := uint64(0); i < aSplice.SpAdded; i++ {
		if !a.Get(aSplice.SpFrom + i).Equals(b.Get(bSplice.SpFrom + i)) {
			return false
		}
	}
	return true
}

func merge(s1, s2 types.Splice) types.Splice {
	return s1
}

func apply(source, target types.List, offset uint64, s types.Splice) types.List {
	// TODO: Add List.IterFrom() and use that to build up toAdd.
	toAdd := make(types.ValueSlice, s.SpAdded)
	for i := range toAdd {
		toAdd[i] = source.Get(s.SpFrom + uint64(i))
	}
	return target.Splice(s.SpAt+offset, s.SpRemoved, toAdd...)
}

func describeSplice(s types.Splice) string {
	return fmt.Sprintf("%d elements removed at %d; adding %d elements", s.SpRemoved, s.SpAt, s.SpAdded)
}
