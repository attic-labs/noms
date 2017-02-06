// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package ngql

import (
	"context"
	"fmt"

	"strings"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/graphql-go/graphql"
)

type typeMap map[hash.Hash]graphql.Type

// In terms of resolving a graph of data, there are three types of value: scalars, lists and maps.
// During resolution, we are converting some noms value to a graphql value. A getFieldFn will
// be invoked for a matching noms type. Its job is to retrieve the sub-value from the noms type
// which is mapped to a graphql map as a fieldname.
type getFieldFn func(v interface{}, fieldName string, ctx context.Context) types.Value

// When a field name is resolved, it may take key:value arguments. A getSubvaluesFn handles
// returning one or more *noms* values whose presence is indicated by the provided arguments.
type getSubvaluesFn func(v types.Value, args map[string]interface{}) (interface{}, error)

// Note: Always returns a graphql.NonNull() as the outer type.
func nomsTypeToGraphQLType(t *types.Type, tm typeMap) graphql.Type {
	gqlType, ok := tm[t.Hash()]
	if ok {
		return gqlType
	}

	// In order to handle cycles, we eagerly create the type so we can put it into the cache before
	// creating any subtypes. Since all noms-types are non-nullable, the graphql NonNull creates a
	// handy piece of state for us to mutate once the subtype is fully created
	newNonNull := &graphql.NonNull{}
	tm[t.Hash()] = newNonNull

	switch t.Kind() {
	case types.NumberKind:
		newNonNull.OfType = graphql.Float

	case types.StringKind:
		newNonNull.OfType = graphql.String

	case types.BoolKind:
		newNonNull.OfType = graphql.Boolean

	case types.StructKind:
		newNonNull.OfType = structToGQLObject(t, tm)

	case types.ListKind, types.SetKind:
		valueTyp := t.Desc.(types.CompoundDesc).ElemTypes[0]
		newNonNull.OfType = graphql.NewList(nomsTypeToGraphQLType(valueTyp, tm))

	case types.MapKind:
		keyTyp := t.Desc.(types.CompoundDesc).ElemTypes[0]
		valueTyp := t.Desc.(types.CompoundDesc).ElemTypes[1]
		newNonNull.OfType = graphql.NewList(mapEntryToGraphQLObject(keyTyp, valueTyp, tm))

	case types.RefKind:
		newNonNull.OfType = refToGraphQLObject(t, tm)

	case types.UnionKind:
		newNonNull.OfType = unionToGQLUnion(t, tm)

	case types.BlobKind, types.ValueKind, types.TypeKind:
		panic(fmt.Sprintf("%d: type not impemented", t.Kind()))

	case types.CycleKind:
		panic("not reached") // we should never attempt to create a schedule for any unresolved cycle

	default:
		panic("not reached")
	}

	return newNonNull
}

// Creates a union of structs type.
func unionToGQLUnion(typ *types.Type, tm typeMap) *graphql.Union {
	unionTyps := typ.Desc.(types.CompoundDesc).ElemTypes
	unionTypes := make([]*graphql.Object, len(unionTyps))

	for i, unionTyp := range unionTyps {
		if unionTyp.Kind() != types.StructKind {
			panic("booh: grqphql-go only supports unions of structs")
		}

		unionType := nomsTypeToGraphQLType(unionTyp, tm).(*graphql.NonNull).OfType.(*graphql.Object)
		unionTypes[i] = unionType
	}

	return graphql.NewUnion(graphql.UnionConfig{
		Name:  getTypeName(typ),
		Types: unionTypes,
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			tm := p.Context.Value(tmKey).(typeMap)
			typ := p.Value.(types.Value).Type()
			gqlType := tm[typ.Hash()].(*graphql.NonNull).OfType.(*graphql.Object)
			return gqlType
		},
	})
}

func structToGQLObject(typ *types.Type, tm typeMap) *graphql.Object {
	structDesc := typ.Desc.(types.StructDesc)
	fields := graphql.Fields{}

	structDesc.IterFields(func(name string, fieldTyp *types.Type) {
		fieldType := nomsTypeToGraphQLType(fieldTyp, tm)

		args, resolveFn := getArgsAndResolveFn(fieldTyp.Kind(), getStructField)
		fields[name] = &graphql.Field{
			Type:    fieldType,
			Args:    args,
			Resolve: resolveFn,
		}
	})

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   getTypeName(typ),
		Fields: fields,
	})
}

func getStructField(v interface{}, fieldName string, ctx context.Context) types.Value {
	return v.(types.Struct).Get(fieldName)
}

var listArgs = graphql.FieldConfigArgument{
	atKey:    &graphql.ArgumentConfig{Type: graphql.Int},
	countKey: &graphql.ArgumentConfig{Type: graphql.Int},
}

func getListValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	l := v.(types.List)
	idx := uint64(0)
	count := l.Len()
	if at, ok := args[atKey].(int); ok {
		idx = uint64(at)
	}
	if c, ok := args[countKey].(int); ok {
		count = uint64(c)
	}

	// Clamp ranges
	if idx < 0 {
		idx = 0
	}
	if idx > l.Len() {
		idx = l.Len()
	}
	if count < 0 {
		count = 0
	}
	if idx+count > l.Len() {
		count = l.Len() - idx
	}

	values := make([]interface{}, count)
	iter := l.IteratorAt(idx)
	for i := uint64(0); i < count; i++ {
		values[i] = maybeGetScalar(iter.Next())
	}

	return values, nil
}

var setArgs = graphql.FieldConfigArgument{
	countKey: &graphql.ArgumentConfig{Type: graphql.Int},
}

func getSetValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	s := v.(types.Set)

	count := s.Len()
	if c, ok := args[countKey].(int); ok {
		count = uint64(c)
	}

	// Clamp ranges
	if count < 0 {
		count = 0
	}
	if count > s.Len() {
		count = s.Len()
	}

	values := make([]interface{}, count)
	i := uint64(0)
	s.Iter(func(v types.Value) bool {
		values[i] = maybeGetScalar(v)
		i++
		return i >= count
	})

	return values, nil
}

var mapArgs = graphql.FieldConfigArgument{
	countKey: &graphql.ArgumentConfig{Type: graphql.Int},
}

func getMapValues(v types.Value, args map[string]interface{}) (interface{}, error) {
	m := v.(types.Map)

	count := m.Len()
	if c, ok := args[countKey].(int); ok {
		count = uint64(c)
	}

	// Clamp ranges
	if count < 0 {
		count = 0
	}
	if count > m.Len() {
		count = m.Len()
	}

	values := make([]mapEntry, count)
	i := uint64(0)
	m.Iter(func(k, v types.Value) bool {
		values[i] = mapEntry{k, v}
		i++
		return i >= count
	})

	return values, nil
}

type mapEntry struct {
	key, value types.Value
}

func getArgsAndResolveFn(kind types.NomsKind, getField getFieldFn) (args graphql.FieldConfigArgument, resolveFn graphql.FieldResolveFn) {
	getSubvalue := getScalarValue

	switch kind {
	case types.ListKind:
		args = listArgs
		getSubvalue = getListValues

	case types.SetKind:
		args = setArgs
		getSubvalue = getSetValues

	case types.MapKind:
		args = mapArgs
		getSubvalue = getMapValues
	}

	resolveFn = func(p graphql.ResolveParams) (interface{}, error) {
		field := getField(p.Source, p.Info.FieldName, p.Context)
		return getSubvalue(field, p.Args)
	}

	return
}

// Map data must be returned as a list of key-value pairs. Each unique keyType:valueType is
// represented as a graphql
//
// type <KeyTypeName><ValueTypeName>Entry {
//	 key: <KeyType>!
//	 value: <ValueType>!
// }
func mapEntryToGraphQLObject(keyTyp, valueTyp *types.Type, tm typeMap) *graphql.Object {
	keyType := nomsTypeToGraphQLType(keyTyp, tm)
	valueType := nomsTypeToGraphQLType(valueTyp, tm)

	keyArgs, keyResolveFn := getArgsAndResolveFn(keyTyp.Kind(), getMapEntryField)
	valueArgs, valueResolveFn := getArgsAndResolveFn(valueTyp.Kind(), getMapEntryField)

	return graphql.NewObject(graphql.ObjectConfig{
		Name: fmt.Sprintf("%s%sEntry", getTypeName(keyTyp), getTypeName(valueTyp)),
		Fields: graphql.Fields{
			keyKey: &graphql.Field{
				Type:    keyType,
				Args:    keyArgs,
				Resolve: keyResolveFn,
			},
			valueKey: &graphql.Field{
				Type:    valueType,
				Args:    valueArgs,
				Resolve: valueResolveFn,
			},
		}})
}

func getMapEntryField(v interface{}, fieldName string, ctx context.Context) types.Value {
	entry := v.(mapEntry)

	switch fieldName {
	case keyKey:
		return entry.key
	case valueKey:
		return entry.value
	default:
		panic("not reached")
	}
}

func getTypeName(typ *types.Type) string {
	switch typ.Kind() {
	case types.NumberKind:
		return "Number"

	case types.StringKind:
		return "String"

	case types.ValueKind:
		return "Value"

	case types.ListKind:
		return fmt.Sprintf("%sList", getTypeName(typ.Desc.(types.CompoundDesc).ElemTypes[0]))

	case types.MapKind:
		kn := getTypeName(typ.Desc.(types.CompoundDesc).ElemTypes[0])
		vn := getTypeName(typ.Desc.(types.CompoundDesc).ElemTypes[0])
		return fmt.Sprintf("%sTo%sMap", kn, vn)

	case types.RefKind:
		return fmt.Sprintf("%sRef", getTypeName(typ.Desc.(types.CompoundDesc).ElemTypes[0]))

	case types.SetKind:
		return fmt.Sprintf("%sSet", getTypeName(typ.Desc.(types.CompoundDesc).ElemTypes[0]))

	case types.StructKind:
		return typ.Desc.(types.StructDesc).Name

	case types.UnionKind:
		unionTyps := typ.Desc.(types.CompoundDesc).ElemTypes
		names := make([]string, len(unionTyps))
		for i, unionTyp := range unionTyps {
			names[i] = getTypeName(unionTyp)
		}
		return strings.Join(names, "Or")

	default:
		panic("type name not implemented")
	}
}

// Refs are represented as structs:
//
// type <ValueTypeName>Entry {
//	 targetHash: String!
//	 targetValue: <ValueType>!
// }
func refToGraphQLObject(typ *types.Type, tm typeMap) *graphql.Object {
	targetTyp := typ.Desc.(types.CompoundDesc).ElemTypes[0]
	targetType := nomsTypeToGraphQLType(targetTyp, tm)

	_, resolveTargetHashFn := getArgsAndResolveFn(types.StringKind, getRefField)
	args, resolveTargetValueFn := getArgsAndResolveFn(targetTyp.Kind(), getRefField)

	return graphql.NewObject(graphql.ObjectConfig{
		Name: getTypeName(typ),
		Fields: graphql.Fields{
			targetHashKey: &graphql.Field{
				Type:    graphql.String,
				Resolve: resolveTargetHashFn,
			},
			targetValueKey: &graphql.Field{
				Type:    targetType,
				Args:    args,
				Resolve: resolveTargetValueFn,
			},
		}})
}

func getRefField(v interface{}, fieldName string, ctx context.Context) types.Value {
	r := v.(types.Ref)

	switch fieldName {
	case targetHashKey:
		return types.String(r.TargetHash().String())
	case targetValueKey:
		return r.TargetValue(ctx.Value(vrKey).(types.ValueReader))
	default:
		panic("not reached")
	}
}

func maybeGetScalar(v types.Value) interface{} {
	switch v.(type) {
	case types.Bool:
		return bool(v.(types.Bool))
	case types.Number:
		return float64(v.(types.Number))
	case types.String:
		return string(v.(types.String))
	}

	return v
}

func getScalarValue(v types.Value, args map[string]interface{}) (interface{}, error) {
	return maybeGetScalar(v), nil
}
