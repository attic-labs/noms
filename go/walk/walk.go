// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package walk implements an API for iterating on Noms values.
package walk

import (
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

// SomeCallback takes a types.Value and returns a bool indicating whether the current walk should skip the tree descending from value. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type SomeCallback func(v types.Value, r *types.Ref) bool

// AllCallback takes a types.Value and processes it. If |v| is a top-level value in a Chunk, then |r| will be the Ref which referenced it (otherwise |r| is nil).
type AllCallback func(v types.Value, r *types.Ref)

// SomeP recursively walks over all types.Values reachable from r and calls cb on them. If cb ever returns true, the walk will stop recursing on the current ref. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func SomeP(v types.Value, vr types.ValueReader, cb SomeCallback) {
	doTreeWalkP(v, vr, cb, true)
}

// AllP recursively walks over all types.Values reachable from r and calls cb on them. If |concurrency| > 1, it is the callers responsibility to make ensure that |cb| is threadsafe.
func AllP(v types.Value, vr types.ValueReader, cb AllCallback) {
	doTreeWalkP(v, vr, func(v types.Value, r *types.Ref) (skip bool) {
		cb(v, r)
		return
	}, true)
}

func WalkRefs(target types.Value, vr types.ValueReader, cb types.RefCallback, deep bool) {
	doRefWalkP(target, vr, cb, deep)
}

func WalkValues(target types.Value, vr types.ValueReader, cb types.ValueCallback, deep bool) {
	callback := func(v types.Value, r *types.Ref) bool {
		if !target.Equals(v) {
			cb(v)
		}
		return false
	}
	doTreeWalkP(target, vr, callback, deep)
	return
}

func doTreeWalkP(v types.Value, vr types.ValueReader, cb SomeCallback, deep bool) {
	var processRef func(r types.Ref)
	var processVal func(v types.Value, r *types.Ref, next bool)
	visited := map[hash.Hash]bool{}

	valueCb := func(v types.Value) {
		processVal(v, nil, deep)
	}

	processVal = func(v types.Value, r *types.Ref, next bool) {
		if cb(v, r) || !next {
			return
		}

		if sr, ok := v.(types.Ref); ok {
			processRef(sr)
		} else {
			v.WalkValues(valueCb)
		}
	}

	processRef = func(r types.Ref) {

		target := r.TargetHash()
		if visited[target] {
			return
		}
		visited[target] = true
		v := vr.ReadValue(target)
		if v == nil {
			d.Chk.Fail("Attempt to visit absent ref:%s", target.String())
			return
		}

		if !deep {
			cb(v, &r)
			return
		}
		processVal(v, &r, deep)

<<<<<<< HEAD
=======
		// Try to avoid the cost of reading |c|. It's only necessary if the caller wants to know about every chunk, or if we need to descend below |c| (ref height > 1).
		var c chunks.Chunk

		if chunkCb != nil || r.Height() > 1 {
			c = bs.Get(tr)
			d.PanicIfTrue(c.IsEmpty())

			if chunkCb != nil {
				chunkCb(r, c)
			}
		}

		if r.Height() == 1 {
			return
		}

		v := types.DecodeValue(c, nil)
		for _, r1 := range v.Chunks() {
			wg.Add(1)
			rq.tail() <- r1
		}
>>>>>>> bc40db4fdb10233a0fa45fb870dbeefb43131660
	}
	//Process initial value
	processVal(v, nil, true)

}

<<<<<<< HEAD
func doRefWalkP(v types.Value, vr types.ValueReader, cb types.RefCallback, deep bool) {
	var processVal func(v types.Value, next bool)
	visited := map[hash.Hash]bool{}

	processVal = func(v types.Value, next bool) {
		if next {
			v.WalkRefs(func(ref types.Ref) {
				target := ref.TargetHash()
				if visited[target] {
					return
=======
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
		d.PanicIfFalse(len(buff) > 0)
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
					d.PanicIfFalse(r == first)
				case <-done:
					break loop
>>>>>>> bc40db4fdb10233a0fa45fb870dbeefb43131660
				}
				visited[target] = true

				if !deep {
					cb(ref)
					return
				}

				cb(ref)
				v := vr.ReadValue(target)
				processVal(v, deep)

			})
		}
	}

	processVal(v, true)

	//Process initial value
}
