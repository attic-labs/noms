// This file was generated by nomdl/codegen.

package main

import (
	"github.com/attic-labs/noms/clients/gen/sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __mainPackageInFile_types_CachedRef = __mainPackageInFile_types_Ref()

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func __mainPackageInFile_types_Ref() ref.Ref {
	p := types.PackageDef{
		Types: types.ListOfTypeRefDef{

			types.MakeStructTypeRef("PhotoUnion",
				[]types.Field{},
				types.Choices{
					types.Field{"Photo", types.MakeTypeRef(ref.Parse("sha1-ee6ba8b7a1135a4360459b053b68bf5f992bb23e"), 1), false},
					types.Field{"Remote", types.MakeTypeRef(ref.Parse("sha1-ee6ba8b7a1135a4360459b053b68bf5f992bb23e"), 2), false},
				},
			),
		},
	}.New()
	return types.RegisterPackage(&p)
}

// PhotoUnion

type PhotoUnion struct {
	m types.Map
}

func NewPhotoUnion() PhotoUnion {
	return PhotoUnion{types.NewMap(
		types.NewString("$type"), types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0),
		types.NewString("$unionIndex"), types.UInt32(0),
		types.NewString("$unionValue"), sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.NewPhoto().NomsValue(),
	)}
}

type PhotoUnionDef struct {
	__unionIndex uint32
	__unionValue interface{}
}

func (def PhotoUnionDef) New() PhotoUnion {
	return PhotoUnion{
		types.NewMap(
			types.NewString("$type"), types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0),
			types.NewString("$unionIndex"), types.UInt32(def.__unionIndex),
			types.NewString("$unionValue"), def.__unionDefToValue(),
		)}
}

func (s PhotoUnion) Def() (d PhotoUnionDef) {
	d.__unionIndex = uint32(s.m.Get(types.NewString("$unionIndex")).(types.UInt32))
	d.__unionValue = s.__unionValueToDef()
	return
}

func (def PhotoUnionDef) __unionDefToValue() types.Value {
	switch def.__unionIndex {
	case 0:
		return def.__unionValue.(sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoDef).New().NomsValue()
	case 1:
		return def.__unionValue.(sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoDef).New().NomsValue()
	}
	panic("unreachable")
}

func (s PhotoUnion) __unionValueToDef() interface{} {
	switch uint32(s.m.Get(types.NewString("$unionIndex")).(types.UInt32)) {
	case 0:
		return sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoFromVal(s.m.Get(types.NewString("$unionValue"))).Def()
	case 1:
		return sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoFromVal(s.m.Get(types.NewString("$unionValue"))).Def()
	}
	panic("unreachable")
}

var __typeRefForPhotoUnion = types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0)

func (m PhotoUnion) TypeRef() types.TypeRef {
	return __typeRefForPhotoUnion
}

func init() {
	types.RegisterFromValFunction(__typeRefForPhotoUnion, func(v types.Value) types.NomsValue {
		return PhotoUnionFromVal(v)
	})
}

func PhotoUnionFromVal(val types.Value) PhotoUnion {
	// TODO: Validate here
	return PhotoUnion{val.(types.Map)}
}

func (s PhotoUnion) NomsValue() types.Value {
	return s.m
}

func (s PhotoUnion) Equals(other types.Value) bool {
	if other, ok := other.(PhotoUnion); ok {
		return s.m.Equals(other.m)
	}
	return false
}

func (s PhotoUnion) Ref() ref.Ref {
	return s.m.Ref()
}

func (s PhotoUnion) Chunks() (futures []types.Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.m.Chunks()...)
	return
}

func (s PhotoUnion) Photo() (val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.Photo, ok bool) {
	if s.m.Get(types.NewString("$unionIndex")).(types.UInt32) != 0 {
		return
	}
	return sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoFromVal(s.m.Get(types.NewString("$unionValue"))), true
}

func (s PhotoUnion) SetPhoto(val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.Photo) PhotoUnion {
	return PhotoUnion{s.m.Set(types.NewString("$unionIndex"), types.UInt32(0)).Set(types.NewString("$unionValue"), val.NomsValue())}
}

func (def PhotoUnionDef) Photo() (val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoDef, ok bool) {
	if def.__unionIndex != 0 {
		return
	}
	return def.__unionValue.(sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoDef), true
}

func (def PhotoUnionDef) SetPhoto(val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.PhotoDef) PhotoUnionDef {
	def.__unionIndex = 0
	def.__unionValue = val
	return def
}

func (s PhotoUnion) Remote() (val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhoto, ok bool) {
	if s.m.Get(types.NewString("$unionIndex")).(types.UInt32) != 1 {
		return
	}
	return sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoFromVal(s.m.Get(types.NewString("$unionValue"))), true
}

func (s PhotoUnion) SetRemote(val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhoto) PhotoUnion {
	return PhotoUnion{s.m.Set(types.NewString("$unionIndex"), types.UInt32(1)).Set(types.NewString("$unionValue"), val.NomsValue())}
}

func (def PhotoUnionDef) Remote() (val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoDef, ok bool) {
	if def.__unionIndex != 1 {
		return
	}
	return def.__unionValue.(sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoDef), true
}

func (def PhotoUnionDef) SetRemote(val sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e.RemotePhotoDef) PhotoUnionDef {
	def.__unionIndex = 1
	def.__unionValue = val
	return def
}

// MapOfStringToSetOfPhotoUnion

type MapOfStringToSetOfPhotoUnion struct {
	m types.Map
}

func NewMapOfStringToSetOfPhotoUnion() MapOfStringToSetOfPhotoUnion {
	return MapOfStringToSetOfPhotoUnion{types.NewMap()}
}

type MapOfStringToSetOfPhotoUnionDef map[string]SetOfPhotoUnionDef

func (def MapOfStringToSetOfPhotoUnionDef) New() MapOfStringToSetOfPhotoUnion {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), v.New().NomsValue())
	}
	return MapOfStringToSetOfPhotoUnion{types.NewMap(kv...)}
}

func (m MapOfStringToSetOfPhotoUnion) Def() MapOfStringToSetOfPhotoUnionDef {
	def := make(map[string]SetOfPhotoUnionDef)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = SetOfPhotoUnionFromVal(v).Def()
		return false
	})
	return def
}

func MapOfStringToSetOfPhotoUnionFromVal(p types.Value) MapOfStringToSetOfPhotoUnion {
	// TODO: Validate here
	return MapOfStringToSetOfPhotoUnion{p.(types.Map)}
}

func (m MapOfStringToSetOfPhotoUnion) NomsValue() types.Value {
	return m.m
}

func (m MapOfStringToSetOfPhotoUnion) Equals(other types.Value) bool {
	if other, ok := other.(MapOfStringToSetOfPhotoUnion); ok {
		return m.m.Equals(other.m)
	}
	return false
}

func (m MapOfStringToSetOfPhotoUnion) Ref() ref.Ref {
	return m.m.Ref()
}

func (m MapOfStringToSetOfPhotoUnion) Chunks() (futures []types.Future) {
	futures = append(futures, m.TypeRef().Chunks()...)
	futures = append(futures, m.m.Chunks()...)
	return
}

// A Noms Value that describes MapOfStringToSetOfPhotoUnion.
var __typeRefForMapOfStringToSetOfPhotoUnion types.TypeRef

func (m MapOfStringToSetOfPhotoUnion) TypeRef() types.TypeRef {
	return __typeRefForMapOfStringToSetOfPhotoUnion
}

func init() {
	__typeRefForMapOfStringToSetOfPhotoUnion = types.MakeCompoundTypeRef("", types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeCompoundTypeRef("", types.SetKind, types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0)))
	types.RegisterFromValFunction(__typeRefForMapOfStringToSetOfPhotoUnion, func(v types.Value) types.NomsValue {
		return MapOfStringToSetOfPhotoUnionFromVal(v)
	})
}

func (m MapOfStringToSetOfPhotoUnion) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToSetOfPhotoUnion) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToSetOfPhotoUnion) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToSetOfPhotoUnion) Get(p string) SetOfPhotoUnion {
	return SetOfPhotoUnionFromVal(m.m.Get(types.NewString(p)))
}

func (m MapOfStringToSetOfPhotoUnion) MaybeGet(p string) (SetOfPhotoUnion, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewSetOfPhotoUnion(), false
	}
	return SetOfPhotoUnionFromVal(v), ok
}

func (m MapOfStringToSetOfPhotoUnion) Set(k string, v SetOfPhotoUnion) MapOfStringToSetOfPhotoUnion {
	return MapOfStringToSetOfPhotoUnion{m.m.Set(types.NewString(k), v.NomsValue())}
}

// TODO: Implement SetM?

func (m MapOfStringToSetOfPhotoUnion) Remove(p string) MapOfStringToSetOfPhotoUnion {
	return MapOfStringToSetOfPhotoUnion{m.m.Remove(types.NewString(p))}
}

type MapOfStringToSetOfPhotoUnionIterCallback func(k string, v SetOfPhotoUnion) (stop bool)

func (m MapOfStringToSetOfPhotoUnion) Iter(cb MapOfStringToSetOfPhotoUnionIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), SetOfPhotoUnionFromVal(v))
	})
}

type MapOfStringToSetOfPhotoUnionIterAllCallback func(k string, v SetOfPhotoUnion)

func (m MapOfStringToSetOfPhotoUnion) IterAll(cb MapOfStringToSetOfPhotoUnionIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), SetOfPhotoUnionFromVal(v))
	})
}

type MapOfStringToSetOfPhotoUnionFilterCallback func(k string, v SetOfPhotoUnion) (keep bool)

func (m MapOfStringToSetOfPhotoUnion) Filter(cb MapOfStringToSetOfPhotoUnionFilterCallback) MapOfStringToSetOfPhotoUnion {
	nm := NewMapOfStringToSetOfPhotoUnion()
	m.IterAll(func(k string, v SetOfPhotoUnion) {
		if cb(k, v) {
			nm = nm.Set(k, v)
		}
	})
	return nm
}

// SetOfPhotoUnion

type SetOfPhotoUnion struct {
	s types.Set
}

func NewSetOfPhotoUnion() SetOfPhotoUnion {
	return SetOfPhotoUnion{types.NewSet()}
}

type SetOfPhotoUnionDef map[PhotoUnionDef]bool

func (def SetOfPhotoUnionDef) New() SetOfPhotoUnion {
	l := make([]types.Value, len(def))
	i := 0
	for d, _ := range def {
		l[i] = d.New().NomsValue()
		i++
	}
	return SetOfPhotoUnion{types.NewSet(l...)}
}

func (s SetOfPhotoUnion) Def() SetOfPhotoUnionDef {
	def := make(map[PhotoUnionDef]bool, s.Len())
	s.s.Iter(func(v types.Value) bool {
		def[PhotoUnionFromVal(v).Def()] = true
		return false
	})
	return def
}

func SetOfPhotoUnionFromVal(p types.Value) SetOfPhotoUnion {
	return SetOfPhotoUnion{p.(types.Set)}
}

func (s SetOfPhotoUnion) NomsValue() types.Value {
	return s.s
}

func (s SetOfPhotoUnion) Equals(other types.Value) bool {
	if other, ok := other.(SetOfPhotoUnion); ok {
		return s.s.Equals(other.s)
	}
	return false
}

func (s SetOfPhotoUnion) Ref() ref.Ref {
	return s.s.Ref()
}

func (s SetOfPhotoUnion) Chunks() (futures []types.Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.s.Chunks()...)
	return
}

// A Noms Value that describes SetOfPhotoUnion.
var __typeRefForSetOfPhotoUnion types.TypeRef

func (m SetOfPhotoUnion) TypeRef() types.TypeRef {
	return __typeRefForSetOfPhotoUnion
}

func init() {
	__typeRefForSetOfPhotoUnion = types.MakeCompoundTypeRef("", types.SetKind, types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0))
	types.RegisterFromValFunction(__typeRefForSetOfPhotoUnion, func(v types.Value) types.NomsValue {
		return SetOfPhotoUnionFromVal(v)
	})
}

func (s SetOfPhotoUnion) Empty() bool {
	return s.s.Empty()
}

func (s SetOfPhotoUnion) Len() uint64 {
	return s.s.Len()
}

func (s SetOfPhotoUnion) Has(p PhotoUnion) bool {
	return s.s.Has(p.NomsValue())
}

type SetOfPhotoUnionIterCallback func(p PhotoUnion) (stop bool)

func (s SetOfPhotoUnion) Iter(cb SetOfPhotoUnionIterCallback) {
	s.s.Iter(func(v types.Value) bool {
		return cb(PhotoUnionFromVal(v))
	})
}

type SetOfPhotoUnionIterAllCallback func(p PhotoUnion)

func (s SetOfPhotoUnion) IterAll(cb SetOfPhotoUnionIterAllCallback) {
	s.s.IterAll(func(v types.Value) {
		cb(PhotoUnionFromVal(v))
	})
}

type SetOfPhotoUnionFilterCallback func(p PhotoUnion) (keep bool)

func (s SetOfPhotoUnion) Filter(cb SetOfPhotoUnionFilterCallback) SetOfPhotoUnion {
	ns := NewSetOfPhotoUnion()
	s.IterAll(func(v PhotoUnion) {
		if cb(v) {
			ns = ns.Insert(v)
		}
	})
	return ns
}

func (s SetOfPhotoUnion) Insert(p ...PhotoUnion) SetOfPhotoUnion {
	return SetOfPhotoUnion{s.s.Insert(s.fromElemSlice(p)...)}
}

func (s SetOfPhotoUnion) Remove(p ...PhotoUnion) SetOfPhotoUnion {
	return SetOfPhotoUnion{s.s.Remove(s.fromElemSlice(p)...)}
}

func (s SetOfPhotoUnion) Union(others ...SetOfPhotoUnion) SetOfPhotoUnion {
	return SetOfPhotoUnion{s.s.Union(s.fromStructSlice(others)...)}
}

func (s SetOfPhotoUnion) Subtract(others ...SetOfPhotoUnion) SetOfPhotoUnion {
	return SetOfPhotoUnion{s.s.Subtract(s.fromStructSlice(others)...)}
}

func (s SetOfPhotoUnion) Any() PhotoUnion {
	return PhotoUnionFromVal(s.s.Any())
}

func (s SetOfPhotoUnion) fromStructSlice(p []SetOfPhotoUnion) []types.Set {
	r := make([]types.Set, len(p))
	for i, v := range p {
		r[i] = v.s
	}
	return r
}

func (s SetOfPhotoUnion) fromElemSlice(p []PhotoUnion) []types.Value {
	r := make([]types.Value, len(p))
	for i, v := range p {
		r[i] = v.NomsValue()
	}
	return r
}
