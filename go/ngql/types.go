// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import (
	"context"
	"fmt"

	"strings"

	"github.com/attic-labs/graphql"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

type typeMap map[typeMapKey]graphql.Type

type typeMapKey struct {
	h             hash.Hash
	boxedIfScalar bool
}

func newTypeMap() *typeMap {
	return &typeMap{}
}

// In terms of resolving a graph of data, there are three types of value:
// scalars, lists and maps. During resolution, we are converting some noms
// value to a graphql value. A getFieldFn will be invoked for a matching noms
// type. Its job is to retrieve the sub-value from the noms type which is
// mapped to a graphql map as a fieldname.
type getFieldFn func(v interface{}, fieldName string, ctx context.Context) types.Value

// When a field name is resolved, it may take key:value arguments. A
// getSubvaluesFn handles returning one or more *noms* values whose presence is
// indicated by the provided arguments.
type getSubvaluesFn func(v types.Value, args map[string]interface{}) (interface{}, error)

// GraphQL requires all memberTypes in a Union to be Structs, so when a noms
// union contains a scalar, we represent it in that context as a "boxed" value.
// E.g.
// Boolean! =>
// type BooleanValue {
//   scalarValue: Boolean!
// }
func scalarToValue(nomsType *types.Type, scalarType graphql.Type, tm *typeMap) graphql.Type {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: fmt.Sprintf("%sValue", getTypeName(nomsType)),
		Fields: graphql.Fields{
			scalarValue: &graphql.Field{
				Type: graphql.NewNonNull(scalarType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source, nil // p.Source is already a go-native scalar type
				},
			},
		}})
}

func isScalar(nomsType *types.Type) bool {
	switch nomsType {
	case types.BoolType, types.NumberType, types.StringType:
		return true
	default:
		return false
	}
}

// Note: Always returns a graphql.NonNull() as the outer type.
func nomsTypeToGraphQLType(nomsType *types.Type, boxedIfScalar bool, tm *typeMap) graphql.Type {
	key := typeMapKey{nomsType.Hash(), boxedIfScalar && isScalar(nomsType)}
	gqlType, ok := (*tm)[key]
	if ok {
		return gqlType
	}

	// The graphql package has built in support for recursive types using
	// FieldsThunk which allows the inner type to refer to an outer type by
	// lazily initializing the fields.
	switch nomsType.Kind() {
	case types.NumberKind:
		gqlType = graphql.Float
		if boxedIfScalar {
			gqlType = scalarToValue(nomsType, gqlType, tm)
		}

	case types.StringKind:
		gqlType = graphql.String
		if boxedIfScalar {
			gqlType = scalarToValue(nomsType, gqlType, tm)
		}

	case types.BoolKind:
		gqlType = graphql.Boolean
		if boxedIfScalar {
			gqlType = scalarToValue(nomsType, gqlType, tm)
		}

	case types.StructKind:
		gqlType = structToGQLObject(nomsType, tm)

	case types.ListKind, types.SetKind:
		nomsValueType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
		var valueType graphql.Type
		if !isEmptyNomsUnion(nomsValueType) {
			valueType = nomsTypeToGraphQLType(nomsValueType, false, tm)
		}

		gqlType = collectionToGraphQLObject(nomsType, valueType, unpackNonNullType(valueType), tm)

	case types.MapKind:
		nomsKeyType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
		nomsValueType := nomsType.Desc.(types.CompoundDesc).ElemTypes[1]
		var entryType, keyType graphql.Type
		if !isEmptyNomsUnion(nomsKeyType) && !isEmptyNomsUnion(nomsValueType) {
			entryType, keyType = mapEntryToGraphQLObject(nomsKeyType, nomsValueType, tm)
		}

		gqlType = collectionToGraphQLObject(nomsType, entryType, unpackNonNullType(keyType), tm)

	case types.RefKind:
		gqlType = refToGraphQLObject(nomsType, tm)

	case types.UnionKind:
		gqlType = unionToGQLUnion(nomsType, tm)

	case types.BlobKind, types.ValueKind, types.TypeKind:
		// TODO: https://github.com/attic-labs/noms/issues/3155
		gqlType = graphql.String

	case types.CycleKind:
		panic("not reached") // we should never attempt to create a schedule for any unresolved cycle

	default:
		panic("not reached")
	}

	newNonNull := graphql.NewNonNull(gqlType)
	(*tm)[key] = newNonNull
	return newNonNull
}

func unpackNonNullType(t graphql.Type) graphql.Type {
	if t, ok := t.(*graphql.NonNull); ok {
		return t.OfType
	}
	return t
}

func isEmptyNomsUnion(nomsType *types.Type) bool {
	return nomsType.Kind() == types.UnionKind && len(nomsType.Desc.(types.CompoundDesc).ElemTypes) == 0
}

// Creates a union of structs type.
func unionToGQLUnion(nomsType *types.Type, tm *typeMap) *graphql.Union {
	nomsMemberTypes := nomsType.Desc.(types.CompoundDesc).ElemTypes
	memberTypes := make([]*graphql.Object, len(nomsMemberTypes))

	for i, nomsUnionType := range nomsMemberTypes {
		// Member types cannot be non-null and must be struct (graphl.Object)
		memberTypes[i] = nomsTypeToGraphQLType(nomsUnionType, true, tm).(*graphql.NonNull).OfType.(*graphql.Object)
	}

	return graphql.NewUnion(graphql.UnionConfig{
		Name:  getTypeName(nomsType),
		Types: memberTypes,
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			tm := p.Context.Value(tmKey).(*typeMap)
			var nomsType *types.Type
			isScalar := false
			if v, ok := p.Value.(types.Value); ok {
				nomsType = v.Type()
			} else {
				switch p.Value.(type) {
				case float64:
					nomsType = types.NumberType
					isScalar = true
				case string:
					nomsType = types.StringType
					isScalar = true
				case bool:
					nomsType = types.BoolType
					isScalar = true
				}
			}
			key := typeMapKey{nomsType.Hash(), isScalar}
			memberType := (*tm)[key]
			// Member types cannot be non-null and must be struct (graphl.Object)
			return memberType.(*graphql.NonNull).OfType.(*graphql.Object)
		},
	})
}

func structToGQLObject(nomsType *types.Type, tm *typeMap) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: getTypeName(nomsType),
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			structDesc := nomsType.Desc.(types.StructDesc)
			fields := graphql.Fields{
				"hash": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Source.(types.Struct).Hash().String(), nil
					},
				},
			}

			structDesc.IterFields(func(name string, nomsFieldType *types.Type) {
				fieldType := nomsTypeToGraphQLType(nomsFieldType, false, tm)

				fields[name] = &graphql.Field{
					Type: fieldType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						field := p.Source.(types.Struct).Get(p.Info.FieldName)
						return maybeGetScalar(field), nil
					},
				}
			})

			return fields
		}),
	})
}

var listArgs = graphql.FieldConfigArgument{
	atKey:    &graphql.ArgumentConfig{Type: graphql.Int},
	countKey: &graphql.ArgumentConfig{Type: graphql.Int},
}

func getListValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	l := v.(types.List)
	idx := 0
	count := int(l.Len())
	len := count

	if at, ok := args[atKey].(int); ok {
		idx = at
	}

	if c, ok := args[countKey].(int); ok {
		count = c
	}

	// Clamp ranges
	if count <= 0 || idx >= len {
		return ([]interface{})(nil), nil
	}
	if idx < 0 {
		idx = 0
	}
	if idx+count > len {
		count = len - idx
	}

	values := make([]interface{}, count)
	iter := l.IteratorAt(uint64(idx))
	for i := uint64(0); i < uint64(count); i++ {
		values[i] = maybeGetScalar(iter.Next())
	}

	return values, nil
}

func getSetValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	s := v.(types.Set)

	iter, nomsKey, nomsThrough, count, singleExactMatch, err := getCollectionArgs(s, args, iteratorFactory{
		IteratorFrom: func(from types.Value) interface{} {
			return s.IteratorFrom(from)
		},
		IteratorAt: func(at uint64) interface{} {
			return s.IteratorAt(at)
		},
	})

	if err != nil {
		return nil, err
	}
	if count == 0 {
		return ([]interface{})(nil), nil
	}

	setIter := iter.(types.SetIterator)
	values := make([]interface{}, 0, count)
	for i := uint64(0); i < count; i++ {
		v := setIter.Next()
		if v == nil {
			break
		}
		if singleExactMatch {
			if nomsKey.Equals(v) {
				values = append(values, maybeGetScalar(v))
			}
			break
		}

		if nomsThrough != nil {
			if !nomsThrough.Less(v) {
				values = append(values, maybeGetScalar(v))
			} else {
				break
			}
		} else {
			values = append(values, maybeGetScalar(v))
		}
	}

	return values, nil
}

type nomsCollection interface {
	types.Value
	Len() uint64
}

func getCollectionArgs(col nomsCollection, args map[string]interface{}, factory iteratorFactory) (iter interface{}, nomsKey, nomsThrough types.Value, count uint64, singleExactMatch bool, err error) {
	typ := col.Type()
	length := col.Len()
	nomsKeyType := typ.Desc.(types.CompoundDesc).ElemTypes[0]

	if keys, ok := args[keysKey]; ok {
		slice := keys.([]interface{})
		nomsKeys := make(types.ValueSlice, len(slice))
		for i, v := range slice {
			var nomsValue types.Value
			nomsValue, err = marshal.Marshal(v)
			if err != nil {
				return
			}
			nomsKeys[i] = nomsValue
		}
		count = uint64(len(slice))
		iter = &mapIteratorForKeys{
			m:    col.(types.Map),
			keys: nomsKeys,
		}
		return

	}

	if key, ok := args[keyKey]; ok {
		nomsKey, err = marshal.Marshal(key)
		if err != nil {
			return
		}
		iter = factory.IteratorFrom(nomsKey)
	} else if at, ok := args[atKey]; ok {
		idx := at.(int)
		if idx < 0 {
			idx = 0
		} else if uint64(idx) > length {
			// count is already 0
			return
		}
		iter = factory.IteratorAt(uint64(idx))
	} else {
		iter = factory.IteratorAt(0)
	}

	if err != nil {
		return
	}
	if iter == nil {
		// count is already 0
		return
	}

	nomsThrough, err = getThroughArg(nomsKeyType, args)
	if err != nil {
		return
	}

	count, singleExactMatch = getCountArg(length, args)
	return
}

func getMapValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	m := v.(types.Map)

	iter, nomsKey, nomsThrough, count, singleExactMatch, err := getCollectionArgs(m, args, iteratorFactory{
		IteratorFrom: func(from types.Value) interface{} {
			return m.IteratorFrom(from)
		},
		IteratorAt: func(at uint64) interface{} {
			return m.IteratorAt(at)
		},
	})

	if err != nil {
		return nil, err
	}
	if count == 0 {
		return ([]interface{})(nil), nil
	}

	mapIter := iter.(types.MapIterator)
	values := make([]mapEntry, 0, count)
	for i := uint64(0); i < count; i++ {
		k, v := mapIter.Next()
		if k == nil {
			break
		}

		if singleExactMatch {
			if nomsKey.Equals(k) {
				values = append(values, mapEntry{k, v})
			}
			break
		}

		if nomsThrough != nil {
			if !nomsThrough.Less(k) {
				values = append(values, mapEntry{k, v})
			} else {
				break
			}
		} else {
			values = append(values, mapEntry{k, v})
		}
	}

	return values, nil
}

func getCountArg(count uint64, args map[string]interface{}) (c uint64, singleExactMatch bool) {
	if c, ok := args[countKey]; ok {
		c := c.(int)
		if c <= 0 {
			return 0, false
		}
		return uint64(c), false
	}
	// If we have key and no count/through we use count 1
	_, hasKey := args[keyKey]
	_, hasThrough := args[throughKey]
	if hasKey && !hasThrough {
		return uint64(1), true
	}

	return count, false
}

func getThroughArg(nomsKeyType *types.Type, args map[string]interface{}) (types.Value, error) {
	var nomsThrough types.Value
	if through, ok := args[throughKey]; ok {
		var err error
		nomsThrough, err = marshal.Marshal(through)
		if err != nil {
			return nil, err
		}
	}
	return nomsThrough, nil
}

type iteratorFactory struct {
	IteratorFrom func(from types.Value) interface{}
	IteratorAt   func(at uint64) interface{}
}

type mapEntry struct {
	key, value types.Value
}

// Map data must be returned as a list of key-value pairs. Each unique keyType:valueType is
// represented as a graphql
//
// type <KeyTypeName><ValueTypeName>Entry {
//	 key: <KeyType>!
//	 value: <ValueType>!
// }
func mapEntryToGraphQLObject(nomsKeyType, nomsValueType *types.Type, tm *typeMap) (*graphql.Object, graphql.Type) {
	keyType := nomsTypeToGraphQLType(nomsKeyType, false, tm)
	return graphql.NewObject(graphql.ObjectConfig{
		Name: fmt.Sprintf("%s%sEntry", getTypeName(nomsKeyType), getTypeName(nomsValueType)),
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			valueType := nomsTypeToGraphQLType(nomsValueType, false, tm)
			return graphql.Fields{
				keyKey: &graphql.Field{
					Type: keyType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						entry := p.Source.(mapEntry)
						return maybeGetScalar(entry.key), nil
					},
				},
				valueKey: &graphql.Field{
					Type: valueType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						entry := p.Source.(mapEntry)
						return maybeGetScalar(entry.value), nil
					},
				},
			}
		}),
	}), keyType
}

func getTypeName(nomsType *types.Type) string {
	switch nomsType.Kind() {
	case types.BoolKind:
		return "Boolean"

	case types.NumberKind:
		return "Number"

	case types.StringKind:
		return "String"

	case types.BlobKind:
		return "Blob"

	case types.ValueKind:
		return "Value"

	case types.ListKind:
		nomsValueType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
		if isEmptyNomsUnion(nomsValueType) {
			return "EmptyList"
		}
		return fmt.Sprintf("%sList", getTypeName(nomsValueType))

	case types.MapKind:
		nomsKeyType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
		nomsValueType := nomsType.Desc.(types.CompoundDesc).ElemTypes[1]
		if isEmptyNomsUnion(nomsKeyType) {
			d.Chk.True(isEmptyNomsUnion(nomsValueType))
			return "EmptyMap"
		}

		return fmt.Sprintf("%sTo%sMap", getTypeName(nomsKeyType), getTypeName(nomsValueType))

	case types.RefKind:
		return fmt.Sprintf("%sRef", getTypeName(nomsType.Desc.(types.CompoundDesc).ElemTypes[0]))

	case types.SetKind:
		nomsValueType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
		if isEmptyNomsUnion(nomsValueType) {
			return "EmptySet"
		}

		return fmt.Sprintf("%sSet", getTypeName(nomsValueType))

	case types.StructKind:
		// GraphQL Name cannot start with a number.
		// GraphQL type names must be globally unique.
		return fmt.Sprintf("%s_%s", nomsType.Desc.(types.StructDesc).Name, nomsType.Hash().String()[:6])

	case types.TypeKind:
		// GraphQL Name cannot start with a number.
		// TODO: https://github.com/attic-labs/noms/issues/3155
		return fmt.Sprintf("Type_%s", nomsType.Hash().String()[:6])

	case types.UnionKind:
		unionMemberTypes := nomsType.Desc.(types.CompoundDesc).ElemTypes
		names := make([]string, len(unionMemberTypes))
		for i, unionMemberType := range unionMemberTypes {
			names[i] = getTypeName(unionMemberType)
		}
		return strings.Join(names, "Or")

	default:
		panic(fmt.Sprintf("%d: (getTypeName) not reached", nomsType.Kind()))
	}
}

func collectionToGraphQLObject(nomsType *types.Type, listType, keyType graphql.Type, tm *typeMap) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: getTypeName(nomsType),
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			fields := graphql.Fields{
				sizeKey: &graphql.Field{
					Type: graphql.Float,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						c := p.Source.(types.Collection)
						return maybeGetScalar(types.Number(c.Len())), nil
					},
				},
			}

			if listType != nil {
				var args graphql.FieldConfigArgument
				var getSubvalues getSubvaluesFn

				switch nomsType.Kind() {
				case types.ListKind:
					args = listArgs
					getSubvalues = getListValues

				case types.SetKind:
					args = graphql.FieldConfigArgument{
						atKey:    &graphql.ArgumentConfig{Type: graphql.Int},
						countKey: &graphql.ArgumentConfig{Type: graphql.Int},
					}
					if graphql.IsInputType(keyType) {
						// TODO: Should compute an graphql.InputObject from the noms type. graphql.InputObject is different from graphql.Object
						// See graphql.IsInputType vs graphql.IsOutputType
						args[keyKey] = &graphql.ArgumentConfig{Type: keyType}
						args[throughKey] = &graphql.ArgumentConfig{Type: keyType}
					}
					getSubvalues = getSetValues

				case types.MapKind:
					args = graphql.FieldConfigArgument{
						atKey:    &graphql.ArgumentConfig{Type: graphql.Int},
						countKey: &graphql.ArgumentConfig{Type: graphql.Int},
					}
					if graphql.IsInputType(keyType) {
						args[keyKey] = &graphql.ArgumentConfig{Type: keyType}
						args[keysKey] = &graphql.ArgumentConfig{Type: graphql.NewList(keyType)}
						args[throughKey] = &graphql.ArgumentConfig{Type: keyType}
					}
					getSubvalues = getMapValues
				}

				fields[elementsKey] = &graphql.Field{
					Type: graphql.NewList(listType),
					Args: args,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						c := p.Source.(types.Collection)
						return getSubvalues(c, p.Args)
					},
				}
			}

			return fields
		}),
	})
}

// Refs are represented as structs:
//
// type <ValueTypeName>Entry {
//	 targetHash: String!
//	 targetValue: <ValueType>!
// }
func refToGraphQLObject(nomsType *types.Type, tm *typeMap) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: getTypeName(nomsType),
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			nomsTargetType := nomsType.Desc.(types.CompoundDesc).ElemTypes[0]
			targetType := nomsTypeToGraphQLType(nomsTargetType, false, tm)

			return graphql.Fields{
				targetHashKey: &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						r := p.Source.(types.Ref)
						return maybeGetScalar(types.String(r.TargetHash().String())), nil
					},
				},

				targetValueKey: &graphql.Field{
					Type: targetType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						r := p.Source.(types.Ref)
						return maybeGetScalar(r.TargetValue(p.Context.Value(vrKey).(types.ValueReader))), nil
					},
				},
			}
		}),
	})
}

func maybeGetScalar(v types.Value) interface{} {
	switch v.(type) {
	case types.Bool:
		return bool(v.(types.Bool))
	case types.Number:
		return float64(v.(types.Number))
	case types.String:
		return string(v.(types.String))
	case *types.Type, types.Blob:
		// TODO: https://github.com/attic-labs/noms/issues/3155
		return v.Hash()
	}

	return v
}

type mapIteratorForKeys struct {
	m    types.Map
	keys types.ValueSlice
	idx  int
}

func (it *mapIteratorForKeys) Next() (k, v types.Value) {
	if it.idx >= len(it.keys) {
		return
	}
	k = it.keys[it.idx]
	it.idx++
	if k != nil {
		v = it.m.Get(k)
	}
	return
}
