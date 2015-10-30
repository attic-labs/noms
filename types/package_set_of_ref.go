// This file was generated by a slightly modified nomdl/codegen
// To generate this I added support for `Package` in the NomDL parser (Just add `Package` after `UInt32` etc)

package types

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

// SetOfRefOfPackage

type SetOfRefOfPackage struct {
	s   Set
	ref *ref.Ref
}

func NewSetOfRefOfPackage() SetOfRefOfPackage {
	return SetOfRefOfPackage{NewSet(), &ref.Ref{}}
}

type SetOfRefOfPackageDef map[ref.Ref]bool

func (def SetOfRefOfPackageDef) New() SetOfRefOfPackage {
	l := make([]Value, len(def))
	i := 0
	for d, _ := range def {
		l[i] = NewRefOfPackage(d)
		i++
	}
	return SetOfRefOfPackage{NewSet(l...), &ref.Ref{}}
}

func (s SetOfRefOfPackage) Def() SetOfRefOfPackageDef {
	def := make(map[ref.Ref]bool, s.Len())
	s.s.Iter(func(v Value) bool {
		def[v.(RefOfPackage).TargetRef()] = true
		return false
	})
	return def
}

func SetOfRefOfPackageFromVal(val Value) SetOfRefOfPackage {
	// TODO: Do we still need FromVal?
	if val, ok := val.(SetOfRefOfPackage); ok {
		return val
	}
	return SetOfRefOfPackage{val.(Set), &ref.Ref{}}
}

func (s SetOfRefOfPackage) InternalImplementation() Set {
	return s.s
}

func (s SetOfRefOfPackage) Equals(other Value) bool {
	return other != nil && s.Ref() == other.Ref()
}

func (s SetOfRefOfPackage) Ref() ref.Ref {
	return EnsureRef(s.ref, s)
}

func (s SetOfRefOfPackage) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, s.TypeRef().Chunks()...)
	chunks = append(chunks, s.s.Chunks()...)
	return
}

// A Noms Value that describes SetOfRefOfPackage.
var __typeRefForSetOfRefOfPackage TypeRef

func (m SetOfRefOfPackage) TypeRef() TypeRef {
	return __typeRefForSetOfRefOfPackage
}

func init() {
	__typeRefForSetOfRefOfPackage = MakeCompoundTypeRef(SetKind, MakeCompoundTypeRef(RefKind, MakePrimitiveTypeRef(PackageKind)))
	RegisterFromValFunction(__typeRefForSetOfRefOfPackage, func(v Value) Value {
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
	return s.s.Has(p)
}

type SetOfRefOfPackageIterCallback func(p RefOfPackage) (stop bool)

func (s SetOfRefOfPackage) Iter(cb SetOfRefOfPackageIterCallback) {
	s.s.Iter(func(v Value) bool {
		return cb(v.(RefOfPackage))
	})
}

type SetOfRefOfPackageIterAllCallback func(p RefOfPackage)

func (s SetOfRefOfPackage) IterAll(cb SetOfRefOfPackageIterAllCallback) {
	s.s.IterAll(func(v Value) {
		cb(v.(RefOfPackage))
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
	return SetOfRefOfPackage{s.s.Insert(s.fromElemSlice(p)...), &ref.Ref{}}
}

func (s SetOfRefOfPackage) Remove(p ...RefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Remove(s.fromElemSlice(p)...), &ref.Ref{}}
}

func (s SetOfRefOfPackage) Union(others ...SetOfRefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Union(s.fromStructSlice(others)...), &ref.Ref{}}
}

func (s SetOfRefOfPackage) Subtract(others ...SetOfRefOfPackage) SetOfRefOfPackage {
	return SetOfRefOfPackage{s.s.Subtract(s.fromStructSlice(others)...), &ref.Ref{}}
}

func (s SetOfRefOfPackage) Any() RefOfPackage {
	return s.s.Any().(RefOfPackage)
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
		r[i] = v
	}
	return r
}

// RefOfPackage

type RefOfPackage struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfPackage(target ref.Ref) RefOfPackage {
	return RefOfPackage{target, &ref.Ref{}}
}

func (r RefOfPackage) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfPackage) Ref() ref.Ref {
	return EnsureRef(r.ref, r)
}

func (r RefOfPackage) Equals(other Value) bool {
	return other != nil && r.Ref() == other.Ref()
}

func (r RefOfPackage) Chunks() []ref.Ref {
	return r.TypeRef().Chunks()
}

func RefOfPackageFromVal(val Value) RefOfPackage {
	// TODO: Do we still need FromVal?
	if val, ok := val.(RefOfPackage); ok {
		return val
	}
	return NewRefOfPackage(val.(Ref).TargetRef())
}

// A Noms Value that describes RefOfPackage.
var __typeRefForRefOfPackage TypeRef

func (m RefOfPackage) TypeRef() TypeRef {
	return __typeRefForRefOfPackage
}

func init() {
	__typeRefForRefOfPackage = MakeCompoundTypeRef(RefKind, MakePrimitiveTypeRef(PackageKind))
	RegisterFromValFunction(__typeRefForRefOfPackage, func(v Value) Value {
		return RefOfPackageFromVal(v)
	})
}

func (r RefOfPackage) TargetValue(cs chunks.ChunkSource) Package {
	return ReadValue(r.target, cs).(Package)
}

func (r RefOfPackage) SetTargetValue(val Package, cs chunks.ChunkSink) RefOfPackage {
	return NewRefOfPackage(WriteValue(val, cs))
}
