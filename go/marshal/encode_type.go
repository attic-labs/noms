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
