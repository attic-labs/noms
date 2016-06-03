// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

// TypeDesc describes a type of the kind returned by Kind(), e.g. Map, Number, or a custom type.
type TypeDesc interface {
	Kind() NomsKind
	Equals(other TypeDesc) bool
}

// PrimitiveDesc implements TypeDesc for all primitive Noms types:
// Blob
// Bool
// Number
// Package
// String
// Type
// Value
type PrimitiveDesc NomsKind

func (p PrimitiveDesc) Kind() NomsKind {
	return NomsKind(p)
}

func (p PrimitiveDesc) Equals(other TypeDesc) bool {
	return p.Kind() == other.Kind()
}

var KindToString = map[NomsKind]string{
	BlobKind:   "Blob",
	BoolKind:   "Bool",
	ListKind:   "List",
	MapKind:    "Map",
	NumberKind: "Number",
	RefKind:    "Ref",
	SetKind:    "Set",
	StringKind: "String",
	TypeKind:   "Type",
	ValueKind:  "Value",
	CycleKind:  "Cycle",
	UnionKind:  "Union",
}

// CompoundDesc describes a List, Map, Set or Ref type.
// ElemTypes indicates what type or types are in the container indicated by kind, e.g. Map key and value or Set element.
type CompoundDesc struct {
	kind      NomsKind
	ElemTypes []*Type
}

func (c CompoundDesc) Kind() NomsKind {
	return c.kind
}

func (c CompoundDesc) Equals(other TypeDesc) bool {
	if c.Kind() != other.Kind() {
		return false
	}
	for i, e := range other.(CompoundDesc).ElemTypes {
		if !e.Equals(c.ElemTypes[i]) {
			return false
		}
	}
	return true
}

type TypeMap map[string]*Type

type field struct {
	name string
	t    *Type
}

type fieldSlice []field

func (s fieldSlice) Len() int           { return len(s) }
func (s fieldSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s fieldSlice) Less(i, j int) bool { return s[i].name < s[j].name }

// StructDesc describes a custom Noms Struct.
// Structs can contain at most one anonymous union, so Union may be nil.
type StructDesc struct {
	Name   string
	fields []field
}

func (s StructDesc) Kind() NomsKind {
	return StructKind
}

func (s StructDesc) Equals(other TypeDesc) bool {
	if s.Kind() != other.Kind() || len(s.fields) != len(other.(StructDesc).fields) {
		return false
	}
	otherDesc := other.(StructDesc)
	for i, field := range s.fields {
		if field.name != otherDesc.fields[i].name || !field.t.Equals(otherDesc.fields[i].t) {
			return false
		}
	}
	return true
}

func (s StructDesc) IterFields(cb func(name string, t *Type)) {
	for _, field := range s.fields {
		cb(field.name, field.t)
	}
}

func (s StructDesc) Field(name string) *Type {
	f, i := s.findField(name)
	if i == -1 {
		return nil
	}
	return f.t
}

func (s StructDesc) SetField(name string, t *Type) {
	f, i := s.findField(name)
	d.Chk.True(i != -1, "No such field %s", name)
	f.t = t
}

func (s StructDesc) findField(name string) (*field, int) {
	i := sort.Search(len(s.fields), func(i int) bool { return s.fields[i].name >= name })
	if i == len(s.fields) || s.fields[i].name != name {
		return nil, -1
	}
	return &s.fields[i], i
}

// Len returns the number of fields in the struct
func (s StructDesc) Len() int {
	return len(s.fields)
}
