// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package functions

import "sync"

// All runs all functions in parallel, and returns when all functions have
// returned.
func All(fs ...func()) {
	wg := &sync.WaitGroup{}
	wg.Add(len(fs))
	for _, f := range fs {
		f := f
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}

// MaybeAll runs all functions in parallel, and returns when all functions have
// returned. If any function returns an error, returns that error, but not
// until all functions have returned.
func MaybeAll(fs ...func() error) (err error) {
	wg := &sync.WaitGroup{}
	wg.Add(len(fs))
	errMtx := sync.Mutex{}
	for _, f := range fs {
		f := f
		go func() {
			defer wg.Done()
			fErr := f()
			if fErr != nil && err == nil {
				errMtx.Lock()
				err = fErr
				errMtx.Unlock()
			}
		}()
	}
	wg.Wait()
	return
}
