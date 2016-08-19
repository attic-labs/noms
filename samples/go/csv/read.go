// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// StringToKind maps names of valid NomsKinds (e.g. Bool, Number, etc) to their associated types.NomsKind
var StringToKind = func(kindMap map[types.NomsKind]string) map[string]types.NomsKind {
	m := map[string]types.NomsKind{}
	for k, v := range kindMap {
		m[v] = k
	}
	return m
}(types.KindToString)

// StringsToKinds looks up each element of strs in the StringToKind map and returns a slice of answers
func StringsToKinds(strs []string) KindSlice {
	kinds := make(KindSlice, len(strs))
	for i, str := range strs {
		k, ok := StringToKind[str]
		d.PanicIfTrue(!ok, "StringToKind[%s] failed", str)
		kinds[i] = k
	}
	return kinds
}

// KindsToStrings looks up each element of kinds in the types.KindToString map and returns a slice of answers
func KindsToStrings(kinds KindSlice) []string {
	strs := make([]string, len(kinds))
	for i, k := range kinds {
		strs[i] = types.KindToString[k]
	}
	return strs
}

// MakeStructTypeFromHeaders creates a struct type from the headers using |kinds| as the type of each field. If |kinds| is empty, default to strings.
func MakeStructTypeFromHeaders(headers []string, structName string, kinds KindSlice) (typ *types.Type, fieldOrder []int, kindMap []types.NomsKind) {
	useStringType := len(kinds) == 0
	d.Chk.True(useStringType || len(headers) == len(kinds))

	fieldMap := make(types.TypeMap, len(headers))
	origOrder := make(map[string]int, len(headers))
	fieldNames := make(sort.StringSlice, len(headers))

	for i, key := range headers {
		fn := types.EscapeStructField(key)
		origOrder[fn] = i
		kind := types.StringKind
		if !useStringType {
			kind = kinds[i]
		}
		_, ok := fieldMap[fn]
		d.PanicIfTrue(ok, `Duplicate field name "%s"`, key)
		fieldMap[fn] = types.MakePrimitiveType(kind)
		fieldNames[i] = fn
	}

	sort.Sort(fieldNames)

	kindMap = make([]types.NomsKind, len(fieldMap))
	fieldOrder = make([]int, len(fieldMap))
	fieldTypes := make([]*types.Type, len(fieldMap))

	for i, fn := range fieldNames {
		typ := fieldMap[fn]
		fieldTypes[i] = typ
		kindMap[i] = typ.Kind()
		fieldOrder[origOrder[fn]] = i
	}

	typ = types.MakeStructType(structName, fieldNames, fieldTypes)
	return
}

// ReadToList takes a CSV reader and reads data into a typed List of structs. Each row gets read into a struct named structName, described by headers. If the original data contained headers it is expected that the input reader has already read those and are pointing at the first data row.
// If kinds is non-empty, it will be used to type the fields in the generated structs; otherwise, they will be left as string-fields.
// In addition to the list, ReadToList returns the typeDef of the structs in the list.
func ReadToList(r *csv.Reader, structName string, headers []string, kinds KindSlice, vrw types.ValueReadWriter) (l types.List, t *types.Type) {
	t, fieldOrder, kindMap := MakeStructTypeFromHeaders(headers, structName, kinds)
	valueChan := make(chan types.Value, 128) // TODO: Make this a function param?
	listChan := types.NewStreamingList(vrw, valueChan)

	for {
		row, err := r.Read()
		if err == io.EOF {
			close(valueChan)
			break
		} else if err != nil {
			panic(err)
		}

		fields := make(types.ValueSlice, len(headers))
		for i, v := range row {
			if i < len(headers) {
				fieldOrigIndex := fieldOrder[i]
				val, err := StringToValue(v, kindMap[fieldOrigIndex])
				if err != nil {
					d.Chk.Fail(fmt.Sprintf("Error parsing value for column '%s': %s", headers[i], err))
				}
				fields[fieldOrigIndex] = val
			}
		}
		valueChan <- types.NewStructWithType(t, fields)
	}

	return <-listChan, t
}

// getFieldIndexByHeaderName takes the collection of headers and the name to search for and returns the index of name within the headers or -1 if not found
func getFieldIndexByHeaderName(headers []string, name string) int {
	for i, header := range headers {
		if header == name {
			return i
		}
	}
	return -1
}

// getPkIndices takes collection of primary keys as strings and determines if they are integers, if so then use those ints as the indices, otherwise it looks up the strings in the headers to find the indices; returning the collection of int indices representing the primary keys maintaining the order of strPks to the return collection
func getPkIndices(strPks []string, headers []string) []int {
	result := make([]int, len(strPks))
	for i, pk := range strPks {
		pkIdx, ok := strconv.Atoi(pk)
		if ok == nil {
			result[i] = pkIdx
		} else {
			result[i] = getFieldIndexByHeaderName(headers, pk)
		}
		if result[i] < 0 {
			d.Chk.Fail(fmt.Sprintf("Invalid pk: %v", pk))
		}
	}
	return result
}

// ReadToMap takes a CSV reader and reads data into a typed Map of structs. Each row gets read into a struct named structName, described by headers. If the original data contained headers it is expected that the input reader has already read those and are pointing at the first data row.
// If kinds is non-empty, it will be used to type the fields in the generated structs; otherwise, they will be left as string-fields.
func ReadToMap(r *csv.Reader, structName string, headersRaw []string, primaryKeys []string, kinds KindSlice, vrw types.ValueReadWriter) types.Map {
	t, fieldOrder, kindMap := MakeStructTypeFromHeaders(headersRaw, structName, kinds)

	pkIndices := getPkIndices(primaryKeys, headersRaw)

	kvChan := make(chan types.Value, 128)
	mapChan := types.NewStreamingMap(vrw, kvChan)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		fields := make(types.ValueSlice, len(headersRaw))
		for i, v := range row {
			if i < len(headersRaw) {
				fieldOrigIndex := fieldOrder[i]
				fields[fieldOrigIndex], err = StringToValue(v, kindMap[fieldOrigIndex])
				if err != nil {
					d.Chk.Fail(fmt.Sprintf("Error parsing value for column '%s': %s", headersRaw[i], err))
				}
			}
		}

		// work backward through the pks so that outermost one is the first/primary one
		// e.g. map<pk1, map<pk2, map<pk3, fields>>>
		var v types.Value
		v = types.NewStructWithType(t, fields)
		for i := len(pkIndices); i > 1; i-- {
			fieldOrigIndex := fieldOrder[pkIndices[i-1]]
			key := fields[fieldOrigIndex]
			m := types.NewMap(key, v)
			v = m
		}

		fieldOrigIndex := fieldOrder[pkIndices[0]]
		kvChan <- fields[fieldOrigIndex]
		kvChan <- v
	}

	close(kvChan)
	return <-mapChan
}
