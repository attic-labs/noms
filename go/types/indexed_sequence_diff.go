// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

var (
	ChangeChanClosedErr = ChangeChannelClosedError{"Change channel closed"}
)

type ChangeChannelClosedError struct {
	msg string
}

func (e ChangeChannelClosedError) Error() string { return e.msg }

func indexedSequenceDiff(last indexedSequence, lastHeight int, lastOffset uint64, current indexedSequence, currentHeight int, currentOffset uint64, changes chan<- Splice, close <-chan struct{}, maxSpliceMatrixSize uint64) error {
	if lastHeight > currentHeight {
		lastChild := last.(indexedMetaSequence).getCompositeChildSequence(0, uint64(last.seqLen()))
		return indexedSequenceDiff(lastChild, lastHeight-1, lastOffset, current, currentHeight, currentOffset, changes, close, maxSpliceMatrixSize)
	}

	if currentHeight > lastHeight {
		currentChild := current.(indexedMetaSequence).getCompositeChildSequence(0, uint64(current.seqLen()))
		return indexedSequenceDiff(last, lastHeight, lastOffset, currentChild, currentHeight-1, currentOffset, changes, close, maxSpliceMatrixSize)
	}

	compareFn := last.getCompareFn(current)
	initialSplices := calcSplices(uint64(last.seqLen()), uint64(current.seqLen()), maxSpliceMatrixSize,
		func(i uint64, j uint64) bool { return compareFn(int(i), int(j)) })

	for _, splice := range initialSplices {
		if !isMetaSequence(last) || splice.SpRemoved == 0 || splice.SpAdded == 0 {
			splice.SpAt += lastOffset
			if splice.SpAdded > 0 {
				splice.SpFrom += currentOffset
			}

			select {
			case changes <- splice:
			case <-close:
				return ChangeChanClosedErr
			}

		} else {
			lastChild := last.(indexedMetaSequence).getCompositeChildSequence(splice.SpAt, splice.SpRemoved)
			currentChild := current.(indexedMetaSequence).getCompositeChildSequence(splice.SpFrom, splice.SpAdded)
			lastChildOffset := lastOffset
			if splice.SpAt > 0 {
				lastChildOffset += last.getOffset(int(splice.SpAt)-1) + 1
			}
			currentChildOffset := currentOffset
			if splice.SpFrom > 0 {
				currentChildOffset += current.getOffset(int(splice.SpFrom)-1) + 1
			}
			err := indexedSequenceDiff(lastChild, lastHeight-1, lastChildOffset, currentChild, currentHeight-1, currentChildOffset, changes, close, maxSpliceMatrixSize)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
