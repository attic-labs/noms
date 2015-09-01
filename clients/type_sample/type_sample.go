package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

func main() {
	cs := &chunks.MemoryStore{}

	boolType := types.MakePrimitiveType("Bool", types.BoolKind)
	uint8Type := types.MakePrimitiveType("UInt8", types.UInt8Kind)
	stringType := types.MakePrimitiveType("String", types.StringKind)
	mapType := types.MakeMapType(types.NewString("MapOfStringToUInt8"), stringType, uint8Type)
	setType := types.MakeSetType(types.NewString("SetOfString"), stringType)
	mahType := types.MakeStructType(types.NewString("MahStruct"), types.NewMap(
		types.NewString("Field1"), stringType,
		types.NewString("Field2"), boolType))
	otherType := types.MakeStructType(types.NewString("MahOtherStruct"), types.NewMap(
		types.NewString("StructField"), mahType,
		types.NewString("StringField"), stringType))

	mRef := types.WriteValue(mapType, cs)
	setRef := types.WriteValue(setType, cs)
	otherRef := types.WriteValue(otherType, cs)
	mahRef := types.WriteValue(mahType, cs)

	printTypeRef := func(r ref.Ref) {
		b, err := ioutil.ReadAll(cs.Get(r))
		d.Chk.NoError(err)
		out := &bytes.Buffer{}
		d.Chk.NoError(json.Indent(out, b[1:], "", "  "))
		fmt.Printf("%s:\n%s\n", r.String(), out.Bytes())
	}

	printTypeRef(boolType.Ref())
	printTypeRef(uint8Type.Ref())
	printTypeRef(stringType.Ref())
	printTypeRef(mRef)
	printTypeRef(setRef)
	printTypeRef(mahRef)
	printTypeRef(otherRef)
}
