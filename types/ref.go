package types

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

type Ref struct {
	target ref.Ref
	t      TypeRef
	ref    *ref.Ref
}

func NewRef(target ref.Ref) Ref {
	return newRef(target, refTypeRef)
}

func newRef(target ref.Ref, t TypeRef) Ref {
	return Ref{target, t, &ref.Ref{}}
}

func (r Ref) Equals(other Value) bool {
	return other != nil && r.Ref() == other.Ref()
}

func (r Ref) Ref() ref.Ref {
	return EnsureRef(r.ref, r)
}

func (r Ref) Chunks() []ref.Ref {
	return nil
}

func (r Ref) TargetRef() ref.Ref {
	return r.target
}

var refTypeRef = MakeCompoundTypeRef(RefKind, MakePrimitiveTypeRef(ValueKind))

func (r Ref) TypeRef() TypeRef {
	return r.t
}

func init() {
	RegisterFromValFunction(refTypeRef, func(v Value) Value {
		return v.(Ref)
	})
}

func (r Ref) TargetValue(cs chunks.ChunkSource) Value {
	return ReadValue(r.target, cs)
}

func (r Ref) SetTargetValue(val Value, cs chunks.ChunkSink) Ref {
	return newRef(WriteValue(val, cs), r.t)
}
