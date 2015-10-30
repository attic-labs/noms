package walk

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

// SomeCallback takes a ref.Ref and returns a bool indicating whether
// the current walk should skip the tree descending from value.
type SomeCallback func(r ref.Ref) bool

// AllCallback takes a ref and processes it.
type AllCallback func(r ref.Ref)

// Some recursively walks over all ref.Refs reachable from r and calls cb on them. If cb ever returns true, the walk will stop recursing on the current ref. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func SomeP(r ref.Ref, cs chunks.ChunkSource, cb SomeCallback, concurrency int) {
	doTreeWalkP(r, cs, cb, concurrency)
}

// All recursively walks over all ref.Refs reachable from r and calls cb on them. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func AllP(r ref.Ref, cs chunks.ChunkSource, cb AllCallback, concurrency int) {
	doTreeWalkP(r, cs, func(r ref.Ref) (skip bool) {
		cb(r)
		return
	}, concurrency)
}

func doTreeWalkP(r ref.Ref, cs chunks.ChunkSource, cb SomeCallback, concurrency int) {
	rq := newRefQueue()
	f := newFailure()

	visited := map[ref.Ref]bool{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	processRef := func(r ref.Ref) {
		defer wg.Done()

		mu.Lock()
		skip := cb(r) || visited[r]
		visited[r] = true
		mu.Unlock()

		if skip || f.didFail() {
			return
		}

		v := types.ReadValue(r, cs)
		if v == nil {
			f.fail(fmt.Errorf("Attempt to copy absent ref:%s", r.String()))
			return
		}

		for _, c := range v.Chunks() {
			wg.Add(1)
			rq.tail() <- c
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

	wg.Add(1)
	rq.tail() <- r
	wg.Wait()

	rq.close()

	f.checkNotFailed()
}

// refQueue emulates a buffered channel of refs of unlimited size.
type refQueue struct {
	head  func() <-chan ref.Ref
	tail  func() chan<- ref.Ref
	close func()
}

func newRefQueue() refQueue {
	head := make(chan ref.Ref, 64)
	tail := make(chan ref.Ref, 64)
	done := make(chan struct{})
	buff := []ref.Ref{}

	push := func(r ref.Ref) {
		buff = append(buff, r)
	}

	pop := func() ref.Ref {
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
					d.Chk.Equal(r, first)
				case <-done:
					break loop
				}
			}
		}
	}()

	return refQueue{
		func() <-chan ref.Ref {
			return head
		},
		func() chan<- ref.Ref {
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
