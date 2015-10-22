// This file was generated by nomdl/codegen.

package datas

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __datasPackageInFile_types_CachedRef = __datasPackageInFile_types_Ref()

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func __datasPackageInFile_types_Ref() ref.Ref {
	p := types.NewPackage([]types.TypeRef{
		types.MakeStructTypeRef("Commit",
			[]types.Field{
				types.Field{"value", types.MakeCompoundTypeRef("", types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind)), false},
				types.Field{"parents", types.MakeCompoundTypeRef("", types.SetKind, types.MakeCompoundTypeRef("", types.RefKind, types.MakeTypeRef(ref.Ref{}, 0))), false},
			},
			types.Choices{},
		),
	}, []ref.Ref{})
	return types.RegisterPackage(&p)
}

// Commit

type Commit struct {
	m   types.Map
	ref *ref.Ref
}

func NewCommit() Commit {
	return Commit{types.NewMap(
		types.NewString("value"), NewRefOfValue(ref.Ref{}),
		types.NewString("parents"), NewSetOfRefOfCommit(),
	), &ref.Ref{}}
}

type CommitDef struct {
	Value   ref.Ref
	Parents SetOfRefOfCommitDef
}

func (def CommitDef) New() Commit {
	return Commit{
		types.NewMap(
			types.NewString("value"), NewRefOfValue(def.Value),
			types.NewString("parents"), def.Parents.New(),
		), &ref.Ref{}}
}

func (s Commit) Def() (d CommitDef) {
	d.Value = s.m.Get(types.NewString("value")).Ref()
	d.Parents = s.m.Get(types.NewString("parents")).(SetOfRefOfCommit).Def()
	return
}

var __typeRefForCommit = types.MakeTypeRef(__datasPackageInFile_types_CachedRef, 0)

func (m Commit) TypeRef() types.TypeRef {
	return __typeRefForCommit
}

func init() {
	types.RegisterFromValFunction(__typeRefForCommit, func(v types.Value) types.Value {
		return CommitFromVal(v)
	})
}

func CommitFromVal(val types.Value) Commit {
	// TODO: Do we still need FromVal?
	if val, ok := val.(Commit); ok {
		return val
	}
	// TODO: Validate here
	return Commit{val.(types.Map), &ref.Ref{}}
}

func (s Commit) InternalImplementation() types.Map {
	return s.m
}

func (s Commit) Equals(other types.Value) bool {
	if other, ok := other.(Commit); ok {
		return s.Ref() == other.Ref()
	}
	return false
}

func (s Commit) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Commit) Chunks() (futures []types.Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.m.Chunks()...)
	return
}

func (s Commit) Value() RefOfValue {
	return s.m.Get(types.NewString("value")).(RefOfValue)
}

func (s Commit) SetValue(val RefOfValue) Commit {
	return Commit{s.m.Set(types.NewString("value"), val), &ref.Ref{}}
}

func (s Commit) Parents() SetOfRefOfCommit {
	return s.m.Get(types.NewString("parents")).(SetOfRefOfCommit)
}

func (s Commit) SetParents(val SetOfRefOfCommit) Commit {
	return Commit{s.m.Set(types.NewString("parents"), val), &ref.Ref{}}
}

// MapOfStringToRefOfCommit

type MapOfStringToRefOfCommit struct {
	m   types.Map
	ref *ref.Ref
}

func NewMapOfStringToRefOfCommit() MapOfStringToRefOfCommit {
	return MapOfStringToRefOfCommit{types.NewMap(), &ref.Ref{}}
}

type MapOfStringToRefOfCommitDef map[string]ref.Ref

func (def MapOfStringToRefOfCommitDef) New() MapOfStringToRefOfCommit {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), NewRefOfCommit(v))
	}
	return MapOfStringToRefOfCommit{types.NewMap(kv...), &ref.Ref{}}
}

func (m MapOfStringToRefOfCommit) Def() MapOfStringToRefOfCommitDef {
	def := make(map[string]ref.Ref)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = v.Ref()
		return false
	})
	return def
}

func MapOfStringToRefOfCommitFromVal(val types.Value) MapOfStringToRefOfCommit {
	// TODO: Do we still need FromVal?
	if val, ok := val.(MapOfStringToRefOfCommit); ok {
		return val
	}
	// TODO: Validate here
	return MapOfStringToRefOfCommit{val.(types.Map), &ref.Ref{}}
}

func (m MapOfStringToRefOfCommit) InternalImplementation() types.Map {
	return m.m
}

func (m MapOfStringToRefOfCommit) Equals(other types.Value) bool {
	if other, ok := other.(MapOfStringToRefOfCommit); ok {
		return m.Ref() == other.Ref()
	}
	return false
}

func (m MapOfStringToRefOfCommit) Ref() ref.Ref {
	return types.EnsureRef(m.ref, m)
}

func (m MapOfStringToRefOfCommit) Chunks() (futures []types.Future) {
	futures = append(futures, m.TypeRef().Chunks()...)
	futures = append(futures, m.m.Chunks()...)
	return
}

// A Noms Value that describes MapOfStringToRefOfCommit.
var __typeRefForMapOfStringToRefOfCommit types.TypeRef

func (m MapOfStringToRefOfCommit) TypeRef() types.TypeRef {
	return __typeRefForMapOfStringToRefOfCommit
}

func init() {
	__typeRefForMapOfStringToRefOfCommit = types.MakeCompoundTypeRef("", types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeCompoundTypeRef("", types.RefKind, types.MakeTypeRef(__datasPackageInFile_types_CachedRef, 0)))
	types.RegisterFromValFunction(__typeRefForMapOfStringToRefOfCommit, func(v types.Value) types.Value {
		return MapOfStringToRefOfCommitFromVal(v)
	})
}

func (m MapOfStringToRefOfCommit) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToRefOfCommit) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToRefOfCommit) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToRefOfCommit) Get(p string) RefOfCommit {
	return m.m.Get(types.NewString(p)).(RefOfCommit)
}

func (m MapOfStringToRefOfCommit) MaybeGet(p string) (RefOfCommit, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewRefOfCommit(ref.Ref{}), false
	}
	return v.(RefOfCommit), ok
}

func (m MapOfStringToRefOfCommit) Set(k string, v RefOfCommit) MapOfStringToRefOfCommit {
	return MapOfStringToRefOfCommit{m.m.Set(types.NewString(k), v), &ref.Ref{}}
}

// TODO: Implement SetM?

func (m MapOfStringToRefOfCommit) Remove(p string) MapOfStringToRefOfCommit {
	return MapOfStringToRefOfCommit{m.m.Remove(types.NewString(p)), &ref.Ref{}}
}

type MapOfStringToRefOfCommitIterCallback func(k string, v RefOfCommit) (stop bool)

func (m MapOfStringToRefOfCommit) Iter(cb MapOfStringToRefOfCommitIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(RefOfCommit))
	})
}

type MapOfStringToRefOfCommitIterAllCallback func(k string, v RefOfCommit)

func (m MapOfStringToRefOfCommit) IterAll(cb MapOfStringToRefOfCommitIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), v.(RefOfCommit))
	})
}

type MapOfStringToRefOfCommitFilterCallback func(k string, v RefOfCommit) (keep bool)

func (m MapOfStringToRefOfCommit) Filter(cb MapOfStringToRefOfCommitFilterCallback) MapOfStringToRefOfCommit {
	nm := NewMapOfStringToRefOfCommit()
	m.IterAll(func(k string, v RefOfCommit) {
		if cb(k, v) {
			nm = nm.Set(k, v)
		}
	})
	return nm
}

// RefOfValue

type RefOfValue struct {
	r   ref.Ref
	ref *ref.Ref
}

func NewRefOfValue(r ref.Ref) RefOfValue {
	return RefOfValue{r, &ref.Ref{}}
}

func (r RefOfValue) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfValue) Equals(other types.Value) bool {
	if other, ok := other.(RefOfValue); ok {
		return r.Ref() == other.Ref()
	}
	return false
}

func (r RefOfValue) Chunks() []types.Future {
	return r.TypeRef().Chunks()
}

func (r RefOfValue) InternalImplementation() ref.Ref {
	return r.r
}

func RefOfValueFromVal(val types.Value) RefOfValue {
	// TODO: Do we still need FromVal?
	if val, ok := val.(RefOfValue); ok {
		return val
	}
	return RefOfValue{val.(types.Ref).Ref(), &ref.Ref{}}
}

// A Noms Value that describes RefOfValue.
var __typeRefForRefOfValue types.TypeRef

func (m RefOfValue) TypeRef() types.TypeRef {
	return __typeRefForRefOfValue
}

func init() {
	__typeRefForRefOfValue = types.MakeCompoundTypeRef("", types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind))
	types.RegisterFromValFunction(__typeRefForRefOfValue, func(v types.Value) types.Value {
		return RefOfValueFromVal(v)
	})
}

func (r RefOfValue) GetValue(cs chunks.ChunkSource) types.Value {
	return types.ReadValue(r.r, cs)
}

func (r RefOfValue) SetValue(val types.Value, cs chunks.ChunkSink) RefOfValue {
	return RefOfValue{types.WriteValue(val, cs), &ref.Ref{}}
}

// SetOfRefOfCommit

type SetOfRefOfCommit struct {
	s   types.Set
	ref *ref.Ref
}

func NewSetOfRefOfCommit() SetOfRefOfCommit {
	return SetOfRefOfCommit{types.NewSet(), &ref.Ref{}}
}

type SetOfRefOfCommitDef map[ref.Ref]bool

func (def SetOfRefOfCommitDef) New() SetOfRefOfCommit {
	l := make([]types.Value, len(def))
	i := 0
	for d, _ := range def {
		l[i] = NewRefOfCommit(d)
		i++
	}
	return SetOfRefOfCommit{types.NewSet(l...), &ref.Ref{}}
}

func (s SetOfRefOfCommit) Def() SetOfRefOfCommitDef {
	def := make(map[ref.Ref]bool, s.Len())
	s.s.Iter(func(v types.Value) bool {
		def[v.Ref()] = true
		return false
	})
	return def
}

func SetOfRefOfCommitFromVal(val types.Value) SetOfRefOfCommit {
	// TODO: Do we still need FromVal?
	if val, ok := val.(SetOfRefOfCommit); ok {
		return val
	}
	return SetOfRefOfCommit{val.(types.Set), &ref.Ref{}}
}

func (s SetOfRefOfCommit) InternalImplementation() types.Set {
	return s.s
}

func (s SetOfRefOfCommit) Equals(other types.Value) bool {
	if other, ok := other.(SetOfRefOfCommit); ok {
		return s.Ref() == other.Ref()
	}
	return false
}

func (s SetOfRefOfCommit) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s SetOfRefOfCommit) Chunks() (futures []types.Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.s.Chunks()...)
	return
}

// A Noms Value that describes SetOfRefOfCommit.
var __typeRefForSetOfRefOfCommit types.TypeRef

func (m SetOfRefOfCommit) TypeRef() types.TypeRef {
	return __typeRefForSetOfRefOfCommit
}

func init() {
	__typeRefForSetOfRefOfCommit = types.MakeCompoundTypeRef("", types.SetKind, types.MakeCompoundTypeRef("", types.RefKind, types.MakeTypeRef(__datasPackageInFile_types_CachedRef, 0)))
	types.RegisterFromValFunction(__typeRefForSetOfRefOfCommit, func(v types.Value) types.Value {
		return SetOfRefOfCommitFromVal(v)
	})
}

func (s SetOfRefOfCommit) Empty() bool {
	return s.s.Empty()
}

func (s SetOfRefOfCommit) Len() uint64 {
	return s.s.Len()
}

func (s SetOfRefOfCommit) Has(p RefOfCommit) bool {
	return s.s.Has(p)
}

type SetOfRefOfCommitIterCallback func(p RefOfCommit) (stop bool)

func (s SetOfRefOfCommit) Iter(cb SetOfRefOfCommitIterCallback) {
	s.s.Iter(func(v types.Value) bool {
		return cb(v.(RefOfCommit))
	})
}

type SetOfRefOfCommitIterAllCallback func(p RefOfCommit)

func (s SetOfRefOfCommit) IterAll(cb SetOfRefOfCommitIterAllCallback) {
	s.s.IterAll(func(v types.Value) {
		cb(v.(RefOfCommit))
	})
}

type SetOfRefOfCommitFilterCallback func(p RefOfCommit) (keep bool)

func (s SetOfRefOfCommit) Filter(cb SetOfRefOfCommitFilterCallback) SetOfRefOfCommit {
	ns := NewSetOfRefOfCommit()
	s.IterAll(func(v RefOfCommit) {
		if cb(v) {
			ns = ns.Insert(v)
		}
	})
	return ns
}

func (s SetOfRefOfCommit) Insert(p ...RefOfCommit) SetOfRefOfCommit {
	return SetOfRefOfCommit{s.s.Insert(s.fromElemSlice(p)...), &ref.Ref{}}
}

func (s SetOfRefOfCommit) Remove(p ...RefOfCommit) SetOfRefOfCommit {
	return SetOfRefOfCommit{s.s.Remove(s.fromElemSlice(p)...), &ref.Ref{}}
}

func (s SetOfRefOfCommit) Union(others ...SetOfRefOfCommit) SetOfRefOfCommit {
	return SetOfRefOfCommit{s.s.Union(s.fromStructSlice(others)...), &ref.Ref{}}
}

func (s SetOfRefOfCommit) Subtract(others ...SetOfRefOfCommit) SetOfRefOfCommit {
	return SetOfRefOfCommit{s.s.Subtract(s.fromStructSlice(others)...), &ref.Ref{}}
}

func (s SetOfRefOfCommit) Any() RefOfCommit {
	return s.s.Any().(RefOfCommit)
}

func (s SetOfRefOfCommit) fromStructSlice(p []SetOfRefOfCommit) []types.Set {
	r := make([]types.Set, len(p))
	for i, v := range p {
		r[i] = v.s
	}
	return r
}

func (s SetOfRefOfCommit) fromElemSlice(p []RefOfCommit) []types.Value {
	r := make([]types.Value, len(p))
	for i, v := range p {
		r[i] = v
	}
	return r
}

// RefOfCommit

type RefOfCommit struct {
	r   ref.Ref
	ref *ref.Ref
}

func NewRefOfCommit(r ref.Ref) RefOfCommit {
	return RefOfCommit{r, &ref.Ref{}}
}

func (r RefOfCommit) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfCommit) Equals(other types.Value) bool {
	if other, ok := other.(RefOfCommit); ok {
		return r.Ref() == other.Ref()
	}
	return false
}

func (r RefOfCommit) Chunks() []types.Future {
	return r.TypeRef().Chunks()
}

func (r RefOfCommit) InternalImplementation() ref.Ref {
	return r.r
}

func RefOfCommitFromVal(val types.Value) RefOfCommit {
	// TODO: Do we still need FromVal?
	if val, ok := val.(RefOfCommit); ok {
		return val
	}
	return RefOfCommit{val.(types.Ref).Ref(), &ref.Ref{}}
}

// A Noms Value that describes RefOfCommit.
var __typeRefForRefOfCommit types.TypeRef

func (m RefOfCommit) TypeRef() types.TypeRef {
	return __typeRefForRefOfCommit
}

func init() {
	__typeRefForRefOfCommit = types.MakeCompoundTypeRef("", types.RefKind, types.MakeTypeRef(__datasPackageInFile_types_CachedRef, 0))
	types.RegisterFromValFunction(__typeRefForRefOfCommit, func(v types.Value) types.Value {
		return RefOfCommitFromVal(v)
	})
}

func (r RefOfCommit) GetValue(cs chunks.ChunkSource) Commit {
	return types.ReadValue(r.r, cs).(Commit)
}

func (r RefOfCommit) SetValue(val Commit, cs chunks.ChunkSink) RefOfCommit {
	return RefOfCommit{types.WriteValue(val, cs), &ref.Ref{}}
}
