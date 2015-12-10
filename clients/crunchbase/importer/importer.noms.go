// This file was generated by nomdl/codegen.

package main

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __mainPackageInFile_importer_CachedRef ref.Ref

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func init() {
	p := types.NewPackage([]types.Type{
		types.MakeStructType("Import",
			[]types.Field{
				types.Field{"Input", types.MakeType(ref.Ref{}, 1), false},
				types.Field{"Date", types.MakeType(ref.Ref{}, 2), false},
				types.Field{"Output", types.MakeCompoundType(types.RefKind, types.MakeCompoundType(types.MapKind, types.MakePrimitiveType(types.StringKind), types.MakeCompoundType(types.RefKind, types.MakeType(ref.Parse("sha1-3e4f60c3fbd518f4a7e903ac1c7c1a97b677c4d9"), 0)))), false},
			},
			types.Choices{},
		),
		types.MakeStructType("Inputs",
			[]types.Field{
				types.Field{"CodeVersion", types.MakePrimitiveType(types.Uint32Kind), false},
				types.Field{"FileSHA1", types.MakePrimitiveType(types.StringKind), false},
			},
			types.Choices{},
		),
		types.MakeStructType("Date",
			[]types.Field{
				types.Field{"RFC3339", types.MakePrimitiveType(types.StringKind), false},
			},
			types.Choices{},
		),
	}, []ref.Ref{
		ref.Parse("sha1-3e4f60c3fbd518f4a7e903ac1c7c1a97b677c4d9"),
	})
	__mainPackageInFile_importer_CachedRef = types.RegisterPackage(&p)
}

// Import

type Import struct {
	_Input  Inputs
	_Date   Date
	_Output RefOfMapOfStringToRefOfCompany

	cs  chunks.ChunkStore
	ref *ref.Ref
}

func NewImport(cs chunks.ChunkStore) Import {
	return Import{
		_Input:  NewInputs(cs),
		_Date:   NewDate(cs),
		_Output: NewRefOfMapOfStringToRefOfCompany(ref.Ref{}),

		cs:  cs,
		ref: &ref.Ref{},
	}
}

type ImportDef struct {
	Input  InputsDef
	Date   DateDef
	Output ref.Ref
}

func (def ImportDef) New(cs chunks.ChunkStore) Import {
	return Import{
		_Input:  def.Input.New(cs),
		_Date:   def.Date.New(cs),
		_Output: NewRefOfMapOfStringToRefOfCompany(def.Output),
		cs:      cs,
		ref:     &ref.Ref{},
	}
}

func (s Import) Def() (d ImportDef) {
	d.Input = s._Input.Def()
	d.Date = s._Date.Def()
	d.Output = s._Output.TargetRef()
	return
}

var __typeForImport types.Type

func (m Import) Type() types.Type {
	return __typeForImport
}

func init() {
	__typeForImport = types.MakeType(__mainPackageInFile_importer_CachedRef, 0)
	types.RegisterStruct(__typeForImport, builderForImport, readerForImport)
}

func builderForImport(cs chunks.ChunkStore, values []types.Value) types.Value {
	i := 0
	s := Import{ref: &ref.Ref{}, cs: cs}
	s._Input = values[i].(Inputs)
	i++
	s._Date = values[i].(Date)
	i++
	s._Output = values[i].(RefOfMapOfStringToRefOfCompany)
	i++
	return s
}

func readerForImport(v types.Value) []types.Value {
	values := []types.Value{}
	s := v.(Import)
	values = append(values, s._Input)
	values = append(values, s._Date)
	values = append(values, s._Output)
	return values
}

func (s Import) Equals(other types.Value) bool {
	return other != nil && __typeForImport.Equals(other.Type()) && s.Ref() == other.Ref()
}

func (s Import) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Import) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeForImport.Chunks()...)
	chunks = append(chunks, s._Input.Chunks()...)
	chunks = append(chunks, s._Date.Chunks()...)
	chunks = append(chunks, s._Output.Chunks()...)
	return
}

func (s Import) ChildValues() (ret []types.Value) {
	ret = append(ret, s._Input)
	ret = append(ret, s._Date)
	ret = append(ret, s._Output)
	return
}

func (s Import) Input() Inputs {
	return s._Input
}

func (s Import) SetInput(val Inputs) Import {
	s._Input = val
	s.ref = &ref.Ref{}
	return s
}

func (s Import) Date() Date {
	return s._Date
}

func (s Import) SetDate(val Date) Import {
	s._Date = val
	s.ref = &ref.Ref{}
	return s
}

func (s Import) Output() RefOfMapOfStringToRefOfCompany {
	return s._Output
}

func (s Import) SetOutput(val RefOfMapOfStringToRefOfCompany) Import {
	s._Output = val
	s.ref = &ref.Ref{}
	return s
}

// Inputs

type Inputs struct {
	_CodeVersion uint32
	_FileSHA1    string

	cs  chunks.ChunkStore
	ref *ref.Ref
}

func NewInputs(cs chunks.ChunkStore) Inputs {
	return Inputs{
		_CodeVersion: uint32(0),
		_FileSHA1:    "",

		cs:  cs,
		ref: &ref.Ref{},
	}
}

type InputsDef struct {
	CodeVersion uint32
	FileSHA1    string
}

func (def InputsDef) New(cs chunks.ChunkStore) Inputs {
	return Inputs{
		_CodeVersion: def.CodeVersion,
		_FileSHA1:    def.FileSHA1,
		cs:           cs,
		ref:          &ref.Ref{},
	}
}

func (s Inputs) Def() (d InputsDef) {
	d.CodeVersion = s._CodeVersion
	d.FileSHA1 = s._FileSHA1
	return
}

var __typeForInputs types.Type

func (m Inputs) Type() types.Type {
	return __typeForInputs
}

func init() {
	__typeForInputs = types.MakeType(__mainPackageInFile_importer_CachedRef, 1)
	types.RegisterStruct(__typeForInputs, builderForInputs, readerForInputs)
}

func builderForInputs(cs chunks.ChunkStore, values []types.Value) types.Value {
	i := 0
	s := Inputs{ref: &ref.Ref{}, cs: cs}
	s._CodeVersion = uint32(values[i].(types.Uint32))
	i++
	s._FileSHA1 = values[i].(types.String).String()
	i++
	return s
}

func readerForInputs(v types.Value) []types.Value {
	values := []types.Value{}
	s := v.(Inputs)
	values = append(values, types.Uint32(s._CodeVersion))
	values = append(values, types.NewString(s._FileSHA1))
	return values
}

func (s Inputs) Equals(other types.Value) bool {
	return other != nil && __typeForInputs.Equals(other.Type()) && s.Ref() == other.Ref()
}

func (s Inputs) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Inputs) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeForInputs.Chunks()...)
	return
}

func (s Inputs) ChildValues() (ret []types.Value) {
	ret = append(ret, types.Uint32(s._CodeVersion))
	ret = append(ret, types.NewString(s._FileSHA1))
	return
}

func (s Inputs) CodeVersion() uint32 {
	return s._CodeVersion
}

func (s Inputs) SetCodeVersion(val uint32) Inputs {
	s._CodeVersion = val
	s.ref = &ref.Ref{}
	return s
}

func (s Inputs) FileSHA1() string {
	return s._FileSHA1
}

func (s Inputs) SetFileSHA1(val string) Inputs {
	s._FileSHA1 = val
	s.ref = &ref.Ref{}
	return s
}

// Date

type Date struct {
	_RFC3339 string

	cs  chunks.ChunkStore
	ref *ref.Ref
}

func NewDate(cs chunks.ChunkStore) Date {
	return Date{
		_RFC3339: "",

		cs:  cs,
		ref: &ref.Ref{},
	}
}

type DateDef struct {
	RFC3339 string
}

func (def DateDef) New(cs chunks.ChunkStore) Date {
	return Date{
		_RFC3339: def.RFC3339,
		cs:       cs,
		ref:      &ref.Ref{},
	}
}

func (s Date) Def() (d DateDef) {
	d.RFC3339 = s._RFC3339
	return
}

var __typeForDate types.Type

func (m Date) Type() types.Type {
	return __typeForDate
}

func init() {
	__typeForDate = types.MakeType(__mainPackageInFile_importer_CachedRef, 2)
	types.RegisterStruct(__typeForDate, builderForDate, readerForDate)
}

func builderForDate(cs chunks.ChunkStore, values []types.Value) types.Value {
	i := 0
	s := Date{ref: &ref.Ref{}, cs: cs}
	s._RFC3339 = values[i].(types.String).String()
	i++
	return s
}

func readerForDate(v types.Value) []types.Value {
	values := []types.Value{}
	s := v.(Date)
	values = append(values, types.NewString(s._RFC3339))
	return values
}

func (s Date) Equals(other types.Value) bool {
	return other != nil && __typeForDate.Equals(other.Type()) && s.Ref() == other.Ref()
}

func (s Date) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Date) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeForDate.Chunks()...)
	return
}

func (s Date) ChildValues() (ret []types.Value) {
	ret = append(ret, types.NewString(s._RFC3339))
	return
}

func (s Date) RFC3339() string {
	return s._RFC3339
}

func (s Date) SetRFC3339(val string) Date {
	s._RFC3339 = val
	s.ref = &ref.Ref{}
	return s
}

// RefOfImport

type RefOfImport struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfImport(target ref.Ref) RefOfImport {
	return RefOfImport{target, &ref.Ref{}}
}

func (r RefOfImport) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfImport) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfImport) Equals(other types.Value) bool {
	return other != nil && __typeForRefOfImport.Equals(other.Type()) && r.Ref() == other.Ref()
}

func (r RefOfImport) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, r.Type().Chunks()...)
	chunks = append(chunks, r.target)
	return
}

func (r RefOfImport) ChildValues() []types.Value {
	return nil
}

// A Noms Value that describes RefOfImport.
var __typeForRefOfImport types.Type

func (m RefOfImport) Type() types.Type {
	return __typeForRefOfImport
}

func init() {
	__typeForRefOfImport = types.MakeCompoundType(types.RefKind, types.MakeType(__mainPackageInFile_importer_CachedRef, 0))
	types.RegisterRef(__typeForRefOfImport, builderForRefOfImport)
}

func builderForRefOfImport(r ref.Ref) types.Value {
	return NewRefOfImport(r)
}

func (r RefOfImport) TargetValue(cs chunks.ChunkStore) Import {
	return types.ReadValue(r.target, cs).(Import)
}

func (r RefOfImport) SetTargetValue(val Import, cs chunks.ChunkSink) RefOfImport {
	return NewRefOfImport(types.WriteValue(val, cs))
}

// RefOfMapOfStringToRefOfCompany

type RefOfMapOfStringToRefOfCompany struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfMapOfStringToRefOfCompany(target ref.Ref) RefOfMapOfStringToRefOfCompany {
	return RefOfMapOfStringToRefOfCompany{target, &ref.Ref{}}
}

func (r RefOfMapOfStringToRefOfCompany) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfMapOfStringToRefOfCompany) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfMapOfStringToRefOfCompany) Equals(other types.Value) bool {
	return other != nil && __typeForRefOfMapOfStringToRefOfCompany.Equals(other.Type()) && r.Ref() == other.Ref()
}

func (r RefOfMapOfStringToRefOfCompany) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, r.Type().Chunks()...)
	chunks = append(chunks, r.target)
	return
}

func (r RefOfMapOfStringToRefOfCompany) ChildValues() []types.Value {
	return nil
}

// A Noms Value that describes RefOfMapOfStringToRefOfCompany.
var __typeForRefOfMapOfStringToRefOfCompany types.Type

func (m RefOfMapOfStringToRefOfCompany) Type() types.Type {
	return __typeForRefOfMapOfStringToRefOfCompany
}

func init() {
	__typeForRefOfMapOfStringToRefOfCompany = types.MakeCompoundType(types.RefKind, types.MakeCompoundType(types.MapKind, types.MakePrimitiveType(types.StringKind), types.MakeCompoundType(types.RefKind, types.MakeType(ref.Parse("sha1-3e4f60c3fbd518f4a7e903ac1c7c1a97b677c4d9"), 0))))
	types.RegisterRef(__typeForRefOfMapOfStringToRefOfCompany, builderForRefOfMapOfStringToRefOfCompany)
}

func builderForRefOfMapOfStringToRefOfCompany(r ref.Ref) types.Value {
	return NewRefOfMapOfStringToRefOfCompany(r)
}

func (r RefOfMapOfStringToRefOfCompany) TargetValue(cs chunks.ChunkStore) MapOfStringToRefOfCompany {
	return types.ReadValue(r.target, cs).(MapOfStringToRefOfCompany)
}

func (r RefOfMapOfStringToRefOfCompany) SetTargetValue(val MapOfStringToRefOfCompany, cs chunks.ChunkSink) RefOfMapOfStringToRefOfCompany {
	return NewRefOfMapOfStringToRefOfCompany(types.WriteValue(val, cs))
}

// MapOfStringToRefOfCompany

type MapOfStringToRefOfCompany struct {
	m   types.Map
	cs  chunks.ChunkStore
	ref *ref.Ref
}

func NewMapOfStringToRefOfCompany(cs chunks.ChunkStore) MapOfStringToRefOfCompany {
	return MapOfStringToRefOfCompany{types.NewTypedMap(cs, __typeForMapOfStringToRefOfCompany), cs, &ref.Ref{}}
}

type MapOfStringToRefOfCompanyDef map[string]ref.Ref

func (def MapOfStringToRefOfCompanyDef) New(cs chunks.ChunkStore) MapOfStringToRefOfCompany {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), NewRefOfCompany(v))
	}
	return MapOfStringToRefOfCompany{types.NewTypedMap(cs, __typeForMapOfStringToRefOfCompany, kv...), cs, &ref.Ref{}}
}

func (m MapOfStringToRefOfCompany) Def() MapOfStringToRefOfCompanyDef {
	def := make(map[string]ref.Ref)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = v.(RefOfCompany).TargetRef()
		return false
	})
	return def
}

func (m MapOfStringToRefOfCompany) Equals(other types.Value) bool {
	return other != nil && __typeForMapOfStringToRefOfCompany.Equals(other.Type()) && m.Ref() == other.Ref()
}

func (m MapOfStringToRefOfCompany) Ref() ref.Ref {
	return types.EnsureRef(m.ref, m)
}

func (m MapOfStringToRefOfCompany) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, m.Type().Chunks()...)
	chunks = append(chunks, m.m.Chunks()...)
	return
}

func (m MapOfStringToRefOfCompany) ChildValues() []types.Value {
	return append([]types.Value{}, m.m.ChildValues()...)
}

// A Noms Value that describes MapOfStringToRefOfCompany.
var __typeForMapOfStringToRefOfCompany types.Type

func (m MapOfStringToRefOfCompany) Type() types.Type {
	return __typeForMapOfStringToRefOfCompany
}

func init() {
	__typeForMapOfStringToRefOfCompany = types.MakeCompoundType(types.MapKind, types.MakePrimitiveType(types.StringKind), types.MakeCompoundType(types.RefKind, types.MakeType(ref.Parse("sha1-3e4f60c3fbd518f4a7e903ac1c7c1a97b677c4d9"), 0)))
	types.RegisterValue(__typeForMapOfStringToRefOfCompany, builderForMapOfStringToRefOfCompany, readerForMapOfStringToRefOfCompany)
}

func builderForMapOfStringToRefOfCompany(cs chunks.ChunkStore, v types.Value) types.Value {
	return MapOfStringToRefOfCompany{v.(types.Map), cs, &ref.Ref{}}
}

func readerForMapOfStringToRefOfCompany(v types.Value) types.Value {
	return v.(MapOfStringToRefOfCompany).m
}

func (m MapOfStringToRefOfCompany) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToRefOfCompany) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToRefOfCompany) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToRefOfCompany) Get(p string) RefOfCompany {
	return m.m.Get(types.NewString(p)).(RefOfCompany)
}

func (m MapOfStringToRefOfCompany) MaybeGet(p string) (RefOfCompany, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewRefOfCompany(ref.Ref{}), false
	}
	return v.(RefOfCompany), ok
}

func (m MapOfStringToRefOfCompany) Set(k string, v RefOfCompany) MapOfStringToRefOfCompany {
	return MapOfStringToRefOfCompany{m.m.Set(types.NewString(k), v), m.cs, &ref.Ref{}}
}

// TODO: Implement SetM?

func (m MapOfStringToRefOfCompany) Remove(p string) MapOfStringToRefOfCompany {
	return MapOfStringToRefOfCompany{m.m.Remove(types.NewString(p)), m.cs, &ref.Ref{}}
}

type MapOfStringToRefOfCompanyIterCallback func(k string, v RefOfCompany) (stop bool)

func (m MapOfStringToRefOfCompany) Iter(cb MapOfStringToRefOfCompanyIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(RefOfCompany))
	})
}

type MapOfStringToRefOfCompanyIterAllCallback func(k string, v RefOfCompany)

func (m MapOfStringToRefOfCompany) IterAll(cb MapOfStringToRefOfCompanyIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), v.(RefOfCompany))
	})
}

func (m MapOfStringToRefOfCompany) IterAllP(concurrency int, cb MapOfStringToRefOfCompanyIterAllCallback) {
	m.m.IterAllP(concurrency, func(k, v types.Value) {
		cb(k.(types.String).String(), v.(RefOfCompany))
	})
}

type MapOfStringToRefOfCompanyFilterCallback func(k string, v RefOfCompany) (keep bool)

func (m MapOfStringToRefOfCompany) Filter(cb MapOfStringToRefOfCompanyFilterCallback) MapOfStringToRefOfCompany {
	out := m.m.Filter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(RefOfCompany))
	})
	return MapOfStringToRefOfCompany{out, m.cs, &ref.Ref{}}
}

// RefOfCompany

type RefOfCompany struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfCompany(target ref.Ref) RefOfCompany {
	return RefOfCompany{target, &ref.Ref{}}
}

func (r RefOfCompany) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfCompany) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfCompany) Equals(other types.Value) bool {
	return other != nil && __typeForRefOfCompany.Equals(other.Type()) && r.Ref() == other.Ref()
}

func (r RefOfCompany) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, r.Type().Chunks()...)
	chunks = append(chunks, r.target)
	return
}

func (r RefOfCompany) ChildValues() []types.Value {
	return nil
}

// A Noms Value that describes RefOfCompany.
var __typeForRefOfCompany types.Type

func (m RefOfCompany) Type() types.Type {
	return __typeForRefOfCompany
}

func init() {
	__typeForRefOfCompany = types.MakeCompoundType(types.RefKind, types.MakeType(ref.Parse("sha1-3e4f60c3fbd518f4a7e903ac1c7c1a97b677c4d9"), 0))
	types.RegisterRef(__typeForRefOfCompany, builderForRefOfCompany)
}

func builderForRefOfCompany(r ref.Ref) types.Value {
	return NewRefOfCompany(r)
}

func (r RefOfCompany) TargetValue(cs chunks.ChunkStore) Company {
	return types.ReadValue(r.target, cs).(Company)
}

func (r RefOfCompany) SetTargetValue(val Company, cs chunks.ChunkSink) RefOfCompany {
	return NewRefOfCompany(types.WriteValue(val, cs))
}
