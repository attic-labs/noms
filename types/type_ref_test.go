package types

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

func TestTypes(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewMemoryStore()

	boolType := MakePrimitiveTypeRef(BoolKind)
	uint8Type := MakePrimitiveTypeRef(UInt8Kind)
	stringType := MakePrimitiveTypeRef(StringKind)
	mapType := MakeCompoundTypeRef("MapOfStringToUInt8", MapKind, stringType, uint8Type)
	setType := MakeCompoundTypeRef("SetOfString", SetKind, stringType)
	mahType := MakeStructTypeRef("MahStruct", []Field{
		Field{"Field1", stringType, false},
		Field{"Field2", boolType, true},
	}, Choices{})
	otherType := MakeStructTypeRef("MahOtherStruct", []Field{}, Choices{
		Field{"StructField", mahType, false},
		Field{"StringField", stringType, false},
	})
	pkgRef := ref.Parse("sha1-0123456789abcdef0123456789abcdef01234567")
	trType := MakeTypeRef(pkgRef, 42)

	mRef := WriteValue(mapType, cs)
	setRef := WriteValue(setType, cs)
	otherRef := WriteValue(otherType, cs)
	mahRef := WriteValue(mahType, cs)
	trRef := WriteValue(trType, cs)

	assert.True(otherType.Equals(ReadValue(otherRef, cs)))
	assert.True(mapType.Equals(ReadValue(mRef, cs)))
	assert.True(setType.Equals(ReadValue(setRef, cs)))
	assert.True(mahType.Equals(ReadValue(mahRef, cs)))
	assert.True(trType.Equals(ReadValue(trRef, cs)))
}

func TestTypeWithPkgRef(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewMemoryStore()

	pkg := NewPackage([]TypeRef{MakePrimitiveTypeRef(Float64Kind)}, []ref.Ref{})

	pkgRef := RegisterPackage(&pkg)
	unresolvedType := MakeTypeRef(pkgRef, 42)
	unresolvedRef := WriteValue(unresolvedType, cs)

	assert.EqualValues(pkgRef, ReadValue(unresolvedRef, cs).Chunks()[0].Ref())
	assert.NotNil(ReadValue(pkgRef, cs))
}

func TestTypeRefTypeRef(t *testing.T) {
	assert.True(t, MakePrimitiveTypeRef(BoolKind).TypeRef().Equals(MakePrimitiveTypeRef(TypeRefKind)))
}
