package types

import (
	"github.com/attic-labs/noms/go/d"
	flag "github.com/juju/gnuflag"
)

var enableTypeSimplification = false

func RegisterTypeSimplificationFlags(flags *flag.FlagSet) {
	flags.BoolVar(&enableTypeSimplification, "type-simplificatino", false, "enables type simplification (see https://github.com/attic-labs/noms/issues/2995)")
}

func accreteTypes(t ...*Type) *Type {
	if enableTypeSimplification {
		return makeSimplifiedUnion(t...)
	} else {
		return MakeUnionType(t...)
	}
}

// makeSimplifiedUnion returns a type that is a supertype of all the input types, but is much
// smaller and less complex than a straight union of all those types would be.
//
// The resulting type is guaranteed to:
// a. be a supertype of all input types
// b. have no direct children that are unions
// c. have at most one element each of kind Ref, Set, List, and Map
// d. have at most one struct element with a given name
// e. all union types reachable from it also fulfill b-e
//
// The simplification is created roughly as follows:
//
// - The input types are deduplicated
// - Any unions in the input set are "flattened" into the input set
// - The inputs are grouped into categories:
//    - ref
//    - list
//    - set
//    - map
//    - struct, by name (each unique struct name will have its own group)
// - The ref, set, and list groups are collapsed like so:
//     {Ref<A>,Ref<B>,...} -> Ref<A|B|...>
// - The map group is collapsed like so:
//     {Map<K1,V1>|Map<K2,V2>...} -> Map<K1|K2,V1|V2>
// - Each struct group is collapsed like so:
//     {struct{foo:number,bar:string}, struct{bar:bool, baz:blob}} ->
//       struct{bar:string|blob}
//
// Anytime any of the above cases generates a union as output, the same process
// is applied to that union recursively.
func makeSimplifiedUnion(in ...*Type) *Type {
	d.Chk.True(len(in) > 0)

	ts := typeset{}
	for _, t := range in {
		// De-cycle so that we handle cycles explicitly below. Otherwise, we would implicitly crawl
		// cycles and recurse forever.
		t := ToUnresolvedType(t)
		ts[t] = struct{}{}
	}

	r := makeSimplifiedUnionImpl(ts)

	if r.HasUnresolvedCycle() {
		r = resolveStructCycles(r, nil)
	}

	return r
}

// typeset is a helper that aggregates the unique set of input types for this algorithm, flattening
// any unions recursively.
type typeset map[*Type]struct{}

func (ts typeset) Add(t *Type) {
	switch t.Kind() {
	case UnionKind:
		for _, et := range t.Desc.(CompoundDesc).ElemTypes {
			ts.Add(et)
		}
	default:
		ts[t] = struct{}{}
	}
}

func newTypeset(t ...*Type) typeset {
	ts := typeset{}
	for _, t := range t {
		ts.Add(t)
	}
	return ts
}

// makeSimplifiedUnionImpl is an implementation detail of MakeSimplifiedUnion.
// Warning: Do not call this directly. It assumes its input has been
// de-cycled using ToUnresolvedType() and will inifinitely recurse otherwise
// on cyclic types otherwise.
func makeSimplifiedUnionImpl(in typeset) *Type {
	type how struct {
		k NomsKind
		n string
	}

	out := []*Type{}
	groups := map[how]typeset{}
	for t, _ := range in {
		var h how
		switch t.Kind() {
		case RefKind, SetKind, ListKind, MapKind:
			h = how{k: t.Kind()}
		case StructKind:
			h = how{k: t.Kind(), n: t.Desc.(StructDesc).Name}
		default:
			out = append(out, t)
			continue
		}
		g := groups[h]
		if g == nil {
			g = typeset{}
			groups[h] = g
		}
		g.Add(t)
	}

	for h, ts := range groups {
		if len(ts) == 1 {
			for t, _ := range ts {
				out = append(out, t)
			}
			continue
		}

		var r *Type
		switch h.k {
		case RefKind:
			r = simplifyRefs(ts)
		case SetKind:
			r = simplifySets(ts)
		case ListKind:
			r = simplifyLists(ts)
		case MapKind:
			r = simplifyMaps(ts)
		case StructKind:
			r = simplifyStructs(h.n, ts)
		}
		out = append(out, r)
	}

	if len(out) == 1 {
		return out[0]
	}

	return MakeUnionType(out...)
}

func simplifyRefs(ts typeset) *Type {
	return simplifyContainers(RefKind, MakeRefType, ts)
}

func simplifySets(ts typeset) *Type {
	return simplifyContainers(SetKind, MakeSetType, ts)
}

func simplifyLists(ts typeset) *Type {
	return simplifyContainers(ListKind, MakeListType, ts)
}

func simplifyContainers(expectedKind NomsKind, makeContainer func(elem *Type) *Type, ts typeset) *Type {
	elemTypes := typeset{}
	for t, _ := range ts {
		d.Chk.Equal(expectedKind, t.Kind())
		elemTypes.Add(t.Desc.(CompoundDesc).ElemTypes[0])
	}
	return makeContainer(makeSimplifiedUnionImpl(elemTypes))
}

func simplifyMaps(ts typeset) *Type {
	keyTypes := typeset{}
	valTypes := typeset{}
	for t, _ := range ts {
		d.Chk.Equal(MapKind, t.Kind())
		desc := t.Desc.(CompoundDesc)
		keyTypes.Add(desc.ElemTypes[0])
		valTypes.Add(desc.ElemTypes[1])
	}
	return MakeMapType(
		makeSimplifiedUnionImpl(keyTypes),
		makeSimplifiedUnionImpl(valTypes))
}

func simplifyStructs(expectedName string, ts typeset) *Type {
	commonFields := map[string]typeset{}

	first := true
	for t, _ := range ts {
		d.Chk.Equal(StructKind, t.Kind())
		desc := t.Desc.(StructDesc)
		d.Chk.Equal(expectedName, desc.Name)
		if first {
			for _, f := range desc.fields {
				ts := typeset{}
				ts.Add(f.t)
				commonFields[f.name] = ts
			}
		} else {
			for n, ts := range commonFields {
				t := desc.Field(n)
				if t != nil {
					ts.Add(t)
				} else {
					delete(commonFields, n)
				}
			}
		}
		first = false
	}

	fm := FieldMap{}
	for n, ts := range commonFields {
		fm[n] = makeSimplifiedUnionImpl(ts)
	}

	return MakeStructTypeFromFields(expectedName, fm)
}
