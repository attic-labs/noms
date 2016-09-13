// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package walk implements an API for iterating on Noms values.
package walk

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

// SomeCallback takes a types.Value and returns a bool indicating whether the current walk should skip the tree descending from value. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type SomeCallback func(v types.Value, r *types.Ref) bool

// AllCallback takes a types.Value and processes it. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type AllCallback func(v types.Value, r *types.Ref)

// SomeP recursively walks over all types.Values reachable from r and calls cb on them. If cb ever returns true, the walk will stop recursing on the current ref. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func SomeP(v types.Value, vr types.ValueReader, cb SomeCallback, concurrency int) {
	doTreeWalkP(v, vr, cb, concurrency, true)
}

// AllP recursively walks over all types.Values reachable from r and calls cb on them. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func AllP(v types.Value, vr types.ValueReader, cb AllCallback, concurrency int) {
	doTreeWalkP(v, vr, func(v types.Value, r *types.Ref) (skip bool) {
		cb(v, r)
		return
	}, concurrency, true)
}

func WalkRefs(target types.Value, vr types.ValueReader, cb types.RefCallback, concurrency int, deep bool) {
	doRefWalkP(target, vr, cb, concurrency, deep)
}

func WalkValues(target types.Value, vr types.ValueReader, cb types.ValueCallback, concurrency int, deep bool) {
	callback := func(v types.Value, r *types.Ref) bool {
		if !target.Equals(v) {
			cb(v)
		}
		return false
	}
	doTreeWalkP(target, vr, callback, concurrency, deep)
	return
}

func doTreeWalkP(v types.Value, vr types.ValueReader, cb SomeCallback, concurrency int, deep bool) {
	var processRef func(r types.Ref)
	var refWorkGroup refWorkP

	var processVal func(v types.Value, r *types.Ref, next bool)

	valueCb := func(v types.Value) {
		processVal(v, nil, deep)
	}

	processVal = func(v types.Value, r *types.Ref, next bool) {
		if cb(v, r) || !next {
			return
		}

		if sr, ok := v.(types.Ref); ok {
			refWorkGroup.addToWorkQueue(sr)
		} else {
			v.WalkValues(valueCb)
		}
	}

	processRef = func(r types.Ref) {

		if refWorkGroup.didProcessRef(r) || refWorkGroup.didFail() {
			return
		}

		target := r.TargetHash()
		v := vr.ReadValue(target)
		if v == nil {
			refWorkGroup.f.fail(fmt.Errorf("Attempt to visit absent ref:%s", target.String()))
			return
		}

		if !deep {
			cb(v, &r)
			return
		}
		processVal(v, &r, deep)

	}
	refWorkGroup = newRefWorkP(concurrency, processRef)
	//Process initial value
	refWorkGroup.start()
	processVal(v, nil, true)
	refWorkGroup.waitAndCleanup()

}

func doRefWalkP(v types.Value, vr types.ValueReader, cb types.RefCallback, concurrency int, deep bool) {
	var processRef func(r types.Ref)
	var refWorkGroup refWorkP

	processVal := func(v types.Value, next bool) {
		if next {
			v.WalkRefs(func(ref types.Ref) {
				refWorkGroup.addToWorkQueue(ref)
			})
		}
	}

	processRef = func(r types.Ref) {
		if refWorkGroup.didProcessRef(r) || refWorkGroup.didFail() {
			return
		}

		if !deep {
			cb(r)
			return
		}

		cb(r)
		target := r.TargetHash()
		v := vr.ReadValue(target)
		processVal(v, deep)

	}

	refWorkGroup = newRefWorkP(concurrency, processRef)
	refWorkGroup.start()
	//Process initial value
	processVal(v, true)
	refWorkGroup.waitAndCleanup()
}

// SomeChunksStopCallback is called for every unique types.Ref |r|. Return true to stop walking beyond |r|.
type SomeChunksStopCallback func(r types.Ref) bool

// SomeChunksChunkCallback is called for every unique chunks.Chunk |c| which wasn't stopped from SomeChunksStopCallback. |r| is a types.Ref referring to |c|.
type SomeChunksChunkCallback func(r types.Ref, c chunks.Chunk)

// SomeChunksP invokes callbacks on every unique chunk reachable from |r| in top-down order. Callbacks are invoked only once for each chunk regardless of how many times the chunk appears.
//
// |stopCb| is invoked for the types.Ref of every chunk. It can return true to stop SomeChunksP from descending any further.
// |chunkCb| is optional, invoked with the chunks.Chunk referenced by |stopCb| if it didn't return true.
func SomeChunksP(r types.Ref, bs types.BatchStore, stopCb SomeChunksStopCallback, chunkCb SomeChunksChunkCallback, concurrency int) {
	var processRef func(r types.Ref)
	var refWorkGroup refWorkP

	processVal := func(v types.Value) {
		v.WalkRefs(func(ref types.Ref) {
			refWorkGroup.addToWorkQueue(ref)
		})
	}

	processRef = func(r types.Ref) {
		if refWorkGroup.didProcessRef(r) || stopCb(r) {
			return
		}

		var c chunks.Chunk

		if chunkCb != nil || r.Height() > 1 {
			c = bs.Get(r.TargetHash())
			d.Chk.False(c.IsEmpty())

			if chunkCb != nil {
				chunkCb(r, c)
			}
		}

		if r.Height() == 1 {
			return
		}

		v := types.DecodeValue(c, nil)
		processVal(v)

	}

	refWorkGroup = newRefWorkP(concurrency, processRef)
	refWorkGroup.start()
	//Process initial value
	refWorkGroup.addToWorkQueue(r)
	refWorkGroup.waitAndCleanup()
}

// refQueue emulates a buffered channel of refs of unlimited size.
type refQueue struct {
	head  func() <-chan types.Ref
	tail  func() chan<- types.Ref
	close func()
}

func newRefQueue() refQueue {
	head := make(chan types.Ref, 64)
	tail := make(chan types.Ref, 64)
	done := make(chan struct{})
	buff := []types.Ref{}

	push := func(r types.Ref) {
		buff = append(buff, r)
	}

	pop := func() types.Ref {
		d.Chk.True(len(buff) > 0)
		r := buff[0]
		buff = buff[1:]
		return r
	}

	go func() {
	loop:
		for {
			if len(buff) == 0 {
				select {
				case r := <-tail:
					push(r)
				case <-done:
					break loop
				}
			} else {
				first := buff[0]
				select {
				case r := <-tail:
					push(r)
				case head <- first:
					r := pop()
					d.Chk.True(r == first)
				case <-done:
					break loop
				}
			}
		}
	}()

	return refQueue{
		func() <-chan types.Ref {
			return head
		},
		func() chan<- types.Ref {
			return tail
		},
		func() {
			close(head)
			done <- struct{}{}
		},
	}
}

type failure struct {
	err error
	mu  *sync.Mutex
}

func newFailure() *failure {
	return &failure{
		mu: &sync.Mutex{},
	}
}

func (f *failure) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err == nil { // only capture first error
		f.err = err
	}
}

func (f *failure) didFail() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err != nil
}

func (f *failure) checkNotFailed() {
	f.mu.Lock()
	defer f.mu.Unlock()
	d.Chk.NoError(f.err)
}

type refWorkP struct {
	rq          refQueue
	f           *failure
	concurrency int
	mu          sync.Mutex
	wg          sync.WaitGroup
	visited     map[hash.Hash]bool
	work        func(ref types.Ref)
}

func newRefWorkP(concurrency int, work func(ref types.Ref)) refWorkP {
	refWork := refWorkP{}
	refWork.rq = newRefQueue()
	refWork.f = newFailure()
	refWork.mu = sync.Mutex{}
	refWork.wg = sync.WaitGroup{}
	refWork.concurrency = concurrency
	refWork.visited = map[hash.Hash]bool{}
	refWork.work = work
	return refWork
}

func (r *refWorkP) addToWorkQueue(ref types.Ref) {
	r.wg.Add(1)
	r.rq.tail() <- ref
}

func (r *refWorkP) didProcessRef(ref types.Ref) bool {
	return r.visited[ref.TargetHash()]
}

func (r *refWorkP) didFail() bool {
	return r.f.didFail()
}

func (r *refWorkP) process(ref types.Ref) {
	defer r.wg.Done()
	r.work(ref)
	r.mu.Lock()
	r.visited[ref.TargetHash()] = true
	r.mu.Unlock()
}

func (r *refWorkP) start() {
	iter := func() {
		for ref := range r.rq.head() {
			r.process(ref)
		}
	}

	for i := 0; i < r.concurrency; i++ {
		go iter()
	}
}

//waitAndCleanup waits for the work to finish and closes the channel and finally fails if there was an error
func (r *refWorkP) waitAndCleanup() {
	r.wg.Wait()
	r.rq.close()
	r.f.checkNotFailed()
}
