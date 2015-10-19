// This file was generated by nomdl/codegen and then had references to this package (types) removed by hand. The $type field of Package was also manually set to the TypeRef that describes a Package directly.

package types

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

// SetOfPackage

type SetOfPackage struct {
	s Set
}

func NewSetOfPackage() SetOfPackage {
	return SetOfPackage{NewSet()}
}

func SetOfPackageFromVal(p Value) SetOfPackage {
	return SetOfPackage{p.(Set)}
}

func (s SetOfPackage) NomsValue() Value {
	return s.s
}

func (s SetOfPackage) Equals(other Value) bool {
	if other, ok := other.(SetOfPackage); ok {
		return s.s.Equals(other.s)
	}
	return false
}

func (s SetOfPackage) Ref() ref.Ref {
	return s.s.Ref()
}

func (s SetOfPackage) Chunks() (futures []Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.s.Chunks()...)
	return
}

// A Noms Value that describes SetOfPackage.
var __typeRefForSetOfPackage TypeRef

func (m SetOfPackage) TypeRef() TypeRef {
	return __typeRefForSetOfPackage
}

func init() {
	__typeRefForSetOfPackage = MakeCompoundTypeRef("", SetKind, MakePrimitiveTypeRef(PackageKind))
	RegisterFromValFunction(__typeRefForSetOfPackage, func(v Value) NomsValue {
		return SetOfPackageFromVal(v)
	})
}

func (s SetOfPackage) Empty() bool {
	return s.s.Empty()
}

func (s SetOfPackage) Len() uint64 {
	return s.s.Len()
}

func (s SetOfPackage) Has(p Package) bool {
	return s.s.Has(p)
}

type SetOfPackageIterCallback func(p Package) (stop bool)

func (s SetOfPackage) Iter(cb SetOfPackageIterCallback) {
	s.s.Iter(func(v Value) bool {
		return cb(v.(Package))
	})
}

type SetOfPackageIterAllCallback func(p Package)

func (s SetOfPackage) IterAll(cb SetOfPackageIterAllCallback) {
	s.s.IterAll(func(v Value) {
		cb(v.(Package))
	})
}

type SetOfPackageFilterCallback func(p Package) (keep bool)

func (s SetOfPackage) Filter(cb SetOfPackageFilterCallback) SetOfPackage {
	ns := NewSetOfPackage()
	s.IterAll(func(v Package) {
		if cb(v) {
			ns = ns.Insert(v)
		}
	})
	return ns
}

func (s SetOfPackage) Insert(p ...Package) SetOfPackage {
	return SetOfPackage{s.s.Insert(s.fromElemSlice(p)...)}
}

func (s SetOfPackage) Remove(p ...Package) SetOfPackage {
	return SetOfPackage{s.s.Remove(s.fromElemSlice(p)...)}
}

func (s SetOfPackage) Union(others ...SetOfPackage) SetOfPackage {
	return SetOfPackage{s.s.Union(s.fromStructSlice(others)...)}
}

func (s SetOfPackage) Subtract(others ...SetOfPackage) SetOfPackage {
	return SetOfPackage{s.s.Subtract(s.fromStructSlice(others)...)}
}

func (s SetOfPackage) Any() Package {
	return s.s.Any().(Package)
}

func (s SetOfPackage) fromStructSlice(p []SetOfPackage) []Set {
	r := make([]Set, len(p))
	for i, v := range p {
		r[i] = v.s
	}
	return r
}

func (s SetOfPackage) fromElemSlice(p []Package) []Value {
	r := make([]Value, len(p))
	for i, v := range p {
		r[i] = v
	}
	return r
}

// SetOfRefOfPackage

type SetOfRefOfPackage struct {
	s Set
}

func NewSetOfRefOfPackage() SetOfRefOfPackage {
	return SetOfRefOfPackage{NewSet()}
}

type SetOfRefOfPackageDef map[ref.Ref]bool

func (def SetOfRefOfPackageDef) New() SetOfRefOfPackage {
	l := make([]Value, len(def))
	i := 0
	for d, _ := range def {
		l[i] = Ref{R: d}
		i++
	}
	return SetOfRefOfPackage{NewSet(l...)}
}

func (s SetOfRefOfPackage) Def() SetOfRefOfPackageDef {
	def := make(map[ref.Ref]bool, s.Len())
	s.s.Iter(func(v Value) bool {
		def[v.Ref()] = true
		return false
	})
	return def
}

func SetOfRefOfPackageFromVal(p Value) SetOfRefOfPackage {
	return SetOfRefOfPackage{p.(Set)}
}

func (s SetOfRefOfPackage) NomsValue() Value {
	return s.s
}

func (s SetOfRefOfPackage) Equals(other Value) bool {
	if other, ok := other.(SetOfRefOfPackage); ok {
		return s.s.Equals(other.s)
	}
	return false
}

func (s SetOfRefOfPackage) Ref() ref.Ref {
	return s.s.Ref()
}

func (s SetOfRefOfPackage) Chunks() (futures []Future) {
	futures = append(futures, s.TypeRef().Chunks()...)
	futures = append(futures, s.s.Chunks()...)
	return
}

// A Noms Value that describes SetOfRefOfPackage.
var __typeRefForSetOfRefOfPackage TypeRef

func (m SetOfRefOfPackage) TypeRef() TypeRef {
	return __typeRefForSetOfRefOfPackage
}

func init() {
	__typeRefForSetOfRefOfPackage = MakeCompoundTypeRef("", SetKind, MakeCompoundTypeRef("", RefKind, MakePrimitiveTypeRef(PackageKind)))
	RegisterFromValFunction(__typeRefForSetOfRefOfPackage, func(v Value) NomsValue {
		return SetOfRefOfPackageFromVal(v)
	})
}

func (s SetOfRefOfPackage) Empty() bool {
	return s.s.Empty()
}

func (s SetOfRefOfPackage) Len() uint64 {
	return s.s.Len()
}

func (s SetOfRefOfPackage) Has(p RefOfPackage) bool {
	return s.s.Has(p.NomsValue())
}

type SetOfRefOfPackageIterCallback func(p RefOfPackage) (stop bool)

func (s SetOfRefOfPackage) Iter(cb SetOfRefOfPackageIterCallback) {
	s.s.Iter(func(v Value) bool {
		return cb(RefOfPackageFromVal(v))
	})
}

type SetOfRefOfPackageIterAllCallback func(p RefOfPackage)

func (s SetOfRefOfPackage) IterAll(cb SetOfRefOfPackageIterAllCallback) {
	// IT'S A HAAAAAACK!
	// Currently, ReadValue() automatically derefs refs. So, in some cases the value passed to the callback by s.s.IterAll() is actually a Package instead of Ref(Package). This works around that until we've fixed it.
	s.s.IterAll(func(v Value) {
		if r, ok := v.(Ref); ok {
			cb(RefOfPackageFromVal(r))
			return
		}
		cb(RefOfPackage{v.(Package).Ref()})
	})
}

type SetOfRefOfPackageFilterCallback func(p RefOfPackage) (keep bool)

func (s SetOfRefOfPackage) Filter(cb SetOfRefOfPackageFilterCallback) SetOfRefOfPackage {
	ns := NewSetOfRefOfPackage()
	s.IterAll(func(v RefOfPackage) {
		if cb(v) {
			ns = ns.Insert(v)
		}
	})
	return ns
}

func (s SetOfRefOfPackage) Insert(p ...RefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Insert(s.fromElemSlice(p)...)}
}

func (s SetOfRefOfPackage) Remove(p ...RefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Remove(s.fromElemSlice(p)...)}
}

func (s SetOfRefOfPackage) Union(others ...SetOfRefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Union(s.fromStructSlice(others)...)}
}

func (s SetOfRefOfPackage) Subtract(others ...SetOfRefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Subtract(s.fromStructSlice(others)...)}
}

func (s SetOfRefOfPackage) Any() RefOfPackage {
	// IT'S A HAAAAAACK!
	// Currently, ReadValue() automatically derefs refs. So, in some cases the value returned by s.s.Any() is actually a Package instead of Ref(Package). This works around that until we've fixed it.
	return RefOfPackage{s.s.Any().(Package).Ref()}
}

func (s SetOfRefOfPackage) fromStructSlice(p []SetOfRefOfPackage) []Set {
	r := make([]Set, len(p))
	for i, v := range p {
		r[i] = v.s
	}
	return r
}

func (s SetOfRefOfPackage) fromElemSlice(p []RefOfPackage) []Value {
	r := make([]Value, len(p))
	for i, v := range p {
		r[i] = v.NomsValue()
	}
	return r
}

// RefOfPackage

type RefOfPackage struct {
	r ref.Ref
}

func NewRefOfPackage(r ref.Ref) RefOfPackage {
	return RefOfPackage{r}
}

func (r RefOfPackage) Ref() ref.Ref {
	return r.r
}

func (r RefOfPackage) Equals(other Value) bool {
	if other, ok := other.(RefOfPackage); ok {
		return r.r == other.r
	}
	return false
}

func (r RefOfPackage) Chunks() []Future {
	return r.TypeRef().Chunks()
}

func (r RefOfPackage) NomsValue() Value {
	return Ref{R: r.r}
}

func RefOfPackageFromVal(p Value) RefOfPackage {
	return RefOfPackage{p.(Ref).Ref()}
}

// A Noms Value that describes RefOfPackage.
var __typeRefForRefOfPackage TypeRef

func (m RefOfPackage) TypeRef() TypeRef {
	return __typeRefForRefOfPackage
}

func init() {
	__typeRefForRefOfPackage = MakeCompoundTypeRef("", RefKind, MakePrimitiveTypeRef(PackageKind))
	RegisterFromValFunction(__typeRefForRefOfPackage, func(v Value) NomsValue {
		return RefOfPackageFromVal(v)
	})
}

func (r RefOfPackage) GetValue(cs chunks.ChunkSource) Package {
	return ReadValue(r.r, cs).(Package)
}

func (r RefOfPackage) SetValue(val Package, cs chunks.ChunkSink) RefOfPackage {
	ref := WriteValue(val, cs)
	return RefOfPackage{ref}
}
