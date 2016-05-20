package types

import "github.com/attic-labs/noms/d"

func assertSubtype(t *Type, v Value) {
	if !isSubtype(t, v.Type()) {
		d.Chk.Fail("Invalid type", "Expected: %s, found: %s", t.Describe(), v.Type().Describe())
	}
}

func isSubtype(requiredType, concreteType *Type) bool {
	if requiredType.Equals(concreteType) {
		return true
	}

	if requiredType.Kind() == UnionKind {
		for _, t := range requiredType.Desc.(CompoundDesc).ElemTypes {
			if isSubtype(t, concreteType) {
				return true
			}
		}
		return false
	}

	if requiredType.Kind() != concreteType.Kind() {
		return requiredType.Kind() == ValueKind
	}

	if desc, ok := requiredType.Desc.(CompoundDesc); ok {
		concreteElemTypes := concreteType.Desc.(CompoundDesc).ElemTypes
		for i, t := range desc.ElemTypes {
			if !compoundSubtype(t, concreteElemTypes[i]) {
				return false
			}
		}
		return true
	}

	if requiredType.Kind() == StructKind {
		requiredDesc := requiredType.Desc.(StructDesc)
		concreteDesc := concreteType.Desc.(StructDesc)
		if requiredDesc.Name != "" && requiredDesc.Name != concreteDesc.Name {
			return false
		}
		type Entry struct {
			name string
			t    *Type
		}
		entries := make([]Entry, 0, len(requiredDesc.Fields))
		requiredDesc.IterFields(func(name string, t *Type) {
			entries = append(entries, Entry{name, t})
		})
		for _, entry := range entries {
			at, ok := concreteDesc.Fields[entry.name]
			if !ok || !isSubtype(entry.t, at) {
				return false
			}
		}
		return true
	}

	panic("unreachable")
}

func compoundSubtype(requiredType, concreteType *Type) bool {
	// In a compound type it is OK to have an empty union.
	if concreteType.Kind() == UnionKind && len(concreteType.Desc.(CompoundDesc).ElemTypes) == 0 {
		return true
	}
	return isSubtype(requiredType, concreteType)
}
