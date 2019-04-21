// Copyright 2019 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/types"
)

// ToJSON encodes a Noms value as JSON.
func ToJSON(v types.Value, w io.Writer, opts ToOptions) error {
	// TODO: This is a quick hack that is expedient. We should marshal directly to the writer without
	// allocating a bunch of Go values.
	p, err := toPile(v, opts)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", opts.Indent)
	return enc.Encode(p)
}

// ToOptions controls how ToJSON works.
type ToOptions struct {
	// Enable support for encoding Noms Lists. Lists are encoded as JSON arrays.
	Lists bool
	// Enable support for encoding Noms Maps. Maps are encoded as JSON objects.
	Maps bool
	// Enable support for encoding Noms Sets. Sets are encoded as JSON arrays.
	Sets bool
	// Enable support for encoding Noms Structs. Structs are encoded as JSON objects.
	Structs bool
	// String to use for indent when pretty-printing
	Indent string
}

func toPile(v types.Value, opts ToOptions) (ret interface{}, err error) {
	switch v := v.(type) {
	case types.Bool:
		return bool(v), nil
	case types.Number:
		return float64(v), nil
	case types.String:
		return string(v), nil
	case types.Struct:
		if !opts.Structs {
			return nil, errors.New("Struct marshaling not enabled")
		}
		r := map[string]interface{}{}
		if v.Name() != "" {
			return nil, errors.New("Named struct marshaling not supported")
		}
		v.IterFields(func(k string, cv types.Value) (stop bool) {
			var cp interface{}
			cp, err = toPile(cv, opts)
			if err != nil {
				return true
			}
			r[k] = cp
			return false
		})
		return r, err
	case types.Map:
		if !opts.Maps {
			return nil, errors.New("Map marshaling not enabled")
		}
		r := make(map[string]interface{}, v.Len())
		v.Iter(func(k, cv types.Value) (stop bool) {
			sk, ok := k.(types.String)
			if !ok {
				err = fmt.Errorf("Map key kind %s not supported", types.KindToString[k.Kind()])
				return true
			}
			var cp interface{}
			cp, err = toPile(cv, opts)
			if err != nil {
				return true
			}
			r[string(sk)] = cp
			return false
		})
		return r, err
	case types.List:
		if !opts.Lists {
			return nil, errors.New("List marshaling not enabled")
		}
		r := make([]interface{}, v.Len())
		v.Iter(func(cv types.Value, i uint64) (stop bool) {
			var cp interface{}
			cp, err = toPile(cv, opts)
			if err != nil {
				return true
			}
			r[i] = cp
			return false
		})
		return r, err
	case types.Set:
		if !opts.Sets {
			return nil, errors.New("Set marshaling not enabled")
		}
		r := make([]interface{}, 0, v.Len())
		v.Iter(func(cv types.Value) (stop bool) {
			var cp interface{}
			cp, err = toPile(cv, opts)
			if err != nil {
				return true
			}
			r = append(r, cp)
			return false
		})
		return r, err
	}
	return nil, fmt.Errorf("Unsupported kind: %s", types.KindToString[v.Kind()])
}
