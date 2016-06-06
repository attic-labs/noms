// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package csv

import (
	"encoding/csv"
	"io"
	"regexp"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

// StringToKind maps names of valid NomsKinds (e.g. Bool, Float32, etc) to their associated types.NomsKind
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
		d.Exp.True(ok)
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

// ReportValidFieldTypes takes a CSV reader and the headers. It returns a slice of types.NomsKind for each column in the data indicating what Noms types could be used to represent that row.
// For example, if all values in a row are negative integers between -127 and 0, the slice for that row would be [types.Int8Kind, types.Int16Kind, types.Int32Kind, types.Int64Kind, types.Float32Kind, types.Float64Kind, types.StringKind]. If even one value in the row is a floating point number, however, all the integer kinds would be dropped. All values can be represented as a string, so that option is always provided.
func ReportValidFieldTypes(r *csv.Reader, headers []string) []KindSlice {
	options := newSchemaOptions(len(headers))
	rowChan := make(chan []string)
	doneChan := make(chan struct{})
	go func() {
		for row := range rowChan {
			options.Test(row)
		}
		doneChan <- struct{}{}
	}()

	for {
		row, err := r.Read()
		if err == io.EOF {
			close(rowChan)
			break
		}
		d.Exp.NoError(err, "Error decoding CSV")

		rowChan <- row
	}
	<-doneChan
	return options.ValidKinds()
}

// Field names are of the form `^[a-zA-Z][a-zA-Z0-9_]*$`)
var fieldNameFirstCharRe = regexp.MustCompile(`^[a-zA-Z]`)
var fieldNameTailNonValidRe = regexp.MustCompile(`[^a-zA-Z0-9]`)

// MakeStructTypeFromHeaders creates a struct type from the headers using |kinds| as the type of each field. If |kinds| is empty, default to strings.
func MakeStructTypeFromHeaders(headers []string, structName string, kinds KindSlice) *types.Type {
	useStringType := len(kinds) == 0
	d.Chk.True(useStringType || len(headers) == len(kinds))
	fields := make(types.TypeMap, len(headers))
	for i, key := range headers {
		kind := types.StringKind
		if !useStringType {
			kind = kinds[i]
		}
		_, ok := fields[key]
		d.Exp.False(ok, `Duplicate field name "%s"`, key)
		fields[key] = types.MakePrimitiveType(kind)
	}
	return types.MakeStructType(structName, fields)
}

// Read takes a CSV reader and reads it into a typed List of structs. Each row gets read into a struct named structName, described by headers. If the original data contained headers it is expected that the input reader has already read those and are pointing at the first data row.
// If kinds is non-empty, it will be used to type the fields in the generated structs; otherwise, they will be left as string-fields.
// In addition to the list, Read returns the typeRef for the structs in the list, and last the typeDef of the structs.
func Read(r *csv.Reader, structName string, headers_raw []string, kinds KindSlice, vrw types.ValueReadWriter) (l types.List, t *types.Type) {
	headers := make([]string, len(headers_raw))
	for i, h := range headers_raw {
		headers[i] = types.EscapeStructField(h)
	}

	t = MakeStructTypeFromHeaders(headers, structName, kinds)
	valueChan := make(chan types.Value, 128) // TODO: Make this a function param?
	listChan := types.NewStreamingList(vrw, valueChan)

	kindMap := make(map[string]types.NomsKind, len(headers))
	t.Desc.(types.StructDesc).IterFields(func(name string, t *types.Type) {
		kindMap[name] = t.Kind()
	})

	for {
		row, err := r.Read()
		if err == io.EOF {
			close(valueChan)
			break
		} else if err != nil {
			panic(err)
		}

		fields := make(map[string]types.Value)
		for i, v := range row {
			if i < len(headers) {
				name := headers[i]
				fields[name] = StringToType(v, kindMap[name])
			}
		}
		valueChan <- types.NewStructWithType(t, fields)
	}

	return <-listChan, t
}
