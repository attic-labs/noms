package types

import "github.com/attic-labs/noms/go/d"

// TypesIntersect returns true if types |a| and |b| have common elements.
// It is useful for determining whether a subset of values can be extracted
// from one object to produce another object.
//
// The rules for determining whether |a| and |b| intersect are:
//    - if either type is Value, return true
//    - if either type is Union, return true iff at least one variant of |a| intersects with one variant of |b|
//    - if |a| & |b| are not the same kind, return false
//    - else
//      - if both are structs, return true iff they have the same name, share a field name and the type
//        of that field intersects
//      - if both are refs, sets or lists, return true iff the element type intersects
//      - if both are maps, return true iff they have a key and value type that intersect
//      - else return true
func TypesIntersect(a, b *Type) bool {
	// Avoid cycles internally.
	return typesIntersectImpl(ToUnresolvedType(a), ToUnresolvedType(b))
}

func typesIntersectImpl(a, b *Type) bool {
	if a.Kind() == ValueKind || b.Kind() == ValueKind {
		return true
	}
	if a.Kind() == UnionKind {
		return unionIntersects(a, b)
	}
	if b.Kind() == UnionKind {
		return unionIntersects(b, a)
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch k := a.Kind(); k {
	case StructKind:
		return structsIntersect(a, b)
	case ListKind, SetKind, RefKind:
		return containersIntersect(k, a, b)
	case MapKind:
		return mapsIntersect(a, b)
	default:
		return true
	}

}

func unionIntersects(a, b *Type) bool {
	d.Chk.True(UnionKind == a.Desc.Kind())
	for _, e := range a.Desc.(CompoundDesc).ElemTypes {
		if typesIntersectImpl(e, b) {
			return true
		}
	}
	return false
}

func containersIntersect(kind NomsKind, a, b *Type) bool {
	d.Chk.True(kind == a.Desc.Kind() && kind == b.Desc.Kind())
	return typesIntersectImpl(a.Desc.(CompoundDesc).ElemTypes[0], b.Desc.(CompoundDesc).ElemTypes[0])
}

// TODO: consider requiring that both maps share a key type
func mapsIntersect(a, b *Type) bool {
	d.Chk.True(MapKind == a.Desc.Kind() && MapKind == b.Desc.Kind())
	aDesc, bDesc := a.Desc.(CompoundDesc), b.Desc.(CompoundDesc)
	if !typesIntersectImpl(aDesc.ElemTypes[0], bDesc.ElemTypes[0]) {
		return false
	}
	return typesIntersectImpl(aDesc.ElemTypes[1], bDesc.ElemTypes[1])
}

func structsIntersect(a, b *Type) bool {
	d.Chk.True(StructKind == a.Kind() && StructKind == b.Kind())
	aDesc := a.Desc.(StructDesc)
	bDesc := b.Desc.(StructDesc)
	if aDesc.Name != bDesc.Name {
		return false
	}
	for _, f := range aDesc.fields {
		t := bDesc.Field(f.name)
		if t == nil {
			continue
		}
		if typesIntersectImpl(f.t, t) {
			return true
		}
	}
	return false
}
