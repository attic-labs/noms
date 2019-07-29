// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "gopkg.in/attic-labs/noms.v7/go/hash"

type SkipValueCallback func(v Value) bool

// WalkValues loads prolly trees progressively by walking down the tree. We don't wants to invoke
// the value callback on internal sub-trees (which are valid values) because they are not logical
// values in the graph
type valueRec struct {
	v  Value
	cb bool
}

const maxRefCount = 1 << 12 // ~16MB of data

// WalkValues recursively walks over all types. Values reachable from r and calls cb on them.
func WalkValues(target Value, vr ValueReader, cb SkipValueCallback) {
	visited := hash.HashSet{}
	refs := map[hash.Hash]bool{}
	values := []valueRec{{target, true}}

	for len(values) > 0 || len(refs) > 0 {
		for len(values) > 0 {
			rec := values[len(values)-1]
			values = values[:len(values)-1]

			v := rec.v
			if rec.cb && cb(v) {
				continue
			}

			if _, ok := v.(Blob); ok {
				continue // don't traverse into blob ptrees
			}

			if r, ok := v.(Ref); ok {
				refs[r.TargetHash()] = true
				continue
			}

			if col, ok := v.(Collection); ok && !col.sequence().isLeaf() {
				ms := col.sequence().(metaSequence)
				for _, mt := range ms.tuples {
					if mt.child != nil {
						values = append(values, valueRec{mt.child, false})
					} else {
						refs[mt.ref.TargetHash()] = false
					}
				}
				continue
			}

			v.WalkValues(func(sv Value) {
				values = append(values, valueRec{sv, true})
			})
		}

		if len(refs) == 0 {
			continue
		}

		hs := hash.HashSet{}
		oldRefs := refs
		refs = map[hash.Hash]bool{}
		for h := range oldRefs {
			if _, ok := visited[h]; ok {
				continue
			}

			if len(hs) >= maxRefCount {
				refs[h] = oldRefs[h]
				continue
			}

			hs.Insert(h)
			visited.Insert(h)
		}

		if len(hs) > 0 {
			valueChan := make(chan Value, len(hs))
			vr.ReadManyValues(hs, valueChan)
			close(valueChan)
			for sv := range valueChan {
				values = append(values, valueRec{sv, oldRefs[sv.Hash()]})
			}
		}
	}
}

func mightContainStructs(t *Type) (mightHaveStructs bool) {
	if t.TargetKind() == StructKind || t.TargetKind() == ValueKind {
		mightHaveStructs = true
		return
	}

	t.WalkValues(func(v Value) {
		mightHaveStructs = mightHaveStructs || mightContainStructs(v.(*Type))
	})

	return
}
