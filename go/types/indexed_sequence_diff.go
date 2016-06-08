// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"errors"
	"math"
)

const (
	DIFF_WITHOUT_LIMIT = math.MaxUint64
)

func maybeLoadCompositeSequence(ms indexedMetaSequence, idx uint64, length uint64, loadLimit uint64) (seq indexedSequence, newLoadLimit uint64, err error) {
	newLoadLimit = loadLimit
	if loadLimit > 0 && loadLimit != DIFF_WITHOUT_LIMIT {
		if length > newLoadLimit {
			return nil, 0, errors.New("load limit exceeded")
		}
		newLoadLimit -= length
	}
	return ms.getCompositeChildSequence(idx, length), newLoadLimit, nil
}

func indexedSequenceDiff(last indexedSequence, lastHeight int, lastOffset uint64,
	current indexedSequence, currentHeight int, currentOffset uint64,
	loadLimit uint64) (splices []Splice, numSequencesLoaded uint64, err error) {

	numSequencesLoaded = 0
	if lastHeight > currentHeight {
		seqLen := uint64(last.seqLen())
		lastChild, newLoadLimit, err := maybeLoadCompositeSequence(last.(indexedMetaSequence), 0, seqLen, loadLimit)
		if err != nil {
			return nil, 0, err
		}
		splices, nSeqLoaded, err := indexedSequenceDiff(lastChild, lastHeight-1, lastOffset, current, currentHeight, currentOffset, newLoadLimit)
		return splices, nSeqLoaded + seqLen, err
	}

	if currentHeight > lastHeight {
		seqLen := uint64(current.seqLen())
		currentChild, newLoadLimit, err := maybeLoadCompositeSequence(current.(indexedMetaSequence), 0, seqLen, loadLimit)
		if err != nil {
			return nil, 0, err
		}
		splices, nSeqLoaded, err := indexedSequenceDiff(last, lastHeight, lastOffset, currentChild, currentHeight-1, currentOffset, newLoadLimit)
		return splices, nSeqLoaded + seqLen, err
	}

	initialSplices := calcSplices(uint64(last.seqLen()), uint64(current.seqLen()),
		func(i uint64, j uint64) bool {
			return last.equalsAt(int(i), current.getItem(int(j)))
		})

	finalSplices := []Splice{}
	newLoadLimit := loadLimit
	for _, splice := range initialSplices {
		if !isMetaSequence(last) || splice.SpRemoved == 0 || splice.SpAdded == 0 {
			splice.SpAt += lastOffset
			if splice.SpAdded > 0 {
				splice.SpFrom += currentOffset
			}
			finalSplices = append(finalSplices, splice)
		} else {
			lastChild, lastLoadLimit, lastErr := maybeLoadCompositeSequence(last.(indexedMetaSequence), splice.SpAt, splice.SpRemoved, newLoadLimit)
			if lastErr != nil {
				return nil, numSequencesLoaded, lastErr
			}
			numSequencesLoaded += splice.SpRemoved
			newLoadLimit = lastLoadLimit
			currentChild, currentLoadLimit, currentErr := maybeLoadCompositeSequence(current.(indexedMetaSequence), splice.SpFrom, splice.SpAdded, newLoadLimit)
			if currentErr != nil {
				return nil, numSequencesLoaded, currentErr
			}
			numSequencesLoaded += splice.SpAdded
			newLoadLimit = currentLoadLimit
			lastChildOffset := lastOffset
			if splice.SpAt > 0 {
				lastChildOffset += last.getOffset(int(splice.SpAt)-1) + 1
			}
			currentChildOffset := currentOffset
			if splice.SpFrom > 0 {
				currentChildOffset += current.getOffset(int(splice.SpFrom)-1) + 1
			}
			childSplices, nSeqLoaded, err := indexedSequenceDiff(lastChild, lastHeight-1, lastChildOffset, currentChild, currentHeight-1, currentChildOffset, newLoadLimit)
			if err != nil {
				return nil, nSeqLoaded + numSequencesLoaded, err
			}
			numSequencesLoaded += nSeqLoaded
			finalSplices = append(finalSplices, childSplices...)
		}
	}

	return finalSplices, numSequencesLoaded, nil
}
