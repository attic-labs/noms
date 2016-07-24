// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package csv

import (
	"math"
	"strconv"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/types"
)

type schemaOptions []*typeCanFit

func newSchemaOptions(fieldCount int) schemaOptions {
	options := make([]*typeCanFit, fieldCount, fieldCount)
	for i := 0; i < fieldCount; i++ {
		options[i] = &typeCanFit{true, true, true}
	}
	return options
}

func (so schemaOptions) Test(fields []string) {
	for i, t := range so {
		if i < len(fields) {
			t.Test(fields[i])
		}
	}
}

func (so schemaOptions) ValidKinds() []KindSlice {
	kinds := make([]KindSlice, len(so))
	for i, t := range so {
		kinds[i] = t.ValidKinds()
	}
	return kinds
}

type typeCanFit struct {
	boolType   bool
	numberType bool
	stringType bool
}

func (tc *typeCanFit) ValidKinds() (kinds KindSlice) {
	if tc.numberType {
		kinds = append(kinds, types.NumberKind)
	}
	if tc.boolType {
		kinds = append(kinds, types.BoolKind)
	}

	kinds = append(kinds, types.StringKind)
	return kinds
}

func (tc *typeCanFit) Test(value string) {
	tc.testNumbers(value)
	tc.testBool(value)
}

func (tc *typeCanFit) testNumbers(value string) {
	if !tc.numberType {
		return
	}

	fval, err := strconv.ParseFloat(value, 64)
	if err != nil {
		tc.numberType = false
		return
	}

	if fval > math.MaxFloat64 {
		tc.numberType = false
	}
}

func (tc *typeCanFit) testBool(value string) {
	if !tc.boolType {
		return
	}
	_, err := strconv.ParseBool(value)
	tc.boolType = err == nil
}

// StringToType takes a piece of data as a string and attempts to convert it to a types.Value of the appropriate types.NomsKind.
func StringToType(s string, k types.NomsKind) types.Value {
	switch k {
	case types.NumberKind:
		if s == "" {
			return types.Number(float64(0))
		}
		fval, err := strconv.ParseFloat(s, 64)
		if err != nil {
			d.Chk.NoError(err, "Could not parse '%s' into number", s)
		}
		return types.Number(fval)
	case types.BoolKind:
		// TODO: This should probably be configurable.
		switch s {
		case "true", "1", "y", "Y":
			return types.Bool(true)
		case "false", "0", "n", "N", "":
			return types.Bool(false)
		default:
			d.Chk.Fail("Could not parse '%s' into bool", s)
		}
	case types.StringKind:
		return types.String(s)
	default:
		d.PanicIfTrue(true, "Invalid column type kind:", k)
	}
	panic("not reached")
}
