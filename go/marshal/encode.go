// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package marshal implements encoding and decoding of Noms values. The mapping
// between Noms objects and Go values is described  in the documentation for the
// Marshal and Unmarshal functions.
package marshal

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/attic-labs/noms/go/types"
)

// Marshal converts a Go value to a Noms value.
//
// Marshal traverses the value v recursively. Marshal uses the following
// type-dependent encodings:
//
// Boolean values are encoded as Noms types.Bool.
//
// Floating point and integer values are encoded as Noms types.Number. At the
// moment this might lead to some loss in precision because types.Number
// currently takes a float64.
//
// String values are encoded as Noms types.String.
//
// Slices and arrays are encoded as Noms types.List by default. If a
// field is tagged with `noms:"set", it will be encoded as Noms types.Set
// instead.
//
// Maps are encoded as Noms types.Map, or a types.Set if the value type is
// struct{} and the field is tagged with `noms:"set"`.
//
// Struct values are encoded as Noms structs (types.Struct). Each exported Go
// struct field becomes a member of the Noms struct unless
//   - The field's tag is "-"
//   - The field is empty and its tag specifies the "omitempty" option.
//   - The field has the "original" tag, in which case the field is used as an
//     initial value onto which the fields of the Go type are added. When
//     combined with the corresponding support for "original" in Unmarshal(),
//     this allows one to find and modify any values of a known subtype.
//
// Additionally, user-defined types can implement the Marshaler interface to
// provide a custom encoding.
//
// The empty values are false, 0, any nil pointer or interface value, and any
// array, slice, map, or string of length zero.
//
// The Noms struct default field name is the Go struct field name where the
// first character is lower cased, but can be specified in the Go struct field's
// tag value. The "noms" key in the Go struct field's tag value is the field
// name. Examples:
//
//   // Field is ignored.
//   Field int `noms:"-"`
//
//   // Field appears in a Noms struct as field "myName".
//   MyName int
//
//   // Field appears in a Noms struct as key "myName".
//   Field int `noms:"myName"`
//
//   // Field appears in a Noms struct as key "myName" and the field is
//   //  omitted from the object if its value is empty, as defined above.
//   Field int `noms:"myName,omitempty"
//
//   // Field appears in a Noms struct as key "field" and the field is
//   //  omitted from the object if its value is empty, as defined above.
//   Field int `noms:",omitempty"
//
// The name of the Noms struct is the name of the Go struct where the first
// character is changed to upper case.
//
// Anonymous struct fields are currently not supported.
//
// Embedded structs are currently not supported (which is the same as anonymous
// struct fields).
//
// Noms values (values implementing types.Value) are copied over without any
// change.
//
// When marshalling interface{} the dynamic type is used.
//
// Go pointers, complex, function are not supported. Attempting to encode such a
// value causes Marshal to return an UnsupportedTypeError.
//
func Marshal(v interface{}) (nomsValue types.Value, err error) {
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
	nomsValue = MustMarshal(v)
	return
}

// MustMarshal marshals a Go value to a Noms value using the same rules as
// Marshal(). Panics on failure.
func MustMarshal(v interface{}) types.Value {
	rv := reflect.ValueOf(v)
	encoder := typeEncoder(rv.Type(), nil, nomsTags{})
	return encoder(rv)
}

// Marshaler is an interface types can implement to provide their own encoding.
type Marshaler interface {
	// MarshalNoms returns the Noms Value encoding of a type, or an error.
	// nil is not a valid return val - if both val and err are nil, Marshal will
	// panic.
	MarshalNoms() (val types.Value, err error)
}

// UnsupportedTypeError is returned by encode when attempting to encode a type
// that isn't supported.
type UnsupportedTypeError struct {
	Type    reflect.Type
	Message string
}

func (e *UnsupportedTypeError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "Type is not supported"
	}
	return msg + ", type: " + e.Type.String()
}

// InvalidTagError is returned by encode and decode when the struct field tag is
// invalid. For example if the field name is not a valid Noms struct field name.
type InvalidTagError struct {
	message string
}

func (e *InvalidTagError) Error() string {
	return e.message
}

// marshalNomsError wraps errors from Marshaler.MarshalNoms. These should be
// unwrapped and never leak to the caller of Marshal.
type marshalNomsError struct {
	err error
}

func (e *marshalNomsError) Error() string {
	return e.err.Error()
}

type nomsTags struct {
	name      string
	omitEmpty bool
	original  bool
	set       bool
	skip      bool
}

var nomsValueInterface = reflect.TypeOf((*types.Value)(nil)).Elem()
var emptyInterface = reflect.TypeOf((*interface{})(nil)).Elem()
var marshalerInterface = reflect.TypeOf((*Marshaler)(nil)).Elem()

type encoderFunc func(v reflect.Value) types.Value

func boolEncoder(v reflect.Value) types.Value {
	return types.Bool(v.Bool())
}

func float64Encoder(v reflect.Value) types.Value {
	return types.Number(v.Float())
}

func intEncoder(v reflect.Value) types.Value {
	return types.Number(float64(v.Int()))
}

func uintEncoder(v reflect.Value) types.Value {
	return types.Number(float64(v.Uint()))
}

func stringEncoder(v reflect.Value) types.Value {
	return types.String(v.String())
}

func nomsValueEncoder(v reflect.Value) types.Value {
	return v.Interface().(types.Value)
}

func marshalerEncoder(t reflect.Type) encoderFunc {
	return func(v reflect.Value) types.Value {
		val, err := v.Interface().(Marshaler).MarshalNoms()
		if err != nil {
			panic(&marshalNomsError{err})
		}
		if val == nil {
			panic(fmt.Errorf("nil result from %s.MarshalNoms", t.String()))
		}
		return val
	}
}

func typeEncoder(t reflect.Type, parentStructTypes []reflect.Type, tags nomsTags) encoderFunc {
	if t.Implements(marshalerInterface) {
		return marshalerEncoder(t)
	}

	switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Float64, reflect.Float32:
		return float64Encoder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintEncoder
	case reflect.String:
		return stringEncoder
	case reflect.Struct:
		return structEncoder(t, parentStructTypes)
	case reflect.Slice, reflect.Array:
		if shouldEncodeAsSet(t, tags) {
			return setFromListEncoder(t, parentStructTypes)
		}
		return listEncoder(t, parentStructTypes)
	case reflect.Map:
		if shouldEncodeAsSet(t, tags) {
			return setEncoder(t, parentStructTypes)
		}
		return mapEncoder(t, parentStructTypes)
	case reflect.Interface:
		return func(v reflect.Value) types.Value {
			// Get the dynamic type.
			v2 := reflect.ValueOf(v.Interface())
			return typeEncoder(v2.Type(), parentStructTypes, tags)(v2)
		}
	case reflect.Ptr:
		// Allow implementations of types.Value (like *types.Type)
		if t.Implements(nomsValueInterface) {
			return nomsValueEncoder
		}
		fallthrough
	default:
		panic(&UnsupportedTypeError{Type: t})
	}
}

func structEncoder(t reflect.Type, parentStructTypes []reflect.Type) encoderFunc {
	if t.Implements(nomsValueInterface) {
		return nomsValueEncoder
	}

	e := encoderCache.get(t)
	if e != nil {
		return e
	}

	parentStructTypes = append(parentStructTypes, t)
	fields, structType, originalFieldIndex := typeFields(t, parentStructTypes, encodeTypeOptions{})
	if structType != nil {
		e = func(v reflect.Value) types.Value {
			values := make([]types.Value, len(fields))
			for i, f := range fields {
				values[i] = f.encoder(v.Field(f.index))
			}
			return types.NewStructWithType(structType, values)
		}
	} else if originalFieldIndex == nil {
		// Slower path: cannot precompute the Noms type since there are Noms collections,
		// but at least there are a set number of fields.
		name := strings.Title(t.Name())
		e = func(v reflect.Value) types.Value {
			data := make(types.StructData, len(fields))
			for _, f := range fields {
				fv := v.Field(f.index)
				if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
					continue
				}
				data[f.name] = f.encoder(fv)
			}
			return types.NewStruct(name, data)
		}
	} else {
		// Slowest path - we are extending some other struct. We need to start with the
		// type of that struct and extend.
		e = func(v reflect.Value) types.Value {
			fv := v.FieldByIndex(originalFieldIndex)
			ret := fv.Interface().(types.Struct)
			if ret.IsZeroValue() {
				ret = types.NewStruct(t.Name(), nil)
			}
			for _, f := range fields {
				fv := v.Field(f.index)
				if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
					continue
				}
				ret = ret.Set(f.name, f.encoder(fv))
			}
			return ret
		}
	}

	encoderCache.set(t, e)
	return e
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		z := reflect.Zero(v.Type())
		return z.Interface() == v.Interface()
	case reflect.Interface:
		return v.IsNil()
	}
	return false
}

type field struct {
	name      string
	encoder   encoderFunc
	index     int
	nomsType  *types.Type
	omitEmpty bool
}

type fieldSlice []field

func (fs fieldSlice) Len() int           { return len(fs) }
func (fs fieldSlice) Swap(i, j int)      { fs[i], fs[j] = fs[j], fs[i] }
func (fs fieldSlice) Less(i, j int) bool { return fs[i].name < fs[j].name }

type encoderCacheT struct {
	sync.RWMutex
	m map[reflect.Type]encoderFunc
}

var encoderCache = &encoderCacheT{}

// Separate Set encoder cache because the same type with and without the
// `noms:",set"` tag encode differently (Set vs Map).
var setEncoderCache = &encoderCacheT{}

func (c *encoderCacheT) get(t reflect.Type) encoderFunc {
	c.RLock()
	defer c.RUnlock()
	return c.m[t]
}

func (c *encoderCacheT) set(t reflect.Type, e encoderFunc) {
	c.Lock()
	defer c.Unlock()
	if c.m == nil {
		c.m = map[reflect.Type]encoderFunc{}
	}
	c.m[t] = e
}

func getTags(f reflect.StructField) (tags nomsTags) {
	reflectTags := f.Tag.Get("noms")
	if reflectTags == "-" {
		tags.skip = true
		return
	}

	tagsSlice := strings.Split(reflectTags, ",")

	// The first tag is always the name, or empty to use the field as the name.
	if len(tagsSlice) == 0 || tagsSlice[0] == "" {
		tags.name = strings.ToLower(f.Name[:1]) + f.Name[1:]
	} else {
		tags.name = tagsSlice[0]
	}

	if !types.IsValidStructFieldName(tags.name) {
		panic(&InvalidTagError{"Invalid struct field name: " + tags.name})
	}

	for i := 1; i < len(tagsSlice); i++ {
		switch tag := tagsSlice[i]; tag {
		case "omitempty":
			tags.omitEmpty = true
		case "original":
			tags.original = true
		case "set":
			tags.set = true
		default:
			panic(&InvalidTagError{"Unrecognized tag: " + tag})
		}
	}
	return
}

func validateField(f reflect.StructField, t reflect.Type) {
	if f.Anonymous {
		panic(&UnsupportedTypeError{t, "Embedded structs are not supported"})
	}
	if unicode.IsLower(rune(f.Name[0])) { // we only allow ascii so this is fine
		panic(&UnsupportedTypeError{t, "Non exported fields are not supported"})
	}
}

func typeFields(t reflect.Type, parentStructTypes []reflect.Type, options encodeTypeOptions) (fields fieldSlice, structType *types.Type, originalFieldIndex []int) {
	canComputeStructType := true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tags := getTags(f)
		if tags.skip {
			continue
		}

		if tags.original {
			originalFieldIndex = f.Index
			continue
		}

		validateField(f, t)
		nt := encodeType(f.Type, parentStructTypes, tags, options)
		if nt == nil {
			canComputeStructType = false
		}

		if tags.omitEmpty && !options.IgnoreOmitEmpty {
			canComputeStructType = false
		}

		fields = append(fields, field{
			name:      tags.name,
			encoder:   typeEncoder(f.Type, parentStructTypes, tags),
			index:     i,
			nomsType:  nt,
			omitEmpty: tags.omitEmpty,
		})

	}
	sort.Sort(fields)
	if canComputeStructType {
		structTypeFields := make([]types.StructField, len(fields))
		for i, fs := range fields {
			structTypeFields[i] = types.StructField{
				Name:     fs.name,
				Type:     fs.nomsType,
				Optional: fs.omitEmpty,
			}
		}
		structType = types.MakeStructType(strings.Title(t.Name()), structTypeFields...)
	}
	return
}

func listEncoder(t reflect.Type, parentStructTypes []reflect.Type) encoderFunc {
	e := encoderCache.get(t)
	if e != nil {
		return e
	}

	var elemEncoder encoderFunc
	// lock e until encoder(s) are initialized
	var init sync.RWMutex
	init.Lock()
	defer init.Unlock()
	e = func(v reflect.Value) types.Value {
		init.RLock()
		defer init.RUnlock()
		values := make([]types.Value, v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			values[i] = elemEncoder(v.Index(i))
		}
		return types.NewList(values...)
	}

	encoderCache.set(t, e)
	elemEncoder = typeEncoder(t.Elem(), parentStructTypes, nomsTags{})
	return e
}

// Encode set from array or slice
func setFromListEncoder(t reflect.Type, parentStructTypes []reflect.Type) encoderFunc {
	e := setEncoderCache.get(t)
	if e != nil {
		return e
	}

	var elemEncoder encoderFunc
	// lock e until encoder(s) are initialized
	var init sync.RWMutex
	init.Lock()
	defer init.Unlock()
	e = func(v reflect.Value) types.Value {
		init.RLock()
		defer init.RUnlock()
		values := make([]types.Value, v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			values[i] = elemEncoder(v.Index(i))
		}
		return types.NewSet(values...)
	}

	setEncoderCache.set(t, e)
	elemEncoder = typeEncoder(t.Elem(), parentStructTypes, nomsTags{})
	return e
}

func setEncoder(t reflect.Type, parentStructTypes []reflect.Type) encoderFunc {
	e := setEncoderCache.get(t)
	if e != nil {
		return e
	}

	var encoder encoderFunc
	// lock e until encoder(s) are initialized
	var init sync.RWMutex
	init.Lock()
	defer init.Unlock()
	e = func(v reflect.Value) types.Value {
		init.RLock()
		defer init.RUnlock()
		values := make([]types.Value, v.Len(), v.Len())
		for i, k := range v.MapKeys() {
			values[i] = encoder(k)
		}
		return types.NewSet(values...)
	}

	setEncoderCache.set(t, e)
	encoder = typeEncoder(t.Key(), parentStructTypes, nomsTags{})
	return e
}

func mapEncoder(t reflect.Type, parentStructTypes []reflect.Type) encoderFunc {
	e := encoderCache.get(t)
	if e != nil {
		return e
	}

	var keyEncoder encoderFunc
	var valueEncoder encoderFunc
	// lock e until encoder(s) are initialized
	var init sync.RWMutex
	init.Lock()
	defer init.Unlock()
	e = func(v reflect.Value) types.Value {
		init.RLock()
		defer init.RUnlock()
		keys := v.MapKeys()
		kvs := make([]types.Value, 2*len(keys))
		for i, k := range keys {
			kvs[2*i] = keyEncoder(k)
			kvs[2*i+1] = valueEncoder(v.MapIndex(k))
		}
		return types.NewMap(kvs...)
	}

	encoderCache.set(t, e)
	keyEncoder = typeEncoder(t.Key(), parentStructTypes, nomsTags{})
	valueEncoder = typeEncoder(t.Elem(), parentStructTypes, nomsTags{})
	return e
}

func shouldEncodeAsSet(t reflect.Type, tags nomsTags) bool {
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return tags.set
	case reflect.Map:
		// map[T]struct{} `noms:,"set"`
		return tags.set &&
			t.Elem().Kind() == reflect.Struct &&
			t.Elem().NumField() == 0
	default:
		panic(fmt.Errorf("called with unexpected kind %v", t.Kind()))
	}
}
