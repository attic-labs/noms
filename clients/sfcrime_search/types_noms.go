// This file was generated by nomdl/codegen.

package main

import (
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __mainPackageInFile_types_CachedRef ref.Ref

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func init() {
	p := types.NewPackage([]types.TypeRef{
		types.MakeStructTypeRef("Incident",
			[]types.Field{
				types.Field{"Category", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Description", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Address", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Date", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Geoposition", types.MakeTypeRef(ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"), 0), false},
			},
			types.Choices{},
		),
		types.MakeStructTypeRef("SQuadTree",
			[]types.Field{
				types.Field{"Nodes", types.MakeCompoundTypeRef(types.ListKind, types.MakeTypeRef(ref.Ref{}, 0)), false},
				types.Field{"Tiles", types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeTypeRef(ref.Ref{}, 1)), false},
				types.Field{"Depth", types.MakePrimitiveTypeRef(types.UInt8Kind), false},
				types.Field{"NumDescendents", types.MakePrimitiveTypeRef(types.UInt32Kind), false},
				types.Field{"Path", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Georectangle", types.MakeTypeRef(ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"), 1), false},
			},
			types.Choices{},
		),
	}, []ref.Ref{
		ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"),
	})
	__mainPackageInFile_types_CachedRef = types.RegisterPackage(&p)
}

// Incident

type Incident struct {
	m   types.Map
	ref *ref.Ref
}

func NewIncident() Incident {
	return Incident{types.NewMap(
		types.NewString("Category"), types.NewString(""),
		types.NewString("Description"), types.NewString(""),
		types.NewString("Address"), types.NewString(""),
		types.NewString("Date"), types.NewString(""),
		types.NewString("Geoposition"), NewGeoposition(),
	), &ref.Ref{}}
}

type IncidentDef struct {
	Category    string
	Description string
	Address     string
	Date        string
	Geoposition GeopositionDef
}

func (def IncidentDef) New() Incident {
	return Incident{
		types.NewMap(
			types.NewString("Category"), types.NewString(def.Category),
			types.NewString("Description"), types.NewString(def.Description),
			types.NewString("Address"), types.NewString(def.Address),
			types.NewString("Date"), types.NewString(def.Date),
			types.NewString("Geoposition"), def.Geoposition.New(),
		), &ref.Ref{}}
}

func (s Incident) Def() (d IncidentDef) {
	d.Category = s.m.Get(types.NewString("Category")).(types.String).String()
	d.Description = s.m.Get(types.NewString("Description")).(types.String).String()
	d.Address = s.m.Get(types.NewString("Address")).(types.String).String()
	d.Date = s.m.Get(types.NewString("Date")).(types.String).String()
	d.Geoposition = s.m.Get(types.NewString("Geoposition")).(Geoposition).Def()
	return
}

var __typeRefForIncident types.TypeRef

func (m Incident) TypeRef() types.TypeRef {
	return __typeRefForIncident
}

func init() {
	__typeRefForIncident = types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0)
	types.RegisterFromValFunction(__typeRefForIncident, func(v types.Value) types.Value {
		return IncidentFromVal(v)
	})
}

func IncidentFromVal(val types.Value) Incident {
	// TODO: Do we still need FromVal?
	if val, ok := val.(Incident); ok {
		return val
	}
	// TODO: Validate here
	return Incident{val.(types.Map), &ref.Ref{}}
}

func (s Incident) InternalImplementation() types.Map {
	return s.m
}

func (s Incident) Equals(other types.Value) bool {
	return other != nil && s.Ref() == other.Ref()
}

func (s Incident) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Incident) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, s.TypeRef().Chunks()...)
	chunks = append(chunks, s.m.Chunks()...)
	return
}

func (s Incident) Category() string {
	return s.m.Get(types.NewString("Category")).(types.String).String()
}

func (s Incident) SetCategory(val string) Incident {
	return Incident{s.m.Set(types.NewString("Category"), types.NewString(val)), &ref.Ref{}}
}

func (s Incident) Description() string {
	return s.m.Get(types.NewString("Description")).(types.String).String()
}

func (s Incident) SetDescription(val string) Incident {
	return Incident{s.m.Set(types.NewString("Description"), types.NewString(val)), &ref.Ref{}}
}

func (s Incident) Address() string {
	return s.m.Get(types.NewString("Address")).(types.String).String()
}

func (s Incident) SetAddress(val string) Incident {
	return Incident{s.m.Set(types.NewString("Address"), types.NewString(val)), &ref.Ref{}}
}

func (s Incident) Date() string {
	return s.m.Get(types.NewString("Date")).(types.String).String()
}

func (s Incident) SetDate(val string) Incident {
	return Incident{s.m.Set(types.NewString("Date"), types.NewString(val)), &ref.Ref{}}
}

func (s Incident) Geoposition() Geoposition {
	return s.m.Get(types.NewString("Geoposition")).(Geoposition)
}

func (s Incident) SetGeoposition(val Geoposition) Incident {
	return Incident{s.m.Set(types.NewString("Geoposition"), val), &ref.Ref{}}
}

// SQuadTree

type SQuadTree struct {
	m   types.Map
	ref *ref.Ref
}

func NewSQuadTree() SQuadTree {
	return SQuadTree{types.NewMap(
		types.NewString("Nodes"), NewListOfIncident(),
		types.NewString("Tiles"), NewMapOfStringToSQuadTree(),
		types.NewString("Depth"), types.UInt8(0),
		types.NewString("NumDescendents"), types.UInt32(0),
		types.NewString("Path"), types.NewString(""),
		types.NewString("Georectangle"), NewGeorectangle(),
	), &ref.Ref{}}
}

type SQuadTreeDef struct {
	Nodes          ListOfIncidentDef
	Tiles          MapOfStringToSQuadTreeDef
	Depth          uint8
	NumDescendents uint32
	Path           string
	Georectangle   GeorectangleDef
}

func (def SQuadTreeDef) New() SQuadTree {
	return SQuadTree{
		types.NewMap(
			types.NewString("Nodes"), def.Nodes.New(),
			types.NewString("Tiles"), def.Tiles.New(),
			types.NewString("Depth"), types.UInt8(def.Depth),
			types.NewString("NumDescendents"), types.UInt32(def.NumDescendents),
			types.NewString("Path"), types.NewString(def.Path),
			types.NewString("Georectangle"), def.Georectangle.New(),
		), &ref.Ref{}}
}

func (s SQuadTree) Def() (d SQuadTreeDef) {
	d.Nodes = s.m.Get(types.NewString("Nodes")).(ListOfIncident).Def()
	d.Tiles = s.m.Get(types.NewString("Tiles")).(MapOfStringToSQuadTree).Def()
	d.Depth = uint8(s.m.Get(types.NewString("Depth")).(types.UInt8))
	d.NumDescendents = uint32(s.m.Get(types.NewString("NumDescendents")).(types.UInt32))
	d.Path = s.m.Get(types.NewString("Path")).(types.String).String()
	d.Georectangle = s.m.Get(types.NewString("Georectangle")).(Georectangle).Def()
	return
}

var __typeRefForSQuadTree types.TypeRef

func (m SQuadTree) TypeRef() types.TypeRef {
	return __typeRefForSQuadTree
}

func init() {
	__typeRefForSQuadTree = types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 1)
	types.RegisterFromValFunction(__typeRefForSQuadTree, func(v types.Value) types.Value {
		return SQuadTreeFromVal(v)
	})
}

func SQuadTreeFromVal(val types.Value) SQuadTree {
	// TODO: Do we still need FromVal?
	if val, ok := val.(SQuadTree); ok {
		return val
	}
	// TODO: Validate here
	return SQuadTree{val.(types.Map), &ref.Ref{}}
}

func (s SQuadTree) InternalImplementation() types.Map {
	return s.m
}

func (s SQuadTree) Equals(other types.Value) bool {
	return other != nil && s.Ref() == other.Ref()
}

func (s SQuadTree) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s SQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, s.TypeRef().Chunks()...)
	chunks = append(chunks, s.m.Chunks()...)
	return
}

func (s SQuadTree) Nodes() ListOfIncident {
	return s.m.Get(types.NewString("Nodes")).(ListOfIncident)
}

func (s SQuadTree) SetNodes(val ListOfIncident) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("Nodes"), val), &ref.Ref{}}
}

func (s SQuadTree) Tiles() MapOfStringToSQuadTree {
	return s.m.Get(types.NewString("Tiles")).(MapOfStringToSQuadTree)
}

func (s SQuadTree) SetTiles(val MapOfStringToSQuadTree) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("Tiles"), val), &ref.Ref{}}
}

func (s SQuadTree) Depth() uint8 {
	return uint8(s.m.Get(types.NewString("Depth")).(types.UInt8))
}

func (s SQuadTree) SetDepth(val uint8) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("Depth"), types.UInt8(val)), &ref.Ref{}}
}

func (s SQuadTree) NumDescendents() uint32 {
	return uint32(s.m.Get(types.NewString("NumDescendents")).(types.UInt32))
}

func (s SQuadTree) SetNumDescendents(val uint32) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("NumDescendents"), types.UInt32(val)), &ref.Ref{}}
}

func (s SQuadTree) Path() string {
	return s.m.Get(types.NewString("Path")).(types.String).String()
}

func (s SQuadTree) SetPath(val string) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("Path"), types.NewString(val)), &ref.Ref{}}
}

func (s SQuadTree) Georectangle() Georectangle {
	return s.m.Get(types.NewString("Georectangle")).(Georectangle)
}

func (s SQuadTree) SetGeorectangle(val Georectangle) SQuadTree {
	return SQuadTree{s.m.Set(types.NewString("Georectangle"), val), &ref.Ref{}}
}

// ListOfIncident

type ListOfIncident struct {
	l   types.List
	ref *ref.Ref
}

func NewListOfIncident() ListOfIncident {
	return ListOfIncident{types.NewList(), &ref.Ref{}}
}

type ListOfIncidentDef []IncidentDef

func (def ListOfIncidentDef) New() ListOfIncident {
	l := make([]types.Value, len(def))
	for i, d := range def {
		l[i] = d.New()
	}
	return ListOfIncident{types.NewList(l...), &ref.Ref{}}
}

func (l ListOfIncident) Def() ListOfIncidentDef {
	d := make([]IncidentDef, l.Len())
	for i := uint64(0); i < l.Len(); i++ {
		d[i] = l.l.Get(i).(Incident).Def()
	}
	return d
}

func ListOfIncidentFromVal(val types.Value) ListOfIncident {
	// TODO: Do we still need FromVal?
	if val, ok := val.(ListOfIncident); ok {
		return val
	}
	// TODO: Validate here
	return ListOfIncident{val.(types.List), &ref.Ref{}}
}

func (l ListOfIncident) InternalImplementation() types.List {
	return l.l
}

func (l ListOfIncident) Equals(other types.Value) bool {
	return other != nil && l.Ref() == other.Ref()
}

func (l ListOfIncident) Ref() ref.Ref {
	return types.EnsureRef(l.ref, l)
}

func (l ListOfIncident) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, l.TypeRef().Chunks()...)
	chunks = append(chunks, l.l.Chunks()...)
	return
}

// A Noms Value that describes ListOfIncident.
var __typeRefForListOfIncident types.TypeRef

func (m ListOfIncident) TypeRef() types.TypeRef {
	return __typeRefForListOfIncident
}

func init() {
	__typeRefForListOfIncident = types.MakeCompoundTypeRef(types.ListKind, types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 0))
	types.RegisterFromValFunction(__typeRefForListOfIncident, func(v types.Value) types.Value {
		return ListOfIncidentFromVal(v)
	})
}

func (l ListOfIncident) Len() uint64 {
	return l.l.Len()
}

func (l ListOfIncident) Empty() bool {
	return l.Len() == uint64(0)
}

func (l ListOfIncident) Get(i uint64) Incident {
	return l.l.Get(i).(Incident)
}

func (l ListOfIncident) Slice(idx uint64, end uint64) ListOfIncident {
	return ListOfIncident{l.l.Slice(idx, end), &ref.Ref{}}
}

func (l ListOfIncident) Set(i uint64, val Incident) ListOfIncident {
	return ListOfIncident{l.l.Set(i, val), &ref.Ref{}}
}

func (l ListOfIncident) Append(v ...Incident) ListOfIncident {
	return ListOfIncident{l.l.Append(l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfIncident) Insert(idx uint64, v ...Incident) ListOfIncident {
	return ListOfIncident{l.l.Insert(idx, l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfIncident) Remove(idx uint64, end uint64) ListOfIncident {
	return ListOfIncident{l.l.Remove(idx, end), &ref.Ref{}}
}

func (l ListOfIncident) RemoveAt(idx uint64) ListOfIncident {
	return ListOfIncident{(l.l.RemoveAt(idx)), &ref.Ref{}}
}

func (l ListOfIncident) fromElemSlice(p []Incident) []types.Value {
	r := make([]types.Value, len(p))
	for i, v := range p {
		r[i] = v
	}
	return r
}

type ListOfIncidentIterCallback func(v Incident, i uint64) (stop bool)

func (l ListOfIncident) Iter(cb ListOfIncidentIterCallback) {
	l.l.Iter(func(v types.Value, i uint64) bool {
		return cb(v.(Incident), i)
	})
}

type ListOfIncidentIterAllCallback func(v Incident, i uint64)

func (l ListOfIncident) IterAll(cb ListOfIncidentIterAllCallback) {
	l.l.IterAll(func(v types.Value, i uint64) {
		cb(v.(Incident), i)
	})
}

type ListOfIncidentFilterCallback func(v Incident, i uint64) (keep bool)

func (l ListOfIncident) Filter(cb ListOfIncidentFilterCallback) ListOfIncident {
	nl := NewListOfIncident()
	l.IterAll(func(v Incident, i uint64) {
		if cb(v, i) {
			nl = nl.Append(v)
		}
	})
	return nl
}

// MapOfStringToSQuadTree

type MapOfStringToSQuadTree struct {
	m   types.Map
	ref *ref.Ref
}

func NewMapOfStringToSQuadTree() MapOfStringToSQuadTree {
	return MapOfStringToSQuadTree{types.NewMap(), &ref.Ref{}}
}

type MapOfStringToSQuadTreeDef map[string]SQuadTreeDef

func (def MapOfStringToSQuadTreeDef) New() MapOfStringToSQuadTree {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), v.New())
	}
	return MapOfStringToSQuadTree{types.NewMap(kv...), &ref.Ref{}}
}

func (m MapOfStringToSQuadTree) Def() MapOfStringToSQuadTreeDef {
	def := make(map[string]SQuadTreeDef)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = v.(SQuadTree).Def()
		return false
	})
	return def
}

func MapOfStringToSQuadTreeFromVal(val types.Value) MapOfStringToSQuadTree {
	// TODO: Do we still need FromVal?
	if val, ok := val.(MapOfStringToSQuadTree); ok {
		return val
	}
	// TODO: Validate here
	return MapOfStringToSQuadTree{val.(types.Map), &ref.Ref{}}
}

func (m MapOfStringToSQuadTree) InternalImplementation() types.Map {
	return m.m
}

func (m MapOfStringToSQuadTree) Equals(other types.Value) bool {
	return other != nil && m.Ref() == other.Ref()
}

func (m MapOfStringToSQuadTree) Ref() ref.Ref {
	return types.EnsureRef(m.ref, m)
}

func (m MapOfStringToSQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, m.TypeRef().Chunks()...)
	chunks = append(chunks, m.m.Chunks()...)
	return
}

// A Noms Value that describes MapOfStringToSQuadTree.
var __typeRefForMapOfStringToSQuadTree types.TypeRef

func (m MapOfStringToSQuadTree) TypeRef() types.TypeRef {
	return __typeRefForMapOfStringToSQuadTree
}

func init() {
	__typeRefForMapOfStringToSQuadTree = types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeTypeRef(__mainPackageInFile_types_CachedRef, 1))
	types.RegisterFromValFunction(__typeRefForMapOfStringToSQuadTree, func(v types.Value) types.Value {
		return MapOfStringToSQuadTreeFromVal(v)
	})
}

func (m MapOfStringToSQuadTree) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToSQuadTree) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToSQuadTree) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToSQuadTree) Get(p string) SQuadTree {
	return m.m.Get(types.NewString(p)).(SQuadTree)
}

func (m MapOfStringToSQuadTree) MaybeGet(p string) (SQuadTree, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewSQuadTree(), false
	}
	return v.(SQuadTree), ok
}

func (m MapOfStringToSQuadTree) Set(k string, v SQuadTree) MapOfStringToSQuadTree {
	return MapOfStringToSQuadTree{m.m.Set(types.NewString(k), v), &ref.Ref{}}
}

// TODO: Implement SetM?

func (m MapOfStringToSQuadTree) Remove(p string) MapOfStringToSQuadTree {
	return MapOfStringToSQuadTree{m.m.Remove(types.NewString(p)), &ref.Ref{}}
}

type MapOfStringToSQuadTreeIterCallback func(k string, v SQuadTree) (stop bool)

func (m MapOfStringToSQuadTree) Iter(cb MapOfStringToSQuadTreeIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(SQuadTree))
	})
}

type MapOfStringToSQuadTreeIterAllCallback func(k string, v SQuadTree)

func (m MapOfStringToSQuadTree) IterAll(cb MapOfStringToSQuadTreeIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), v.(SQuadTree))
	})
}

type MapOfStringToSQuadTreeFilterCallback func(k string, v SQuadTree) (keep bool)

func (m MapOfStringToSQuadTree) Filter(cb MapOfStringToSQuadTreeFilterCallback) MapOfStringToSQuadTree {
	nm := NewMapOfStringToSQuadTree()
	m.IterAll(func(k string, v SQuadTree) {
		if cb(k, v) {
			nm = nm.Set(k, v)
		}
	})
	return nm
}
