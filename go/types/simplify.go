package types

import (
	"sort"

	"github.com/attic-labs/noms/go/d"
)

// makeSimplifiedType returns a type that is a supertype of all the input types but is much
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
//     {struct{foo:number,bar:string}, struct{bar:blob, baz:bool}} ->
//       struct{foo?:number,bar:string|blob,baz?:bool}
//
// Anytime any of the above cases generates a union as output, the same process
// is applied to that union recursively.
func makeSimplifiedType(intersectStructs bool, in *Type) *Type {
	return in
	// seen := map[*Type]*Type{}
	// pending := map[string]*unsimplifiedStruct{}
	//
	// out, _ := removeAndCollectStructFields(in, seen, pending)
	//
	// result := makeSimplifiedTypeImpl(out, intersectStructs)
	// for _, rec := range pending {
	// 	desc := rec.t.Desc.(StructDesc)
	//
	// 	desc.fields = simplifyStructFields(rec.fieldSets, intersectStructs)
	// 	rec.t.Desc = desc
	// }
	// return result
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

func newTypeset(t ...*Type) typeset {
	ts := make(typeset, len(t))
	for _, t := range t {
		ts.add(t)
	}
	return ts
}

// makeSimplifiedTypeImpl is an implementation detail.
// Warning: Do not call this directly. It assumes its input has been de-cycled using
// ToUnresolvedType() and will infinitely recurse on cyclic types otherwise.
func makeSimplifiedTypeImpl(in *Type, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) *Type {
	switch in.TargetKind() {
	case BoolKind, NumberKind, StringKind, BlobKind, ValueKind, TypeKind, CycleKind:
		return in
	case ListKind, MapKind, RefKind, SetKind:
		elemTypes := make(typeSlice, len(in.Desc.(CompoundDesc).ElemTypes))
		for i, t := range in.Desc.(CompoundDesc).ElemTypes {
			elemTypes[i] = makeSimplifiedTypeImpl(t, seenStructs, seenByName, intersectStructs)
		}
		return makeCompoundType(in.TargetKind(), elemTypes...)
	case StructKind:
		// Structs have been replaced by "placeholders" in removeAndCollectStructFields.
		return in
	case UnionKind:
		elemTypes := make(typeSlice, len(in.Desc.(CompoundDesc).ElemTypes))
		ts := make(typeset, len(elemTypes))
		for _, t := range in.Desc.(CompoundDesc).ElemTypes {
			t = makeSimplifiedTypeImpl(t, seenStructs, seenByName, intersectStructs)
			ts.add(t)
		}

		return bucketElements(ts, seenStructs, seenByName, intersectStructs)
	}
	panic("Unknown noms kind")
}

func bucketElements(in typeset, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) *Type {
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
			r = mergeCompoundTypesForUnion(h.k, ts, seenStructs, seenByName, intersectStructs)
		case MapKind:
			r = mergeMapTypesForUnion(ts, seenStructs, seenByName, intersectStructs)
		case StructKind:
			r = mergeStructTypes(h.n, ts, seenStructs, seenByName, intersectStructs)
		}
		out = append(out, r)
	}

	if len(out) == 1 {
		return out[0]
	}

	sort.Sort(out)

	return makeCompoundType(UnionKind, out...)
}

// type unsimplifiedStruct struct {
// 	t         *Type
// 	fieldSets []structTypeFields
// }

// func removeAndCollectStructFields(t *Type, seen map[*Type]*Type, pendingStructs map[string]*unsimplifiedStruct) (*Type, bool) {
// 	switch t.TargetKind() {
// 	case BoolKind, NumberKind, StringKind, BlobKind, ValueKind, TypeKind:
// 		return t, false
// 	case ListKind, MapKind, RefKind, SetKind, UnionKind:
// 		elemTypes := t.Desc.(CompoundDesc).ElemTypes
// 		changed := false
// 		newElemTypes := make(typeSlice, len(elemTypes))
// 		for i, et := range elemTypes {
// 			et2, c := removeAndCollectStructFields(et, seen, pendingStructs)
// 			newElemTypes[i] = et2
// 			changed = changed || c
// 		}
// 		if !changed {
// 			return t, false
// 		}
//
// 		return makeCompoundType(t.TargetKind(), newElemTypes...), true
//
// 	case StructKind:
// 		newStruct, found := seen[t]
// 		if found {
// 			return newStruct, true
// 		}
//
// 		desc := t.Desc.(StructDesc)
// 		name := desc.Name
// 		var pending *unsimplifiedStruct
// 		if name != "" {
// 			var ok bool
// 			pending, ok = pendingStructs[name]
// 			if ok {
// 				newStruct = pending.t
// 			} else {
// 				newStruct = newType(StructDesc{Name: name})
// 				pending = &unsimplifiedStruct{newStruct, []structTypeFields{}}
// 				pendingStructs[name] = pending
// 			}
//
// 		} else {
// 			newStruct = newType(StructDesc{Name: name})
// 		}
// 		seen[t] = newStruct
//
// 		newFields := make(structTypeFields, len(desc.fields))
// 		changed := false
// 		for i, f := range desc.fields {
// 			nt, c := removeAndCollectStructFields(f.Type, seen, pendingStructs)
// 			newFields[i] = StructField{Name: f.Name, Type: nt, Optional: f.Optional}
// 			changed = changed || c
// 		}
//
// 		if !changed {
// 			newFields = desc.fields
// 		}
//
// 		if name != "" {
// 			pending.fieldSets = append(pending.fieldSets, newFields)
// 		} else {
// 			fs := make(structTypeFields, len(newFields))
// 			for i, f := range newFields {
// 				// nt := makeSimplifiedTypeImpl(f.Type, inter)
// 				fs[i] = StructField{Name: f.Name, Type: f.Type, Optional: f.Optional}
// 			}
// 			newStruct.Desc = StructDesc{"", fs}
// 		}
// 		return newStruct, true
//
// 	case CycleKind:
// 		return t, false
// 	}
//
// 	panic("unreachable") // no more noms kinds
// }

func simplifyStructFields(in []structTypeFields, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) structTypeFields {
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
		nt, _, _ := simplifyTypeImpl2(makeCompoundType(UnionKind, fti.ts...), seenStructs, seenByName, intersectStructs)
		fields = append(fields, StructField{
			Name: name,
			// Type:     makeSimplifiedTypeImpl(makeCompoundType(UnionKind, fti.ts...), intersectStructs),
			Type:     nt,
			Optional: !(intersectStructs && fti.anyNonOptional) && fti.count < count,
		})
	}

	sort.Sort(fields)

	return fields
}

func simplifyType2(t *Type, intersectStructs bool) *Type {
	seenStructs := typeset{}
	seenByName := map[string]typeset{}
	rv, changed, hasCycles := simplifyTypeImpl2(t, seenStructs, seenByName, intersectStructs)
	if !changed {
		return t
	}
	// fmt.Println("----------------------------------------")
	// fmt.Println("changed", changed)
	// fmt.Println("hasCycles", hasCycles)
	// fmt.Println("Describe", rv.Describe())
	if len(seenByName) > 0 {
		d.PanicIfFalse(len(seenStructs) > 0)
		d.PanicIfFalse(changed)
		d.PanicIfFalse(hasCycles)

		env := make(map[string]*Type, len(seenByName))
		for name, ts := range seenByName {
			// fmt.Println("name", name, len(ts))
			newStruct := mergeStructTypes(name, ts, seenStructs, seenByName, intersectStructs)
			// fmt.Println("merged", name, newStruct.Describe())
			env[name] = newStruct
			// fmt.Println("merged", name, newStruct.Describe())
		}

		rv2 := resolveStructCycles2(rv, env, map[string]struct{}{})
		// fmt.Println("resolved", rv.Describe(), "to", rv2.Describe())
		return rv2
	}

	return rv
}

func simplifyTypeImpl2(t *Type, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) (*Type, bool, bool) {
	if seenStructs.has(t) {
		// Already handled.
		name := t.Desc.(StructDesc).Name
		return MakeCycleType(name), true, true
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
			nt, c, hc := simplifyTypeImpl2(et, seenStructs, seenByName, intersectStructs)
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
			seenStructs.add(t)
		}

		changed, hasCycles := false, false
		fields := make(structTypeFields, len(desc.fields))
		for i, f := range desc.fields {
			ft, c, hc := simplifyTypeImpl2(f.Type, seenStructs, seenByName, intersectStructs)
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
			bucket, ok := seenByName[name]
			if !ok {
				bucket = typeset{}
				seenByName[name] = bucket
			}
			bucket.add(newStruct)

			return MakeCycleType(name), true, true
		}

		return newStruct, changed, hasCycles

	case CycleKind:
		return t, false, true

	case UnionKind:
		return mergeUnion2(t, seenStructs, seenByName, intersectStructs)
	}
	panic("Unknown noms kind " + k.String())
}

func mergeUnion2(t *Type, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) (*Type, bool, bool) {
	// For unions we merge all structs with the same name
	elemTypes := t.Desc.(CompoundDesc).ElemTypes
	changed, hasCycles := false, false

	// Remove pointer equals types and flatten unions

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
		t, c, hc := simplifyTypeImpl2(ot, seenStructs, seenByName, intersectStructs)
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
			r = mergeCompoundTypesForUnion(h.k, ts, seenStructs, seenByName, intersectStructs)
		case MapKind:
			r = mergeMapTypesForUnion(ts, seenStructs, seenByName, intersectStructs)
		case StructKind:
			d.PanicIfFalse(h.n == "") // Only non named struct are kept at this level.
			r = mergeStructTypes(h.n, ts, seenStructs, seenByName, intersectStructs)

		case CycleKind:
			d.PanicIfFalse(hasCycles) // should have detected this earlier.
			// All the types in a group have the same name
			r = MakeCycleType(h.n)

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

func mergeCompoundTypesForUnion(k NomsKind, ts typeset, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) *Type {
	elemTypes := make(typeset, len(ts))
	for t := range ts {
		d.PanicIfFalse(t.TargetKind() == k)
		elemTypes.add(t.Desc.(CompoundDesc).ElemTypes[0])
	}

	elemType := bucketElements(elemTypes, seenStructs, seenByName, intersectStructs)
	return makeCompoundType(k, elemType)
}

func mergeMapTypesForUnion(ts typeset, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) *Type {
	keyTypes := make(typeset, len(ts))
	valTypes := make(typeset, len(ts))
	for t := range ts {
		d.PanicIfFalse(t.TargetKind() == MapKind)
		elemTypes := t.Desc.(CompoundDesc).ElemTypes
		keyTypes.add(elemTypes[0])
		valTypes.add(elemTypes[1])
	}

	kt := bucketElements(keyTypes, seenStructs, seenByName, intersectStructs)
	vt := bucketElements(valTypes, seenStructs, seenByName, intersectStructs)

	return makeCompoundType(MapKind, kt, vt)
}

func mergeStructTypes(name string, ts typeset, seenStructs typeset, seenByName map[string]typeset, intersectStructs bool) *Type {
	fieldset := make([]structTypeFields, len(ts))
	i := 0
	for t := range ts {
		desc := t.Desc.(StructDesc)
		d.PanicIfFalse(desc.Name == name)
		fieldset[i] = desc.fields
		i++
	}
	fields := simplifyStructFields(fieldset, seenStructs, seenByName, intersectStructs)
	return newType(StructDesc{name, fields})
}

// Drops cycles and replaces them with pointers to parent structs
func resolveStructCycles2(t *Type, env map[string]*Type, seen map[string]struct{}) *Type {
	switch desc := t.Desc.(type) {
	case CompoundDesc:
		for i, et := range desc.ElemTypes {
			desc.ElemTypes[i] = resolveStructCycles2(et, env, seen)
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
			desc.fields[i].Type = resolveStructCycles2(f.Type, env, seen)
		}

	case CycleDesc:
		name := string(desc)
		if nt, ok := env[name]; ok {
			return resolveStructCycles2(nt, env, seen)
		}
	}

	return t
}

// // Drops cycles and replaces them with pointers to parent structs
// func resolveStructCycles2(t *Type, env map[string]*Type, seen typeset) *Type {
// 	switch desc := t.Desc.(type) {
// 	case CompoundDesc:
// 		elemTypes := make(typeSlice, len(desc.ElemTypes))
// 		for i, et := range desc.ElemTypes {
// 			elemTypes[i] = resolveStructCycles2(et, env, seen)
// 		}
// 		return makeCompoundType(t.TargetKind(), elemTypes...)
//
// 	case StructDesc:
// 		fields := make(structTypeFields, len(desc.fields))
// 		for i, f := range desc.fields {
// 			fields[i] = StructField{f.Name, resolveStructCycles2(f.Type, env, seen), f.Optional}
// 		}
// 		return makeStructTypeQuickly(desc.Name, fields, checkKindNoValidate)
//
// 	case CycleDesc:
// 		name := string(desc)
// 		if nt, ok := env[name]; ok {
// 			if seen.has(nt) {
// 				return nt
// 			}
// 			nt = resolveStructCycles2(nt, env, seen)
// 			seen.add(nt)
// 			return nt
// 		}
// 	}
//
// 	return t
// }
