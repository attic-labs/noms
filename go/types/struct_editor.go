// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

// StructEditor allows for efficient editing of Noms structs. Edits are
// buffered to memory and can be applied via Struct(), which returns a new
// Struct. Prior to Struct(), Get() will return the value that the resulting
// Struct would return if it were built immediately prior to the respective call.
// Note: The implementation biases performance towards a usage which applies
// edits in name-order.
type StructEditor struct {
	s              Struct
	edits          structEditSlice // edits may contain duplicate name values, in which case, the last edit of a given name is used
	normalized     bool
	estimatedCount int
}

// NewStructEditor returns a new StructEditor starting with the fields in s.
func NewStructEditor(s Struct) *StructEditor {
	return &StructEditor{s, structEditSlice{}, true, 0}
}

// Kind returns the kind of editor this is.
func (se *StructEditor) Kind() NomsKind {
	return StructKind
}

// Value builds a new Struct based on the edits done to the StructEditor.
func (se *StructEditor) Value() Value {
	return se.Struct()
}

// Struct builds a new Struct based on the edits done to the StructEditor.
func (se *StructEditor) Struct() Struct {
	if len(se.edits) == 0 {
		return se.s // no edits
	}

	se.normalize()

	kvsChan := make(chan chan structEntry)

	go func() {
		for i, edit := range se.edits {
			if i+1 < len(se.edits) && se.edits[i+1].name == edit.name {
				continue // next edit supercedes this one
			}

			edit := edit

			kvc := make(chan structEntry, 1)
			kvsChan <- kvc

			if edit.value == nil {
				kvc <- structEntry{edit.name, nil}
				continue
			}

			if v, ok := edit.value.(Value); ok {
				kvc <- structEntry{edit.name, v}
				continue
			}

			go func() {
				sv := edit.value.Value()
				kvc <- structEntry{edit.name, sv}
			}()
		}

		close(kvsChan)
	}()

	entries := se.s.structEntries()
	w := newStructBinaryNomsWriter(se.s.Name(), len(entries)+se.estimatedCount)

	i := 0
	for sec := range kvsChan {
		se := <-sec
		for ; i < len(entries) && se.name > entries[i].name; i++ {
			w.writeFieldRaw(entries[i].name, entries[i].buff)
		}

		if se.value != nil {
			w.writeField(se.name, se.value)
		}

		if i < len(entries) && entries[i].name == se.name {
			i++
		}
	}

	for ; i < len(entries); i++ {
		w.writeFieldRaw(entries[i].name, entries[i].buff)
	}

	return Struct{se.s.vrw, w.close()}
}

// Set updates an existing field or creates a new field in the resulting
// Struct.
func (se *StructEditor) Set(n string, v Valuable) *StructEditor {
	d.PanicIfTrue(v == nil)
	se.set(n, v)
	se.estimatedCount++
	return se
}

// Delete removes a field in the resulting Struct. Deleting a non existing
// field is a no op.
func (se *StructEditor) Delete(k string) *StructEditor {
	se.set(k, nil)
	se.estimatedCount--
	return se
}

// Get returns the Valuable for the field with the name n.
func (se *StructEditor) Get(n string) Valuable {
	if idx, found := se.findEdit(n); found {
		v := se.edits[idx].value
		if v != nil {
			return v
		}
	}

	return se.s.Get(n)
}

func (se *StructEditor) set(n string, v Valuable) {
	if len(se.edits) == 0 {
		se.edits = append(se.edits, structEdit{n, v})
		return
	}

	final := se.edits[len(se.edits)-1]
	if final.name == n {
		se.edits[len(se.edits)-1] = structEdit{n, v}
		return // update the last edit
	}

	se.edits = append(se.edits, structEdit{n, v})

	if se.normalized && final.name < n {
		// fast-path: edits take place in name-order
		return
	}

	// de-normalize
	se.normalized = false
}

// Find the edit position of the last edit for a given name.
func (se *StructEditor) findEdit(n string) (idx int, found bool) {
	se.normalize()

	idx = sort.Search(len(se.edits), func(i int) bool {
		return se.edits[i].name >= n
	})

	if idx == len(se.edits) {
		return
	}

	if se.edits[idx].name != n {
		return
	}

	// advance to final edit position where nv.name == n
	for idx < len(se.edits) && se.edits[idx].name == n {
		idx++
	}
	idx--

	found = true
	return
}

func (se *StructEditor) normalize() {
	if se.normalized {
		return
	}

	sort.Stable(se.edits)
	// TODO: GC duplicate names over some threshold of collectable memory?
	se.normalized = true
}

type structEntry struct {
	name  string
	value Value
}

type structEdit struct {
	name  string
	value Valuable
}

type structEditSlice []structEdit

func (mes structEditSlice) Len() int           { return len(mes) }
func (mes structEditSlice) Swap(i, j int)      { mes[i], mes[j] = mes[j], mes[i] }
func (mes structEditSlice) Less(i, j int) bool { return mes[i].name < mes[j].name }
