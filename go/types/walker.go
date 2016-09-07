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
	DoTreeWalkP(v, vr, cb, concurrency, true)
}

// AllP recursively walks over all Values reachable from r and calls cb on them. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func AllP(v Value, vr ValueReader, cb AllCallback, concurrency int) {
	DoTreeWalkP(v, vr, func(v Value, r *Ref) (skip bool) {
		cb(v, r)
		return
	}, concurrency, true)
}

func WalkRefs(target Value, vr ValueReader, cb RefCallback, concurrency int, deep bool) {

	walkRefP(target, vr, cb, 1, deep)

}

func WalkValues(target Value, vr ValueReader, cb ValueCallback, concurrency int, deep bool) {
	callback := func(v Value, r *Ref) bool {
		if !target.Equals(v) {
			cb(v)
		}
		return false
	}
	DoTreeWalkP(target, vr, callback, concurrency, deep)
	return
}

func walkRefP(v Value, vr ValueReader, cb RefCallback, concurrency int, deep bool) {
	rq := newRefQueue()
	f := newFailure()

	visited := map[hash.Hash]bool{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	processVal := func(v Value, next bool) {
		if next {
			v.WalkRefs(func(ref *Ref) {
				wg.Add(1)
				rq.tail() <- *ref
			})
		}
	}

	processRef := func(r Ref) {
		defer wg.Done()

		mu.Lock()
		skip := visited[r.TargetHash()]
		visited[r.TargetHash()] = true
		mu.Unlock()

		if skip || f.didFail() {
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

	iter := func() {
		for r := range rq.head() {
			processRef(r)
		}
	}

	for i := 0; i < concurrency; i++ {
		go iter()
	}
	//Process initial value
	processVal(v, true)

	wg.Wait()

	rq.close()

	f.checkNotFailed()

}

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
		for _, r1 := range v.Chunks() {
			wg.Add(1)
			rq.tail() <- r1
		}
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
