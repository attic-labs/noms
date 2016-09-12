// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package walk implements an API for iterating on Noms values.
package types

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

// SomeCallback takes a Value and returns a bool indicating whether the current walk should skip the tree descending from value. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type SomeCallback func(v Value, r *Ref) bool

// AllCallback takes a Value and processes it. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type AllCallback func(v Value, r *Ref)

type ValueCallback func(v Value)
type RefCallback func(r *Ref)

// SomeP recursively walks over all Values reachable from r and calls cb on them. If cb ever returns true, the walk will stop recursing on the current ref. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func SomeP(v Value, vr ValueReader, cb SomeCallback, concurrency int) {
	walkTreeP(v, vr, cb, concurrency, true)
}

// AllP recursively walks over all Values reachable from r and calls cb on them. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func AllP(v Value, vr ValueReader, cb AllCallback, concurrency int) {
	walkTreeP(v, vr, func(v Value, r *Ref) (skip bool) {
		cb(v, r)
		return
	}, concurrency, true)
}

func WalkRefs(target Value, vr ValueReader, cb RefCallback, concurrency int, deep bool) {

	walkRefP(target, vr, cb, concurrency, deep)

}

func WalkValues(target Value, vr ValueReader, cb ValueCallback, concurrency int, deep bool) {
	callback := func(v Value, r *Ref) bool {
		if !target.Equals(v) {
			cb(v)
		}
		return false
	}
	walkTreeP(target, vr, callback, concurrency, deep)
	return
}

type refWorkP struct {
	rq          refQueue
	f           *failure
	concurrency int
	mu          sync.Mutex
	wg          sync.WaitGroup
	visited     map[hash.Hash]bool
	work        func(ref Ref)
}

func newRefWorkP(concurrency int, work func(ref Ref)) refWorkP {
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

func (r *refWorkP) addToWorkQueue(ref Ref) {
	r.wg.Add(1)
	r.rq.tail() <- ref
}

func (r *refWorkP) didProcessRef(ref Ref) bool {
	return r.visited[ref.TargetHash()]
}

func (r *refWorkP) didFail() bool {
	return r.f.didFail()
}

func (r *refWorkP) process(ref Ref) {
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

//addToQueue()
//setRefWorker(func)
//didProcessRef(r Ref))
//didFail()
//cleanup()
//start()

func walkRefP(v Value, vr ValueReader, cb RefCallback, concurrency int, deep bool) {
	var processRef func(r Ref)
	var refWorkGroup refWorkP

	processVal := func(v Value, next bool) {
		if next {
			v.WalkRefs(func(ref *Ref) {
				refWorkGroup.addToWorkQueue(*ref)
			})
		}
	}

	processRef = func(r Ref) {
		if refWorkGroup.didProcessRef(r) || refWorkGroup.didFail() {
			return
		}

		if !deep {
			cb(&r)
			return
		} else {
			cb(&r)
			target := r.TargetHash()
			v := vr.ReadValue(target)
			processVal(v, deep)
		}

	}

	refWorkGroup = newRefWorkP(concurrency, processRef)
	refWorkGroup.start()
	//Process initial value
	processVal(v, true)
	refWorkGroup.waitAndCleanup()
}

func walkTreeP(v Value, vr ValueReader, cb SomeCallback, concurrency int, deep bool) {
	var processRef func(r Ref)
	var refWorkGroup refWorkP

	var processVal func(v Value, r *Ref, next bool)

	valueCb := func(v Value) {
		processVal(v, nil, deep)
	}

	processVal = func(v Value, r *Ref, next bool) {
		if cb(v, r) || !next {
			return
		}

		if sr, ok := v.(Ref); ok {
			refWorkGroup.addToWorkQueue(sr)
		} else {
			v.WalkValues(valueCb)
		}
	}
	processRef = func(r Ref) {

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
		} else {
			processVal(v, &r, deep)
		}

	}
	refWorkGroup = newRefWorkP(concurrency, processRef)
	//Process initial value
	refWorkGroup.start()
	processVal(v, nil, true)
	refWorkGroup.waitAndCleanup()

}

//addToWorkQueue
//processWork
//paralellism
//initialize

//initialize()
func DoTreeWalkP(v Value, vr ValueReader, cb SomeCallback, concurrency int, deep bool) {
	rq := newRefQueue()
	f := newFailure()

	visited := map[hash.Hash]bool{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	fmt.Println("head item processing this: ", KindToString[v.Type().Kind()])

	var processVal func(v Value, r *Ref, next bool)

	valueCb := func(v Value) {
		processVal(v, nil, deep)
	}

	processVal = func(v Value, r *Ref, next bool) {
		if cb(v, r) || !next {
			return
		}

		if sr, ok := v.(Ref); ok {
			wg.Add(1)
			rq.tail() <- sr
		} else {
			v.WalkValues(valueCb)

		}
	}

	processRef := func(r Ref) {
		defer wg.Done()
		fmt.Println("found ref")

		mu.Lock()
		skip := visited[r.TargetHash()]
		visited[r.TargetHash()] = true
		mu.Unlock()

		if skip || f.didFail() {
			return
		}

		target := r.TargetHash()
		v := vr.ReadValue(target)
		if v == nil {
			f.fail(fmt.Errorf("Attempt to visit absent ref:%s", target.String()))
			return
		}

		if !deep {
			cb(v, &r)
			return
		} else {
			processVal(v, &r, deep)
		}

	}

	iter := func() {
		for r := range rq.head() {
			processRef(r)
		}
	}

	for i := 0; i < concurrency; i++ {
		go iter()
	}

	processVal(v, nil, true)
	wg.Wait()

	rq.close()

	f.checkNotFailed()
}

// SomeChunksStopCallback is called for every unique Ref |r|. Return true to stop walking beyond |r|.
type SomeChunksStopCallback func(r Ref) bool

// SomeChunksChunkCallback is called for every unique chunks.Chunk |c| which wasn't stopped from SomeChunksStopCallback. |r| is a Ref referring to |c|.
type SomeChunksChunkCallback func(r Ref, c chunks.Chunk)

// SomeChunksP invokes callbacks on every unique chunk reachable from |r| in top-down order. Callbacks are invoked only once for each chunk regardless of how many times the chunk appears.
//
// |stopCb| is invoked for the Ref of every chunk. It can return true to stop SomeChunksP from descending any further.
// |chunkCb| is optional, invoked with the chunks.Chunk referenced by |stopCb| if it didn't return true.
func SomeChunksP(r Ref, bs BatchStore, stopCb SomeChunksStopCallback, chunkCb SomeChunksChunkCallback, concurrency int) {
	rq := newRefQueue()
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	visitedRefs := map[hash.Hash]bool{}

	addToChunkQueue := func(r *Ref) {
		wg.Add(1)
		rq.tail() <- *r
	}

	walkChunk := func(r Ref) {
		defer wg.Done()

		tr := r.TargetHash()

		mu.Lock()
		visited := visitedRefs[tr]
		visitedRefs[tr] = true
		mu.Unlock()

		if visited || stopCb(r) {
			return
		}

		// Try to avoid the cost of reading |c|. It's only necessary if the caller wants to know about every chunk, or if we need to descend below |c| (ref height > 1).
		var c chunks.Chunk

		if chunkCb != nil || r.Height() > 1 {
			c = bs.Get(tr)
			d.Chk.False(c.IsEmpty())

			if chunkCb != nil {
				chunkCb(r, c)
			}
		}

		if r.Height() == 1 {
			return
		}

		v := DecodeValue(c, nil)
		v.WalkRefs(addToChunkQueue)
	}

	iter := func() {
		for r := range rq.head() {
			walkChunk(r)
		}
	}

	for i := 0; i < concurrency; i++ {
		go iter()
	}

	wg.Add(1)
	rq.tail() <- r
	wg.Wait()
	rq.close()
}

// refQueue emulates a buffered channel of refs of unlimited size.
type refQueue struct {
	head  func() <-chan Ref
	tail  func() chan<- Ref
	close func()
}

func newRefQueue() refQueue {
	head := make(chan Ref, 64)
	tail := make(chan Ref, 64)
	done := make(chan struct{})
	buff := []Ref{}

	push := func(r Ref) {
		buff = append(buff, r)
	}

	pop := func() Ref {
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
		func() <-chan Ref {
			return head
		},
		func() chan<- Ref {
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
