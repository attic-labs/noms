// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

func assertSubtype(t *Type, v Value) {
	if !isSubtype(t, v.Type(), nil) {
		d.Chk.Fail("Invalid type", "%s is not a subtype of %s", v.Type().Describe(), t.Describe())
	}
}

func isSubtype(requiredType, concreteType *Type, parentStructTypes []*Type) bool {
	if requiredType.Equals(concreteType) {
		return true
	}

	if requiredType.Kind() == UnionKind {
		// If we're comparing two unions all component types must be compatible
		if concreteType.Kind() == UnionKind {
			for _, t := range concreteType.Desc.(CompoundDesc).ElemTypes {
				if t.Kind() == UnionKind {
					d.Chk.Fail("Nested UnionKind")
				}
				if !isSubtype(requiredType, t, parentStructTypes) {
					return false
				}
			}
			return true
		}

		for _, t := range requiredType.Desc.(CompoundDesc).ElemTypes {
			if isSubtype(t, concreteType, parentStructTypes) {
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
			if !compoundSubtype(t, concreteElemTypes[i], parentStructTypes) {
				return false
			}
		}
		return true
	}

	if requiredType.Kind() == StructKind {
		_, found := indexOfType(requiredType, parentStructTypes)
		if found {
			return true
		}
		requiredDesc := requiredType.Desc.(StructDesc)
		concreteDesc := concreteType.Desc.(StructDesc)
		if requiredDesc.Name != "" && requiredDesc.Name != concreteDesc.Name {
			return false
		}
		j := 0
		for _, field := range requiredDesc.fields {
			for ; j < concreteDesc.Len() && concreteDesc.fields[j].name != field.name; j++ {
			}
			if j == concreteDesc.Len() {
				return false
			}

			f := concreteDesc.fields[j]
			if !isSubtype(field.t, f.t, append(parentStructTypes, requiredType)) {
				return false
			}
		}
		return true

	}

	panic("unreachable")
}

func compoundSubtype(requiredType, concreteType *Type, parentStructTypes []*Type) bool {
	// In a compound type it is OK to have an empty union.
	if concreteType.Kind() == UnionKind && len(concreteType.Desc.(CompoundDesc).ElemTypes) == 0 {
		return true
	}
	return isSubtype(requiredType, concreteType, parentStructTypes)
}
