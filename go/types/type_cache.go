// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

type TypeCache struct {
	identTable *identTable
	trieRoots  map[NomsKind]*typeTrie
	typeBytes  map[uint32][]byte
	nextId     uint32
	mu         *sync.Mutex
}

var staticTypeCache = NewTypeCache()

var BoolType = makePrimitiveType(BoolKind)
var NumberType = makePrimitiveType(NumberKind)
var StringType = makePrimitiveType(StringKind)
var BlobType = makePrimitiveType(BlobKind)
var TypeType = makePrimitiveType(TypeKind)
var ValueType = makePrimitiveType(ValueKind)

func NewTypeCache() *TypeCache {
	return &TypeCache{
		newIdentTable(),
		map[NomsKind]*typeTrie{
			ListKind:   newTypeTrie(),
			SetKind:    newTypeTrie(),
			RefKind:    newTypeTrie(),
			MapKind:    newTypeTrie(),
			StructKind: newTypeTrie(),
			CycleKind:  newTypeTrie(),
			UnionKind:  newTypeTrie(),
		},
		map[uint32][]byte{},
		256, // The first 255 type ids are reserved for the 8bit space of NomsKinds.
		&sync.Mutex{},
	}
}

func (tc *TypeCache) Lock() {
	tc.mu.Lock()
}

func (tc *TypeCache) Unlock() {
	tc.mu.Unlock()
}

func (tc *TypeCache) nextTypeId() uint32 {
	next := tc.nextId
	tc.nextId++
	return next
}

func (tc *TypeCache) getCompoundType(kind NomsKind, elemTypes ...*Type) *Type {
	d.Chk.NotNil(tc)
	trie := tc.trieRoots[kind]
	for i := 0; i < len(elemTypes); i++ {
		trie = trie.Traverse(elemTypes[i].id)
	}

	if trie.t == nil {
		trie.t = newType(CompoundDesc{kind, elemTypes}, tc.nextTypeId())
	}

	return trie.t
}

func (tc *TypeCache) makeStructType(name string, fields map[string]*Type) *Type {
	verifyStructName(name)
	fs := make(fieldSlice, len(fields))
	i := 0
	for fn, t := range fields {
		verifyFieldName(fn)
		fs[i] = field{fn, t}
		i++
	}

	sort.Sort(fs)

	trie := tc.trieRoots[StructKind].Traverse(tc.identTable.GetId(name))
	for i := 0; i < len(fs); i++ {
		trie = trie.Traverse(tc.identTable.GetId(fs[i].name))
		trie = trie.Traverse(fs[i].t.id)
	}

	if trie.t == nil {
		t := newType(StructDesc{name, fs}, 0)
		if t.serialization == nil {
			// HasUnresolvedCycle
			t, _ = toUnresolvedType(t, tc, -1, nil)
			resolveStructCycles(t, nil)
			if !t.HasUnresolvedCycle() {
				ensureSerialization(t, nil)
			}
		}
		t.id = tc.nextTypeId()
		trie.t = t
	}

	return trie.t
}

func indexOfType(t *Type, tl []*Type) (uint32, bool) {
	for i, tt := range tl {
		if tt == t {
			return uint32(i), true
		}
	}
	return 0, false
}

// Returns a new type where cyclic pointer references are replaced with Cycle<N> types.
func toUnresolvedType(t *Type, tc *TypeCache, level int, parentStructTypes []*Type) (*Type, bool) {
	i, found := indexOfType(t, parentStructTypes)
	if found {
		cycle := CycleDesc(uint32(len(parentStructTypes)) - i - 1)
		return &Type{cycle, &hash.Hash{}, 0, nil}, true // This type is just a placeholder. It doesn't need an id
	}

	switch desc := t.Desc.(type) {
	case CompoundDesc:
		ts := make(typeSlice, len(desc.ElemTypes))
		didChange := false
		for i, et := range desc.ElemTypes {
			st, changed := toUnresolvedType(et, tc, level, parentStructTypes)
			ts[i] = st
			didChange = didChange || changed
		}

		if !didChange {
			return t, false
		}

		return &Type{CompoundDesc{t.Kind(), ts}, &hash.Hash{}, tc.nextTypeId(), nil}, true
	case StructDesc:
		fs := make(fieldSlice, len(desc.fields))
		didChange := false
		for i, f := range desc.fields {
			fs[i].name = f.name
			st, changed := toUnresolvedType(f.t, tc, level+1, append(parentStructTypes, t))
			fs[i].t = st
			didChange = didChange || changed
		}

		if !didChange {
			return t, false
		}

		return &Type{StructDesc{desc.Name, fs}, &hash.Hash{}, tc.nextTypeId(), nil}, true
	case CycleDesc:
		cycleLevel := int(desc)
		return t, cycleLevel <= level // Only cycles which can be resolved in the current struct.
	}

	return t, false
}

// Drops cycles and replaces them with pointers to parent structs
func resolveStructCycles(t *Type, parentStructTypes []*Type) *Type {
	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for i, et := range desc.ElemTypes {
			desc.ElemTypes[i] = resolveStructCycles(et, parentStructTypes)
		}

	case StructDesc:
		for i, f := range desc.fields {
			desc.fields[i].t = resolveStructCycles(f.t, append(parentStructTypes, t))
		}

	case CycleDesc:
		idx := int(desc)
		if idx < len(parentStructTypes) {
			return parentStructTypes[len(parentStructTypes)-1-int(desc)]
		}
	}

	return t
}

// Traverses a fully resolved cyclic type and ensures all types have serializations
func ensureSerialization(t *Type, parentStructTypes []*Type) {
	_, found := indexOfType(t, parentStructTypes)
	if found {
		return
	}

	d.Chk.True(t.id != 0)
	if t.serialization == nil {
		serializeType(t)
	}

	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for _, et := range desc.ElemTypes {
			ensureSerialization(et, parentStructTypes)
		}

	case StructDesc:
		for _, f := range desc.fields {
			ensureSerialization(f.t, append(parentStructTypes, t))
		}
	}
}

// MakeUnionType creates a new union type unless the elemTypes can be folded into a single non union type.
func (tc *TypeCache) makeUnionType(elemTypes ...*Type) *Type {
	seenTypes := map[hash.Hash]bool{}
	ts := flattenUnionTypes(typeSlice(elemTypes), &seenTypes)
	if len(ts) == 1 {
		return ts[0]
	}

	sort.Sort(ts)
	return tc.getCompoundType(UnionKind, ts...)
}

func (tc *TypeCache) getCyclicType(level uint32) *Type {
	trie := tc.trieRoots[CycleKind].Traverse(level)

	if trie.t == nil {
		trie.t = newType(CycleDesc(level), tc.nextTypeId())
	}

	return trie.t
}

func flattenUnionTypes(ts typeSlice, seenTypes *map[hash.Hash]bool) typeSlice {
	if len(ts) == 0 {
		return ts
	}

	ts2 := make(typeSlice, 0, len(ts))
	for _, t := range ts {
		if t.Kind() == UnionKind {
			ts2 = append(ts2, flattenUnionTypes(t.Desc.(CompoundDesc).ElemTypes, seenTypes)...)
		} else {
			if !(*seenTypes)[t.Hash()] {
				(*seenTypes)[t.Hash()] = true
				ts2 = append(ts2, t)
			}
		}
	}
	return ts2
}

func MakeListType(elemType *Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.getCompoundType(ListKind, elemType)
}

func MakeSetType(elemType *Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.getCompoundType(SetKind, elemType)
}

func MakeRefType(elemType *Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.getCompoundType(RefKind, elemType)
}

func MakeMapType(keyType, valType *Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.getCompoundType(MapKind, keyType, valType)
}

func MakeStructType(name string, fields map[string]*Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.makeStructType(name, fields)
}

func MakeUnionType(elemTypes ...*Type) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.makeUnionType(elemTypes...)
}

func MakeCycleType(level uint32) *Type {
	staticTypeCache.Lock()
	defer staticTypeCache.Unlock()
	return staticTypeCache.getCyclicType(level)
}

// typeTrie is node is a "type trie". All types in noms are created in a deterministic order. A typeTrie stores types within a typeCache and allows construction of a prexisting type to return the already existing one rather than allocate a new one.
type typeTrie struct {
	t       *Type
	entries map[uint32]*typeTrie
}

func newTypeTrie() *typeTrie {
	return &typeTrie{entries: map[uint32]*typeTrie{}}
}

func (tct *typeTrie) Traverse(typeId uint32) *typeTrie {
	if t, ok := tct.entries[typeId]; ok {
		return t
	}

	// Insert edge
	next := newTypeTrie()
	tct.entries[typeId] = next
	return next
}

type identTable struct {
	entries map[string]uint32
	nextId  uint32
}

func newIdentTable() *identTable {
	return &identTable{entries: map[string]uint32{}}
}

func (it *identTable) GetId(ident string) uint32 {
	if id, ok := it.entries[ident]; ok {
		return id
	}

	id := it.nextId
	it.nextId++
	it.entries[ident] = id
	return id
}
