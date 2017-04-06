// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package marshal implements encoding and decoding of Noms values. The mapping
// between Noms objects and Go values is described  in the documentation for the
// Marshal and Unmarshal functions.
package marshal

import (
	"fmt"
	"reflect"

	"github.com/attic-labs/noms/go/types"
)

// MarshalType computes a Noms type from a Go type
//
// The rules for MarshalType is the same as for Marshal, except for omitempty
// is ignored since that cannot be determined statically.
//
// If a Go struct contains a noms tag with original the field is skipped since
// the Noms type depends on the original Noms value which is not available.
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
	nt = MustMarshalType(v)
	return
}

// MustMarshalType computes a Noms type from a Go type or panics if there is an
// error.
func MustMarshalType(v interface{}) (nt *types.Type) {
	rv := reflect.ValueOf(v)
	nt = encodeType(rv.Type(), map[string]reflect.Type{}, nomsTags{}, encodeTypeOptions{
		IgnoreOmitEmpty: true,
		ReportErrors:    true,
	})

	if nt == nil {
		panic(&UnsupportedTypeError{Type: rv.Type()})
	}

	return
}

// TypeMarshaler is an interface types can implement to provide their own
// encoding of type.
type TypeMarshaler interface {
	// MarshalNomsType returns the Noms Type encoding of a type, or an error.
	// nil is not a valid return val - if both val and err are nil, MarshalType
	// will panic.
	MarshalNomsType() (t *types.Type, err error)
}

var typeOfTypesType = reflect.TypeOf((*types.Type)(nil))
var typeMarshalerInterface = reflect.TypeOf((*TypeMarshaler)(nil)).Elem()

type encodeTypeOptions struct {
	IgnoreOmitEmpty, ReportErrors bool
}

func encodeType(t reflect.Type, seenStructs map[string]reflect.Type, tags nomsTags, options encodeTypeOptions) *types.Type {
	if t.Implements(typeMarshalerInterface) {
		v := reflect.Zero(t)
		typ, err := v.Interface().(TypeMarshaler).MarshalNomsType()
		if err != nil {
			panic(&marshalNomsError{err})
		}
		if typ == nil {
			panic(fmt.Errorf("nil result from %s.MarshalNomsType", t))
		}
		return typ
	}

	if t.Implements(marshalerInterface) {
		// There is no way to determine the noms type now. For Marshal it can be
		// different each time MarshalNoms is called and is handled further up the
		// stack.
		if options.ReportErrors {
			err := fmt.Errorf("Cannot marshal type which implements %s, perhaps implement %s for %s", marshalerInterface, typeMarshalerInterface, t)
			panic(&marshalNomsError{err})
		}

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

		if options.ReportErrors {
			err := fmt.Errorf("Cannot marshal type %s, it requires type parameters", t)
			panic(&marshalNomsError{err})
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
		return structEncodeType(t, seenStructs, options)
	case reflect.Array, reflect.Slice:
		elemType := encodeType(t.Elem(), seenStructs, nomsTags{}, options)
		if elemType != nil {
			return types.MakeListType(elemType)
		}
	case reflect.Map:
		keyType := encodeType(t.Key(), seenStructs, nomsTags{}, options)
		if keyType == nil {
			break
		}

		if shouldMapEncodeAsSet(t, tags) {
			return types.MakeSetType(keyType)
		}

		valueType := encodeType(t.Elem(), seenStructs, nomsTags{}, options)
		if valueType != nil {
			return types.MakeMapType(keyType, valueType)
		}
	}

	// This will be reported as an error at a different layer.
	return nil
}

// structEncodeType returns the Noms types.Type if it can be determined from the
// reflect.Type. In some cases we cannot determine the type by only looking at
// the type but we also need to look at the value. In these cases this returns
// nil and we have to wait until we have a value to be able to determine the
// type.
func structEncodeType(t reflect.Type, seenStructs map[string]reflect.Type, options encodeTypeOptions) *types.Type {
	name := t.Name()
	if name != "" {
		if _, ok := seenStructs[name]; ok {
			return types.MakeCycleType(name)
		}
		seenStructs[name] = t
	}

	_, structType, _ := typeFields(t, seenStructs, options)
	return structType
}
