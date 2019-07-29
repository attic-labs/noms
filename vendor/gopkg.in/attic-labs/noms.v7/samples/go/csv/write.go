// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package csv

import (
	"encoding/csv"
	"fmt"
	"io"

	"gopkg.in/attic-labs/noms.v7/go/d"
	"gopkg.in/attic-labs/noms.v7/go/types"
)

func getElemDesc(s types.Collection, index int) types.StructDesc {
	t := types.TypeOf(s).Desc.(types.CompoundDesc).ElemTypes[index]
	if types.StructKind != t.TargetKind() {
		d.Panic("Expected StructKind, found %s", t.Kind())
	}
	return t.Desc.(types.StructDesc)
}

// GetListElemDesc ensures that l is a types.List of structs, pulls the types.StructDesc that describes the elements of l out of vr, and returns the StructDesc.
func GetListElemDesc(l types.List, vr types.ValueReader) types.StructDesc {
	return getElemDesc(l, 0)
}

// GetMapElemDesc ensures that m is a types.Map of structs, pulls the types.StructDesc that describes the elements of m out of vr, and returns the StructDesc.
// If m is a nested types.Map of types.Map, then GetMapElemDesc will descend the levels of the enclosed types.Maps to get to a types.Struct
func GetMapElemDesc(m types.Map, vr types.ValueReader) types.StructDesc {
	t := types.TypeOf(m).Desc.(types.CompoundDesc).ElemTypes[1]
	if types.StructKind == t.TargetKind() {
		return t.Desc.(types.StructDesc)
	} else if types.MapKind == t.TargetKind() {
		_, v := m.First()
		return GetMapElemDesc(v.(types.Map), vr)
	}
	panic(fmt.Sprintf("Expected StructKind or MapKind, found %s", t.Kind().String()))
}

func writeValuesFromChan(structChan chan types.Struct, sd types.StructDesc, comma rune, output io.Writer) {
	fieldNames := getFieldNamesFromStruct(sd)
	csvWriter := csv.NewWriter(output)
	csvWriter.Comma = comma
	if csvWriter.Write(fieldNames) != nil {
		d.Panic("Failed to write header %v", fieldNames)
	}
	record := make([]string, len(fieldNames))
	for s := range structChan {
		for i, f := range fieldNames {
			record[i] = fmt.Sprintf("%v", s.Get(f))
		}
		if csvWriter.Write(record) != nil {
			d.Panic("Failed to write record %v", record)
		}
	}

	csvWriter.Flush()
	if csvWriter.Error() != nil {
		d.Panic("error flushing csv")
	}
}

// WriteList takes a types.List l of structs (described by sd) and writes it to output as comma-delineated values.
func WriteList(l types.List, sd types.StructDesc, comma rune, output io.Writer) {
	structChan := make(chan types.Struct, 1024)
	go func() {
		l.IterAll(func(v types.Value, index uint64) {
			structChan <- v.(types.Struct)
		})
		close(structChan)
	}()
	writeValuesFromChan(structChan, sd, comma, output)
}

func sendMapValuesToChan(m types.Map, structChan chan<- types.Struct) {
	m.IterAll(func(k, v types.Value) {
		if subMap, ok := v.(types.Map); ok {
			sendMapValuesToChan(subMap, structChan)
		} else {
			structChan <- v.(types.Struct)
		}
	})
}

// WriteMap takes a types.Map m of structs (described by sd) and writes it to output as comma-delineated values.
func WriteMap(m types.Map, sd types.StructDesc, comma rune, output io.Writer) {
	structChan := make(chan types.Struct, 1024)
	go func() {
		sendMapValuesToChan(m, structChan)
		close(structChan)
	}()
	writeValuesFromChan(structChan, sd, comma, output)
}

func getFieldNamesFromStruct(structDesc types.StructDesc) (fieldNames []string) {
	structDesc.IterFields(func(name string, t *types.Type, optional bool) {
		if !types.IsPrimitiveKind(t.TargetKind()) {
			d.Panic("Expected primitive kind, found %s", t.TargetKind().String())
		}
		fieldNames = append(fieldNames, name)
	})
	return
}
