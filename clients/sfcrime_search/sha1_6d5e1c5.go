// This file was generated by nomdl/codegen.

package main

import (
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __mainPackageInFile_sha1_6d5e1c5_CachedRef ref.Ref

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func init() {
	p := types.NewPackage([]types.TypeRef{
		types.MakeStructTypeRef("Geoposition",
			[]types.Field{
				types.Field{"Latitude", types.MakePrimitiveTypeRef(types.Float32Kind), false},
				types.Field{"Longitude", types.MakePrimitiveTypeRef(types.Float32Kind), false},
			},
			types.Choices{},
		),
		types.MakeStructTypeRef("Georectangle",
			[]types.Field{
				types.Field{"TopLeft", types.MakeTypeRef(ref.Ref{}, 0), false},
				types.Field{"BottomRight", types.MakeTypeRef(ref.Ref{}, 0), false},
			},
			types.Choices{},
		),
	}, []ref.Ref{})
	__mainPackageInFile_sha1_6d5e1c5_CachedRef = types.RegisterPackage(&p)
}

// Geoposition

type Geoposition struct {
	m   types.Map
	ref *ref.Ref
}

func NewGeoposition() Geoposition {
	return Geoposition{types.NewMap(
		types.NewString("Latitude"), types.Float32(0),
		types.NewString("Longitude"), types.Float32(0),
	), &ref.Ref{}}
}

type GeopositionDef struct {
	Latitude  float32
	Longitude float32
}

func (def GeopositionDef) New() Geoposition {
	return Geoposition{
		types.NewMap(
			types.NewString("Latitude"), types.Float32(def.Latitude),
			types.NewString("Longitude"), types.Float32(def.Longitude),
		), &ref.Ref{}}
}

func (s Geoposition) Def() (d GeopositionDef) {
	d.Latitude = float32(s.m.Get(types.NewString("Latitude")).(types.Float32))
	d.Longitude = float32(s.m.Get(types.NewString("Longitude")).(types.Float32))
	return
}

var __typeRefForGeoposition types.TypeRef

func (m Geoposition) TypeRef() types.TypeRef {
	return __typeRefForGeoposition
}

func init() {
	__typeRefForGeoposition = types.MakeTypeRef(__mainPackageInFile_sha1_6d5e1c5_CachedRef, 0)
	types.RegisterFromValFunction(__typeRefForGeoposition, func(v types.Value) types.Value {
		return GeopositionFromVal(v)
	})
}

func GeopositionFromVal(val types.Value) Geoposition {
	// TODO: Do we still need FromVal?
	if val, ok := val.(Geoposition); ok {
		return val
	}
	// TODO: Validate here
	return Geoposition{val.(types.Map), &ref.Ref{}}
}

func (s Geoposition) InternalImplementation() types.Map {
	return s.m
}

func (s Geoposition) Equals(other types.Value) bool {
	return other != nil && s.Ref() == other.Ref()
}

func (s Geoposition) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Geoposition) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, s.TypeRef().Chunks()...)
	chunks = append(chunks, s.m.Chunks()...)
	return
}

func (s Geoposition) Latitude() float32 {
	return float32(s.m.Get(types.NewString("Latitude")).(types.Float32))
}

func (s Geoposition) SetLatitude(val float32) Geoposition {
	return Geoposition{s.m.Set(types.NewString("Latitude"), types.Float32(val)), &ref.Ref{}}
}

func (s Geoposition) Longitude() float32 {
	return float32(s.m.Get(types.NewString("Longitude")).(types.Float32))
}

func (s Geoposition) SetLongitude(val float32) Geoposition {
	return Geoposition{s.m.Set(types.NewString("Longitude"), types.Float32(val)), &ref.Ref{}}
}

// Georectangle

type Georectangle struct {
	m   types.Map
	ref *ref.Ref
}

func NewGeorectangle() Georectangle {
	return Georectangle{types.NewMap(
		types.NewString("TopLeft"), NewGeoposition(),
		types.NewString("BottomRight"), NewGeoposition(),
	), &ref.Ref{}}
}

type GeorectangleDef struct {
	TopLeft     GeopositionDef
	BottomRight GeopositionDef
}

func (def GeorectangleDef) New() Georectangle {
	return Georectangle{
		types.NewMap(
			types.NewString("TopLeft"), def.TopLeft.New(),
			types.NewString("BottomRight"), def.BottomRight.New(),
		), &ref.Ref{}}
}

func (s Georectangle) Def() (d GeorectangleDef) {
	d.TopLeft = s.m.Get(types.NewString("TopLeft")).(Geoposition).Def()
	d.BottomRight = s.m.Get(types.NewString("BottomRight")).(Geoposition).Def()
	return
}

var __typeRefForGeorectangle types.TypeRef

func (m Georectangle) TypeRef() types.TypeRef {
	return __typeRefForGeorectangle
}

func init() {
	__typeRefForGeorectangle = types.MakeTypeRef(__mainPackageInFile_sha1_6d5e1c5_CachedRef, 1)
	types.RegisterFromValFunction(__typeRefForGeorectangle, func(v types.Value) types.Value {
		return GeorectangleFromVal(v)
	})
}

func GeorectangleFromVal(val types.Value) Georectangle {
	// TODO: Do we still need FromVal?
	if val, ok := val.(Georectangle); ok {
		return val
	}
	// TODO: Validate here
	return Georectangle{val.(types.Map), &ref.Ref{}}
}

func (s Georectangle) InternalImplementation() types.Map {
	return s.m
}

func (s Georectangle) Equals(other types.Value) bool {
	return other != nil && s.Ref() == other.Ref()
}

func (s Georectangle) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Georectangle) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, s.TypeRef().Chunks()...)
	chunks = append(chunks, s.m.Chunks()...)
	return
}

func (s Georectangle) TopLeft() Geoposition {
	return s.m.Get(types.NewString("TopLeft")).(Geoposition)
}

func (s Georectangle) SetTopLeft(val Geoposition) Georectangle {
	return Georectangle{s.m.Set(types.NewString("TopLeft"), val), &ref.Ref{}}
}

func (s Georectangle) BottomRight() Geoposition {
	return s.m.Get(types.NewString("BottomRight")).(Geoposition)
}

func (s Georectangle) SetBottomRight(val Geoposition) Georectangle {
	return Georectangle{s.m.Set(types.NewString("BottomRight"), val), &ref.Ref{}}
}
