// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package marshal implements encoding and decoding of Noms values. The mapping
// between Noms objects and Go values is described  in the documentation for the
// Marshal and Unmarshal functions.
package marshal

import (
	"reflect"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// MarshalType ...
func MarshalType(v interface{}) (nt *types.Type, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case *UnsupportedTypeError, *InvalidTagError:
				err = r.(error)
			case *marshalNomsError:
				err = r.err
			default:
				panic(r)
			}
		}
	}()
	rv := reflect.ValueOf(v)
	nt = nomsType(rv.Type(), nil, nomsTags{}, nomsTypeOptions{
		IgnoreOmitempty: true,
	})

	if nt == nil {
		err = &UnsupportedTypeError{Type: rv.Type()}
	}

	return
}

// MustMarshalType ...
func MustMarshalType(v interface{}) *types.Type {
	t, err := MarshalType(v)
	d.Chk.NoError(err)
	return t
}

var typeOfTypesType = reflect.TypeOf((*types.Type)(nil))

type nomsTypeOptions struct {
	IgnoreOmitempty bool
}

func nomsType(t reflect.Type, parentStructTypes []reflect.Type, tags nomsTags, options nomsTypeOptions) *types.Type {
	if t.Implements(marshalerInterface) {
		// There is no way to determine the noms type now, it can be different each
		// time MarshalNoms is called. This is handled further up the stack.
		return nil
	}

	if t.Implements(nomsValueInterface) {
		if t == typeOfTypesType {
			return types.TypeType
		}

		// Use Name because List and Blob are convertible to each other on Go.
		switch t.Name() {
		case "Blob":
			return types.BlobType
		case "Bool":
			return types.BoolType
		case "Number":
			return types.NumberType
		case "String":
			return types.StringType
		}
		// The rest of the Noms types need the value to get the exact type.
		return nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return types.BoolType
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return types.NumberType
	case reflect.String:
		return types.StringType
	case reflect.Struct:
		return structNomsType(t, parentStructTypes, options)
	case reflect.Array, reflect.Slice:
		elemType := nomsType(t.Elem(), parentStructTypes, nomsTags{}, options)
		if elemType != nil {
			return types.MakeListType(elemType)
		}
	case reflect.Map:
		keyType := nomsType(t.Key(), parentStructTypes, nomsTags{}, options)
		if keyType == nil {
			break
		}

		if shouldMapEncodeAsSet(t, tags) {
			return types.MakeSetType(keyType)
		}

		valueType := nomsType(t.Elem(), parentStructTypes, nomsTags{}, options)
		if valueType != nil {
			return types.MakeMapType(keyType, valueType)
		}
	}

	// This will be reported as an error at a different layer.
	return nil
}

// structNomsType returns the Noms types.Type if it can be determined from the
// reflect.Type. In some cases we cannot determine the type by only looking at
// the type but we also need to look at the value. In this cases this returns
// nil and we have to wait until we have a value to be able to determine the
// type.
func structNomsType(t reflect.Type, parentStructTypes []reflect.Type, options nomsTypeOptions) *types.Type {
	for i, pst := range parentStructTypes {
		if pst == t {
			return types.MakeCycleType(uint32(i))
		}
	}

	parentStructTypes = append(parentStructTypes, t)

	_, structType, _ := typeFields(t, parentStructTypes, options)
	return structType
}
