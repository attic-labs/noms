package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

// simplifyType returns a type that is a supertype of all the input types but is much
// smaller and less complex than a straight union of all those types would be.
//
// The resulting type is guaranteed to:
// a. be a supertype of all input types
// b. have no direct children that are unions
// c. have at most one element each of kind Ref, Set, List, and Map
// d. have at most one struct element with a given name
// e. all union types reachable from it also fulfill b-e
// f. all named unions are pointing at the same simplified struct
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
//     {struct{foo:number,bar:string}, struct{bar:blob, baz:bool}} ->
//       struct{foo?:number,bar:string|blob,baz?:bool}
//
// All the above rules are applied recursively.
func simplifyType(t *Type, intersectStructs bool) *Type {
	simplifier := newSimplifier(intersectStructs)
	rv, changed, hasCycles := simplifier.simplify(t)
	if !changed {
		return t
	}
	if len(simplifier.seenByName) > 0 {
		d.PanicIfFalse(len(simplifier.seenStructs) > 0)
		d.PanicIfFalse(changed)
		d.PanicIfFalse(hasCycles)

		env := make(map[string]*Type, len(simplifier.seenByName))
		for name, ts := range simplifier.seenByName {
			env[name] = simplifier.mergeStructTypes(name, ts)
		}

		return simplifier.resolveStructCycles(rv, env, map[string]struct{}{})
	}

	return rv
}

// typeset is a helper that aggregates the unique set of input types for this algorithm, flattening
// any unions recursively.
type typeset map[*Type]struct{}

func (ts typeset) add(t *Type) {
	switch t.TargetKind() {
	case UnionKind:
		for _, et := range t.Desc.(CompoundDesc).ElemTypes {
			ts.add(et)
		}
	default:
		ts[t] = struct{}{}
	}
}

func (ts typeset) has(t *Type) bool {
	_, ok := ts[t]
	return ok
}

type typeSimplifier struct {
	seenStructs         typeset
	seenByName          map[string]typeset
	seenNamedCycleTypes map[string]*Type
	intersectStructs    bool
}

func newSimplifier(intersectStructs bool) *typeSimplifier {
	return &typeSimplifier{
		typeset{},
		map[string]typeset{},
		map[string]*Type{},
		intersectStructs,
	}
}

func (simplifier *typeSimplifier) bucketElements(in typeset) *Type {
	type how struct {
		k NomsKind
		n string
	}
	out := make(typeSlice, 0, len(in))
	groups := map[how]typeset{}
	for t := range in {
		var h how
		switch t.TargetKind() {
		case RefKind, SetKind, ListKind, MapKind:
			h = how{k: t.TargetKind()}
		case StructKind:
			h = how{k: t.TargetKind(), n: t.Desc.(StructDesc).Name}
		default:
			out = append(out, t)
			continue
		}
		g := groups[h]
		if g == nil {
			g = typeset{}
			groups[h] = g
		}
		g.add(t)
	}

	for h, ts := range groups {
		if len(ts) == 1 {
			for t := range ts {
				out = append(out, t)
			}
			continue
		}

		var r *Type
		switch h.k {
		case ListKind, RefKind, SetKind:
			r = simplifier.mergeCompoundTypesForUnion(h.k, ts)
		case MapKind:
			r = simplifier.mergeMapTypesForUnion(ts)
		case StructKind:
			r = simplifier.mergeStructTypes(h.n, ts)
		}
		out = append(out, r)
	}

	if len(out) == 1 {
		return out[0]
	}

	sort.Sort(out)

	return makeCompoundType(UnionKind, out...)
}

func (simplifier *typeSimplifier) simplifyStructFields(in []structTypeFields) structTypeFields {
	// We gather all the fields/types into allFields. If the number of
	// times a field name is present is less that then number of types we
	// are simplifying then the field must be optional.
	// If we see an optional field we do not increment the count for it and
	// it will be treated as optional in the end.

	// If intersectStructs is true we need to pick the more restrictive version (n: T over n?: T).
	type fieldTypeInfo struct {
		anyNonOptional bool
		count          int
		ts             typeSlice
	}
	allFields := map[string]fieldTypeInfo{}

	for _, ff := range in {
		for _, f := range ff {
			name := f.Name
			t := f.Type
			optional := f.Optional
			fti, ok := allFields[name]
			if !ok {
				fti = fieldTypeInfo{
					ts: typeSlice{},
				}
			}
			fti.ts = append(fti.ts, t)
			if !optional {
				fti.count++
				fti.anyNonOptional = true
			}
			allFields[name] = fti
		}
	}

	count := len(in)
	fields := make(structTypeFields, 0, count)
	for name, fti := range allFields {
		nt, _, _ := simplifier.simplify(makeCompoundType(UnionKind, fti.ts...))
		fields = append(fields, StructField{
			Name:     name,
			Type:     nt,
			Optional: !(simplifier.intersectStructs && fti.anyNonOptional) && fti.count < count,
		})
	}

	sort.Sort(fields)

	return fields
}

// makeCycleType makes sure we fold same named cycle types into one type.
func (simplifier *typeSimplifier) makeCycleType(name string) *Type {
	// TODO: Feel like this should not be needed. Should be grouped together...
	if t, ok := simplifier.seenNamedCycleTypes[name]; ok {
		return t
	}

	t := MakeCycleType(name)
	simplifier.seenNamedCycleTypes[name] = t
	return t
}

func (simplifier *typeSimplifier) simplify(t *Type) (*Type, bool, bool) {
	if simplifier.seenStructs.has(t) {
		// Already handled.
		name := t.Desc.(StructDesc).Name
		return simplifier.makeCycleType(name), true, true
	}

	k := t.TargetKind()
	switch k {
	case BoolKind, NumberKind, StringKind, BlobKind, ValueKind, TypeKind:
		return t, false, false

	case ListKind, MapKind, RefKind, SetKind:
		elemTypes := t.Desc.(CompoundDesc).ElemTypes
		newElemTypes := make(typeSlice, len(elemTypes))
		changed, hasCycles := false, false
		for i, et := range elemTypes {
			nt, c, hc := simplifier.simplify(et)
			changed = changed || c
			hasCycles = hasCycles || hc
			newElemTypes[i] = nt
		}
		if !changed {
			return t, false, hasCycles
		}
		return makeCompoundType(k, newElemTypes...), true, hasCycles

	case StructKind:
		desc := t.Desc.(StructDesc)
		name := desc.Name

		if name != "" {
			simplifier.seenStructs.add(t)
		}

		changed, hasCycles := false, false
		fields := make(structTypeFields, len(desc.fields))
		for i, f := range desc.fields {
			ft, c, hc := simplifier.simplify(f.Type)
			if c {
				changed = true
				fields[i] = StructField{f.Name, ft, f.Optional}
			} else {
				fields[i] = desc.fields[i]
			}
			hasCycles = hasCycles || hc
		}
		newStruct := t
		if changed {
			newStruct = makeStructTypeQuickly(name, fields, checkKindNoValidate)
		}

		if name != "" {
			bucket, ok := simplifier.seenByName[name]
			if !ok {
				bucket = typeset{}
				simplifier.seenByName[name] = bucket
			}
			bucket.add(newStruct)

			return simplifier.makeCycleType(name), true, true
		}

		return newStruct, changed, hasCycles

	case CycleKind:
		name := string(t.Desc.(CycleDesc))
		return simplifier.makeCycleType(name), true, true

	case UnionKind:
		return simplifier.mergeUnion(t)
	}
	panic("Unknown noms kind " + k.String())
}

func (simplifier *typeSimplifier) mergeUnion(t *Type) (*Type, bool, bool) {
	// For unions we merge all structs with the same name
	elemTypes := t.Desc.(CompoundDesc).ElemTypes
	changed, hasCycles := false, false

	// Remove pointer equal types and flatten unions
	ts := make(typeset, len(elemTypes))
	for _, t := range elemTypes {
		changed = changed || t.TargetKind() == UnionKind || ts.has(t)
		ts.add(t)
	}

	type how struct {
		k NomsKind
		n string
	}

	out := make(typeSlice, 0, len(elemTypes))
	groups := map[how]typeset{}

	for ot := range ts {
		t, c, hc := simplifier.simplify(ot)
		changed = changed || c
		hasCycles = hasCycles || hc

		k := t.TargetKind()
		h := how{k: k}
		switch k {
		case BoolKind, NumberKind, StringKind, BlobKind, ValueKind, TypeKind:
			out = append(out, t)
			continue

		case ListKind, MapKind, RefKind, SetKind:
			break

		case StructKind:
			d.PanicIfFalse(t.Desc.(StructDesc).Name == "") // Only non named struct are kept at this level.
			// No need to set h.n to "" again.

		case CycleKind:
			d.PanicIfFalse(hasCycles) // should have detected this earlier.
			h.n = string(t.Desc.(CycleDesc))

		case UnionKind:
			panic("should not see unions at this level")
		default:
			panic("Unknown noms kind " + k.String())
		}

		g := groups[h]
		if g == nil {
			g = typeset{}
			groups[h] = g
		}
		g.add(t)
	}

	for h, ts := range groups {
		if len(ts) == 1 {
			for t := range ts {
				out = append(out, t)
			}
			continue
		}

		changed = true

		var r *Type
		switch h.k {
		case BoolKind, NumberKind, StringKind, BlobKind, ValueKind, TypeKind, UnionKind:
			panic("Should not be part of groups")

		case ListKind, RefKind, SetKind:
			r = simplifier.mergeCompoundTypesForUnion(h.k, ts)
		case MapKind:
			r = simplifier.mergeMapTypesForUnion(ts)
		case StructKind:
			d.PanicIfFalse(h.n == "") // Only non named struct are kept at this level.
			r = simplifier.mergeStructTypes(h.n, ts)

		case CycleKind:
			d.PanicIfFalse(hasCycles) // should have detected this earlier.
			// All the types in a group have the same name
			r = simplifier.makeCycleType(h.n)

			// TODO: Why wasn't this grouping Cycle<A> and Cycle<A>???

		default:
			panic("Unknown noms kind " + h.k.String())
		}

		out = append(out, r)
	}

	if len(out) == 1 {
		// changed = true because this used to be union type.
		return out[0], true, hasCycles
	}

	if !sort.IsSorted(out) {
		sort.Sort(out)
		changed = true
	}

	if changed || !sort.IsSorted(t.Desc.(CompoundDesc).ElemTypes) {
		return makeCompoundType(UnionKind, out...), true, hasCycles
	}

	return t, false, hasCycles
}

func (simplifier *typeSimplifier) mergeCompoundTypesForUnion(k NomsKind, ts typeset) *Type {
	elemTypes := make(typeset, len(ts))
	for t := range ts {
		d.PanicIfFalse(t.TargetKind() == k)
		elemTypes.add(t.Desc.(CompoundDesc).ElemTypes[0])
	}

	elemType := simplifier.bucketElements(elemTypes)
	return makeCompoundType(k, elemType)
}

func (simplifier *typeSimplifier) mergeMapTypesForUnion(ts typeset) *Type {
	keyTypes := make(typeset, len(ts))
	valTypes := make(typeset, len(ts))
	for t := range ts {
		d.PanicIfFalse(t.TargetKind() == MapKind)
		elemTypes := t.Desc.(CompoundDesc).ElemTypes
		keyTypes.add(elemTypes[0])
		valTypes.add(elemTypes[1])
	}

	kt := simplifier.bucketElements(keyTypes)
	vt := simplifier.bucketElements(valTypes)

	return makeCompoundType(MapKind, kt, vt)
}

func (simplifier *typeSimplifier) mergeStructTypes(name string, ts typeset) *Type {
	fieldset := make([]structTypeFields, len(ts))
	i := 0
	for t := range ts {
		desc := t.Desc.(StructDesc)
		d.PanicIfFalse(desc.Name == name)
		fieldset[i] = desc.fields
		i++
	}
	fields := simplifier.simplifyStructFields(fieldset)
	return newType(StructDesc{name, fields})
}

// Drops cycles and replaces them with pointers to parent structs
func (simplifier *typeSimplifier) resolveStructCycles(t *Type, env map[string]*Type, seen map[string]struct{}) *Type {
	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for i, et := range desc.ElemTypes {
			desc.ElemTypes[i] = simplifier.resolveStructCycles(et, env, seen)
		}

	case StructDesc:
		name := desc.Name
		if name != "" {
			if _, ok := seen[name]; ok {
				return t
			}
			seen[name] = struct{}{}
		}

		for i, f := range desc.fields {
			desc.fields[i].Type = simplifier.resolveStructCycles(f.Type, env, seen)
		}

	case CycleDesc:
		name := string(desc)
		if nt, ok := env[name]; ok {
			return simplifier.resolveStructCycles(nt, env, seen)
		}
	}

	return t
}
