package types

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/chunks"
)

func TestTypes(t *testing.T) {
	assert := assert.New(t)
	cs := &chunks.MemoryStore{}

	boolType := MakePrimitiveType("Bool", BoolKind)
	uint8Type := MakePrimitiveType("UInt8", UInt8Kind)
	stringType := MakePrimitiveType("String", StringKind)
	mapType := MakeMapType(NewString("MapOfStringToUInt8"), stringType, uint8Type)
	setType := MakeSetType(NewString("SetOfString"), stringType)
	mahType := MakeStructType(NewString("MahStruct"), NewMap(
		NewString("Field1"), stringType,
		NewString("Field2"), boolType))
	otherType := MakeStructType(NewString("MahOtherStruct"), NewMap(
		NewString("StructField"), mahType,
		NewString("StringField"), stringType))

	mRef := WriteValue(mapType, cs)
	setRef := WriteValue(setType, cs)
	otherRef := WriteValue(otherType, cs)
	mahRef := WriteValue(mahType, cs)

	assert.True(otherType.Equals(ReadValue(otherRef, cs)))
	assert.True(mapType.Equals(ReadValue(mRef, cs)))
	assert.True(setType.Equals(ReadValue(setRef, cs)))
	assert.True(mahType.Equals(ReadValue(mahRef, cs)))
}
