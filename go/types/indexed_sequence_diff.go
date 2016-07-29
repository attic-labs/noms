// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

func indexedSequenceDiff(last indexedSequence, lastHeight int, lastOffset uint64, current indexedSequence, currentHeight int, currentOffset uint64, changes chan<- Splice, closeChan <-chan struct{}, maxSpliceMatrixSize uint64) bool {
	if lastHeight > currentHeight {
		lastChild := last.(indexedMetaSequence).getCompositeChildSequence(0, uint64(last.seqLen())).(indexedSequence)
		return indexedSequenceDiff(lastChild, lastHeight-1, lastOffset, current, currentHeight, currentOffset, changes, closeChan, maxSpliceMatrixSize)
	}

	if currentHeight > lastHeight {
		currentChild := current.(indexedMetaSequence).getCompositeChildSequence(0, uint64(current.seqLen())).(indexedSequence)
		return indexedSequenceDiff(last, lastHeight, lastOffset, currentChild, currentHeight-1, currentOffset, changes, closeChan, maxSpliceMatrixSize)
	}

	compareFn := last.getCompareFn(current)
	initialSplices := calcSplices(uint64(last.seqLen()), uint64(current.seqLen()), maxSpliceMatrixSize,
		func(i uint64, j uint64) bool { return compareFn(int(i), int(j)) })

	for _, splice := range initialSplices {
		if !isMetaSequence(last) || splice.SpRemoved == 0 || splice.SpAdded == 0 {
			// We have meta data about the number of leaves below us, so if an entire meta sequence was removed, we don't need to dig down to compute the diff, we can just use math.
			lastAtCum := uint64(0)
			if splice.SpAt > 0 {
				lastAtCum = last.cumulativeNumberOfLeaves(int(splice.SpAt) - 1)
			}
			lastEndRemoveCum := uint64(0)
			if splice.SpAt+splice.SpRemoved > 0 {
				lastEndRemoveCum = last.cumulativeNumberOfLeaves(int(splice.SpAt+splice.SpRemoved) - 1)
			}
			currentFromCum := uint64(0)
			if splice.SpFrom > 0 {
				currentFromCum = current.cumulativeNumberOfLeaves(int(splice.SpFrom) - 1)
			}
			currentEndAddedCum := uint64(0)
			if splice.SpFrom+splice.SpAdded > 0 {
				currentEndAddedCum = current.cumulativeNumberOfLeaves(int(splice.SpFrom+splice.SpAdded) - 1)
			}

			splice.SpRemoved = lastEndRemoveCum - lastAtCum
			splice.SpAdded = currentEndAddedCum - currentFromCum
			splice.SpAt = lastOffset + lastAtCum
			if splice.SpAdded > 0 {
				splice.SpFrom = currentOffset + currentFromCum
			}

			select {
			case changes <- splice:
			case <-closeChan:
				return false
			}

		} else {
			lastChild := last.(indexedMetaSequence).getCompositeChildSequence(splice.SpAt, splice.SpRemoved).(indexedSequence)
			currentChild := current.(indexedMetaSequence).getCompositeChildSequence(splice.SpFrom, splice.SpAdded).(indexedSequence)
			lastChildOffset := lastOffset
			if splice.SpAt > 0 {
				lastChildOffset += last.cumulativeNumberOfLeaves(int(splice.SpAt) - 1)
			}
			currentChildOffset := currentOffset
			if splice.SpFrom > 0 {
				currentChildOffset += current.cumulativeNumberOfLeaves(int(splice.SpFrom) - 1)
			}
			if ok := indexedSequenceDiff(lastChild, lastHeight-1, lastChildOffset, currentChild, currentHeight-1, currentChildOffset, changes, closeChan, maxSpliceMatrixSize); !ok {
				return false
			}
		}
	}

	return true
}
