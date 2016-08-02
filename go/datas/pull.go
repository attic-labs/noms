// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"sort"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

type PullProgress struct {
	DoneCount, KnownCount, DoneBytes uint64
}

// Pull objects that descends from sourceRef from srcDB to sinkDB. sinkHeadRef should point to a Commit (in sinkDB) that's an ancestor of sourceRef. This allows the algorithm to figure out which portions of data are already present in sinkDB and skip copying them.
func Pull(srcDB, sinkDB Database, sourceRef, sinkHeadRef types.Ref, concurrency int, progressCh chan PullProgress) {
	srcQ, sinkQ := &types.RefByHeight{sourceRef}, &types.RefByHeight{sinkHeadRef}

	// We generally expect that sourceRef descends from sinkHeadRef, so that walking down from sinkHeadRef yields useful hints. If it's not even in the srcDB, then just clear out sinkQ right now and don't bother.
	if !srcDB.has(sinkHeadRef.TargetHash()) {
		sinkQ.PopBack()
	}

	// Since we expect sourceRef to descend from sinkHeadRef, we assume srcDB has a superset of the data in sinkDB. There are some cases where, logically, the code wants to read data it knows to be in sinkDB. In this case, it doesn't actually matter which Database the data comes from, so as an optimization we use whichever is a LocalDatabase -- if either is.
	mostLocalDB := srcDB
	if _, ok := sinkDB.(*LocalDatabase); ok {
		mostLocalDB = sinkDB
	}
	// traverseWorker below takes refs off of {src,sink,com}Chan, processes them to figure out what reachable refs should be traversed, and then sends the results to {srcRes,sinkRes,comRes}Chan.
	// sending to (or closing) the 'done' channel causes traverseWorkers to exit.
	srcChan := make(chan types.Ref)
	sinkChan := make(chan types.Ref)
	comChan := make(chan types.Ref)
	srcResChan := make(chan traverseResult)
	sinkResChan := make(chan traverseResult)
	comResChan := make(chan traverseResult)
	done := make(chan struct{})

	workerWg := &sync.WaitGroup{}
	defer func() {
		close(done)
		workerWg.Wait()

		close(srcChan)
		close(sinkChan)
		close(comChan)
		close(srcResChan)
		close(sinkResChan)
		close(comResChan)
	}()
	traverseWorker := func() {
		workerWg.Add(1)
		go func() {
			for {
				select {
				case srcRef := <-srcChan:
					srcResChan <- traverseSource(srcRef, srcDB, sinkDB)
				case sinkRef := <-sinkChan:
					sinkResChan <- traverseSink(sinkRef, mostLocalDB)
				case comRef := <-comChan:
					comResChan <- traverseCommon(comRef, sinkHeadRef, mostLocalDB)
				case <-done:
					workerWg.Done()
					return
				}
			}
		}()
	}
	for i := 0; i < concurrency; i++ {
		traverseWorker()
	}

	var doneCount, knownCount, doneBytes uint64
	updateProgress := func(moreDone, moreKnown, moreBytes uint64) {
		if progressCh == nil {
			return
		}
		doneCount, knownCount, doneBytes = doneCount+moreDone, knownCount+moreKnown, doneBytes+moreBytes
		progressCh <- PullProgress{doneCount, knownCount + uint64(srcQ.Len()), doneBytes}
	}

	// hc and reachableChunks aren't goroutine-safe, so only write them here.
	hc := hintCache{}
	reachableChunks := hash.HashSet{}
	for !srcQ.Empty() {
		srcRefs, sinkRefs, comRefs := planWork(srcQ, sinkQ)
		srcWork, sinkWork, comWork := len(srcRefs), len(sinkRefs), len(comRefs)
		if srcWork+comWork > 0 {
			updateProgress(0, uint64(srcWork+comWork), 0)
		}

		// These goroutines send work to traverseWorkers, blocking when all are busy. They self-terminate when they've sent all they have.
		go sendWork(srcChan, srcRefs)
		go sendWork(sinkChan, sinkRefs)
		go sendWork(comChan, comRefs)
		//  Don't use srcRefs, sinkRefs, or comRefs after this point. The goroutines above own them.

		for srcWork+sinkWork+comWork > 0 {
			select {
			case res := <-srcResChan:
				for _, reachable := range res.reachables {
					srcQ.PushBack(reachable)
					reachableChunks.Insert(reachable.TargetHash())
				}
				if !res.readHash.IsEmpty() {
					reachableChunks.Remove(res.readHash)
				}
				srcWork--
				updateProgress(1, 0, uint64(res.readBytes))
			case res := <-sinkResChan:
				for _, reachable := range res.reachables {
					sinkQ.PushBack(reachable)
					hc[reachable.TargetHash()] = res.readHash
				}
				sinkWork--
			case res := <-comResChan:
				isHeadOfSink := res.readHash == sinkHeadRef.TargetHash()
				for _, reachable := range res.reachables {
					sinkQ.PushBack(reachable)
					if !isHeadOfSink {
						srcQ.PushBack(reachable)
					}
					hc[reachable.TargetHash()] = res.readHash
				}
				comWork--
				updateProgress(1, 0, uint64(res.readBytes))
			}
		}
		sort.Sort(sinkQ)
		sort.Sort(srcQ)
		sinkQ.Unique()
		srcQ.Unique()
	}

	hints := types.Hints{}
	for hash := range reachableChunks {
		if hint, present := hc[hash]; present {
			hints[hint] = struct{}{}
		}
	}
	sinkDB.validatingBatchStore().AddHints(hints)
}

type traverseResult struct {
	readHash   hash.Hash
	reachables types.RefSlice
	readBytes  int
}

// planWork deals with three possible situations:
// - head of srcQ is higher than head of sinkQ
// - head of sinkQ is higher than head of srcQ
// - both heads are at the same height
//
// As we build up lists of refs to be processed in parallel, we need to avoid blowing past potential common refs. When processing a given Ref, we enumerate Refs of all Chunks that are directly reachable, which must _by definition_ be shorter than the given Ref. This means that, for example, if the queues are the same height we know that nothing can happen that will put more Refs of that height on either queue. In general, if you look at the height of the Ref at the head of a queue, you know that all Refs of that height in the current graph under consideration are already in the queue. Conversely, for any height less than that of the head of the queue, it's possible that Refs of that height remain to be discovered. Given this, we can figure out which Refs are safe to pull off the 'taller' queue in the cases where the heights of the two queues are not equal.
// If one queue is 'taller' than the other, it's clear that we can process all refs from the taller queue with height greater than the height of the 'shorter' queue. We should also be able to process refs from the taller queue that are of the same height as the shorter queue, as long as we also check to see if they're common to both queues. It is not safe, however, to pull unique items off the shorter queue at this point. It's possible that, in processing some of the Refs from the taller queue, that these Refs will be discovered to be common after all.
// TODO: Bug 2203
func planWork(srcQ, sinkQ *types.RefByHeight) (srcRefs, sinkRefs, comRefs types.RefSlice) {
	srcHt, sinkHt := tallestHeight(srcQ), tallestHeight(sinkQ)
	if srcHt > sinkHt {
		srcRefs = popRefsOfHeight(srcQ, srcHt)
		return
	}
	if sinkHt > srcHt {
		sinkRefs = popRefsOfHeight(sinkQ, sinkHt)
		return
	}
	d.Chk.True(srcHt == sinkHt)
	srcRefs, comRefs = findCommon(srcQ, sinkQ, srcHt)
	sinkRefs = popRefsOfHeight(sinkQ, sinkHt)
	return
}

func popRefsOfHeight(q *types.RefByHeight, height uint64) (refs types.RefSlice) {
	for tallestHeight(q) == height {
		r := q.PopBack()
		refs = append(refs, r)
	}
	return
}

func findCommon(taller, shorter *types.RefByHeight, height uint64) (tallRefs, comRefs types.RefSlice) {
	d.Chk.True(tallestHeight(taller) == height)
	d.Chk.True(tallestHeight(shorter) == height)
	comIndices := []int{}
	// Walk through shorter and taller in tandem from the back (where the tallest Refs are). Refs from taller that go into a work queue are popped off directly, but doing so to shorter would mess up shortIdx. So, instead just keep track of the indices of common refs and drop them from shorter at the end.
	for shortIdx := shorter.Len() - 1; !taller.Empty() && tallestHeight(taller) == height; {
		tallPeek := taller.PeekEnd()
		shortPeek := shorter.PeekAt(shortIdx)
		if types.HeightOrder(tallPeek, shortPeek) {
			tallRefs = append(tallRefs, taller.PopBack())
			continue
		}
		if shortPeek.Equals(tallPeek) {
			comIndices = append(comIndices, shortIdx)
			comRefs = append(comRefs, taller.PopBack())
		}
		shortIdx--
	}
	shorter.DropIndices(comIndices)
	return
}

func tallestHeight(h *types.RefByHeight) uint64 {
	return h.PeekEnd().Height()
}

func sendWork(ch chan<- types.Ref, refs types.RefSlice) {
	for _, r := range refs {
		ch <- r
	}
}

type hintCache map[hash.Hash]hash.Hash

func traverseSource(srcRef types.Ref, srcDB, sinkDB Database) traverseResult {
	h := srcRef.TargetHash()
	if !sinkDB.has(h) {
		srcBS := srcDB.validatingBatchStore()
		c := srcBS.Get(h)
		v := types.DecodeValue(c, srcDB)
		d.Chk.True(v != nil, "Expected decoded chunk to be non-nil.")
		sinkDB.validatingBatchStore().SchedulePut(c, srcRef.Height(), types.Hints{})
		return traverseResult{h, v.Chunks(), len(c.Data())}
	}
	return traverseResult{}
}

func traverseSink(sinkRef types.Ref, db Database) traverseResult {
	if sinkRef.Height() > 1 {
		return traverseResult{sinkRef.TargetHash(), sinkRef.TargetValue(db).Chunks(), 0}
	}
	return traverseResult{}
}

func traverseCommon(comRef, sinkHead types.Ref, db Database) traverseResult {
	if comRef.Height() > 1 && isRefOfCommitType(comRef.Type()) {
		commit := comRef.TargetValue(db).(types.Struct)
		// We don't want to traverse the parents of sinkHead, but we still want to traverse its Value on the sinkDB side. We also still want to traverse all children, in both the srcDB and sinkDB, of any common Commit that is not at the Head of sinkDB.
		exclusionSet := types.NewSet()
		if comRef.Equals(sinkHead) {
			exclusionSet = commit.Get(ParentsField).(types.Set)
		}
		chunks := types.RefSlice(commit.Chunks())
		for i := 0; i < len(chunks); {
			if exclusionSet.Has(chunks[i]) {
				end := len(chunks) - 1
				chunks.Swap(i, end)
				chunks = chunks[:end]
				continue
			}
			i++
		}
		return traverseResult{comRef.TargetHash(), chunks, 0}
	}
	return traverseResult{}
}
