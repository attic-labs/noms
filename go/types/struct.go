// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

var EmptyStructType = MakeStructType("")
var EmptyStruct = Struct{"", structValueFields{}, &hash.Hash{}}

type StructData map[string]Value

type StructValueField struct {
	Name  string
	Value Value
}

type structValueFields []StructValueField

func (fs structValueFields) Len() int           { return len(fs) }
func (fs structValueFields) Swap(i, j int)      { fs[i], fs[j] = fs[j], fs[i] }
func (fs structValueFields) Less(i, j int) bool { return fs[i].Name < fs[j].Name }

type Struct struct {
	name   string
	fields structValueFields
	h      *hash.Hash
}

func NewStruct(name string, data StructData) Struct {
	valueFields := make(structValueFields, len(data))
	i := 0
	for name, value := range data {
		valueFields[i] = StructValueField{name, value}
		i++
	}

	sort.Sort(valueFields)

	return Struct{name, valueFields, &hash.Hash{}}
}

func (s Struct) hashPointer() *hash.Hash {
	return s.h
}

// Value interface
func (s Struct) Equals(other Value) bool {
	return s.Hash() == other.Hash()
}

func (s Struct) Less(other Value) bool {
	return valueLess(s, other)
}

func (s Struct) Hash() hash.Hash {
	if s.h.IsEmpty() {
		*s.h = getHash(s)
	}

	return *s.h
}

func (s Struct) WalkValues(cb ValueCallback) {
	for _, f := range s.fields {
		cb(f.Value)
	}
}

func (s Struct) WalkRefs(cb RefCallback) {
	for _, f := range s.fields {
		f.Value.WalkRefs(cb)
	}
}

func (s Struct) typeOf() *Type {
	typeFields := make(structTypeFields, len(s.fields))
	for i, valueField := range s.fields {
		typeFields[i] = StructField{
			Name:     valueField.Name,
			Optional: false,
			Type:     valueField.Value.typeOf(),
		}
	}
	return makeStructTypeQuickly(s.name, typeFields, checkKindNoValidate)
}

// Len is the number of fields in the struct.
func (s Struct) Len() int {
	return len(s.fields)
}

// Name is the name of the struct.
func (s Struct) Name() string {
	return s.name
}

// IterFields iterates over the fields, calling cb for every field in the
// struct.
func (s Struct) IterFields(cb func(name string, value Value)) {
	for _, f := range s.fields {
		cb(f.Name, f.Value)
	}
}

func (s Struct) Kind() NomsKind {
	return StructKind
}

// MaybeGet returns the value of a field in the struct. If the struct does not a have a field with
// the name name then this returns (nil, false).
func (s Struct) MaybeGet(n string) (Value, bool) {
	i := s.findField(n)
	if i == -1 {
		return nil, false
	}
	return s.fields[i].Value, true
}

func (s Struct) searchField(name string) int {
	return sort.Search(len(s.fields), func(i int) bool { return s.fields[i].Name >= name })
}

func (s Struct) findField(name string) int {
	i := s.searchField(name)
	if i == len(s.fields) || s.fields[i].Name != name {
		return -1
	}
	return i
}

// Get returns the value of a field in the struct. If the struct does not a have a field with the
// name name then this panics.
func (s Struct) Get(n string) Value {
	i := s.findField(n)
	if i == -1 {
		d.Chk.Fail(fmt.Sprintf(`Struct has no field "%s"`, n))
	}
	return s.fields[i].Value
}

// Set returns a new struct where the field name has been set to value. If name is not an
// existing field in the struct or the type of value is different from the old value of the
// struct field a new struct type is created.
func (s Struct) Set(n string, v Value) Struct {
	i := s.searchField(n)
	var valueFields structValueFields

	if i != len(s.fields) && s.fields[i].Name == n {
		// Found
		valueFields = make(structValueFields, len(s.fields))
		copy(valueFields, s.fields)
		valueFields[i].Value = v
	} else {
		// Not found.
		valueFields = make(structValueFields, len(s.fields)+1)
		copy(valueFields[:i], s.fields[:i])
		copy(valueFields[i+1:], s.fields[i:])

		valueFields[i] = StructValueField{n, v}
	}

	return Struct{s.name, valueFields, &hash.Hash{}}
}

// IsZeroValue can be used to test if a struct is the same as Struct{}.
func (s Struct) IsZeroValue() bool {
	return s.fields == nil && s.name == "" && s.h == nil
}

// Delete returns a new struct where the field name has been removed.
// If name is not an existing field in the struct then the current struct is returned.
func (s Struct) Delete(n string) Struct {
	i := s.findField(n)
	if i == -1 {
		return s
	}

	valueFields := make(structValueFields, len(s.fields)-1)
	copy(valueFields[:i], s.fields[:i])
	copy(valueFields[i:], s.fields[i+1:])

	return Struct{s.name, valueFields, &hash.Hash{}}
}

func (s Struct) Diff(last Struct, changes chan<- ValueChanged, closeChan <-chan struct{}) {
	if s.Equals(last) {
		return
	}
	fs1, fs2 := s.fields, last.fields
	i1, i2 := 0, 0
	for i1 < len(fs1) && i2 < len(fs2) {
		f1, f2 := fs1[i1], fs2[i2]
		fn1, fn2 := f1.Name, f2.Name

		var change ValueChanged
		if fn1 == fn2 {
			if !s.fields[i1].Value.Equals(last.fields[i2].Value) {
				change = ValueChanged{ChangeType: DiffChangeModified, V: String(fn1)}
			}
			i1++
			i2++
		} else if fn1 < fn2 {
			change = ValueChanged{ChangeType: DiffChangeAdded, V: String(fn1)}
			i1++
		} else {
			change = ValueChanged{ChangeType: DiffChangeRemoved, V: String(fn2)}
			i2++
		}

		if change != (ValueChanged{}) && !sendChange(changes, closeChan, change) {
			return
		}
	}

	for ; i1 < len(fs1); i1++ {
		if !sendChange(changes, closeChan, ValueChanged{ChangeType: DiffChangeAdded, V: String(fs1[i1].Name)}) {
			return
		}
	}

	for ; i2 < len(fs2); i2++ {
		if !sendChange(changes, closeChan, ValueChanged{ChangeType: DiffChangeRemoved, V: String(fs2[i2].Name)}) {
			return
		}
	}
}

var escapeChar = "Q"
var headFieldNamePattern = regexp.MustCompile("[a-zA-Z]")
var tailFieldNamePattern = regexp.MustCompile("[a-zA-Z0-9_]")
var spaceRegex = regexp.MustCompile("[ ]")
var escapeRegex = regexp.MustCompile(escapeChar)

var fieldNameComponentRe = regexp.MustCompile("^" + headFieldNamePattern.String() + tailFieldNamePattern.String() + "*")
var fieldNameRe = regexp.MustCompile(fieldNameComponentRe.String() + "$")

type encodingFunc func(string, *regexp.Regexp) string

func CamelCaseFieldName(input string) string {
	//strip invalid struct characters and leave spaces
	encode := func(s1 string, p *regexp.Regexp) string {
		if p.MatchString(s1) || spaceRegex.MatchString(s1) {
			return s1
		}
		return ""
	}

	strippedField := escapeField(input, encode)
	splitField := strings.Fields(strippedField)

	if len(splitField) == 0 {
		return ""
	}

	//Camelcase field
	output := strings.ToLower(splitField[0])
	if len(splitField) > 1 {
		for _, field := range splitField[1:] {
			output += strings.Title(strings.ToLower(field))
		}
	}
	//Because we are removing characters, we may generate an invalid field name
	//i.e. -- 1A B, we will remove the first bad chars and process until 1aB
	//1aB is invalid struct field name so we will return ""
	if !IsValidStructFieldName(output) {
		return ""
	}
	return output
}

func escapeField(input string, encode encodingFunc) string {
	output := ""
	pattern := headFieldNamePattern
	for _, ch := range input {
		output += encode(string([]rune{ch}), pattern)
		pattern = tailFieldNamePattern
	}
	return output
}

// EscapeStructField escapes names for use as noms structs with regards to non CSV imported data.
// Disallowed characters are encoded as 'Q<hex-encoded-utf8-bytes>'.
// Note that Q itself is also escaped since it is the escape character.
func EscapeStructField(input string) string {
	if !escapeRegex.MatchString(input) && IsValidStructFieldName(input) {
		return input
	}
	encode := func(s1 string, p *regexp.Regexp) string {
		if p.MatchString(s1) && s1 != escapeChar {
			return s1
		}

		var hs = fmt.Sprintf("%X", s1)
		var buf bytes.Buffer
		buf.WriteString(escapeChar)
		if len(hs) == 1 {
			buf.WriteString("0")
		}
		buf.WriteString(hs)
		return buf.String()
	}
	return escapeField(input, encode)
}

// IsValidStructFieldName returns whether the name is valid as a field name in a struct.
// Valid names must start with `a-zA-Z` and after that `a-zA-Z0-9_`.
func IsValidStructFieldName(name string) bool {
	return fieldNameRe.MatchString(name)
}

func verifyFields(fs structTypeFields) {
	for i, f := range fs {
		verifyFieldName(f.Name)
		if i > 0 && strings.Compare(fs[i-1].Name, f.Name) >= 0 {
			d.Chk.Fail("Field names must be unique and ordered alphabetically")
		}
	}
}

func verifyName(name, kind string) {
	if !IsValidStructFieldName(name) {
		d.Panic(`Invalid struct%s name: "%s"`, kind, name)
	}
}

func verifyFieldName(name string) {
	verifyName(name, " field")
}

func verifyStructName(name string) {
	if name != "" {
		verifyName(name, "")
	}
}
