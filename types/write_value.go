package types

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/enc"
	"github.com/attic-labs/noms/ref"
)

type primitive interface {
	ToPrimitive() interface{}
}

func WriteValue(v Value, cs chunks.ChunkSink) ref.Ref {
	d.Chk.NotNil(cs)

	e := toEncodeable(v, cs)
	dst := cs.Put()
	enc.Encode(dst, e)
	return dst.Ref()
}

func toEncodeable(v Value, cs chunks.ChunkSink) interface{} {
	switch v := v.(type) {
	case blobLeaf:
		return v.Reader()
	case compoundBlob:
		return encCompoundBlobFromCompoundBlob(v, cs)
	case compoundList:
		return encCompoundListFromCompoundList(v, cs)
	case listLeaf:
		return makeListEncodeable(v, cs)
	case Map:
		return makeMapEncodeable(v, cs)
	case primitive:
		return v.ToPrimitive()
	case Ref:
		return v.Ref()
	case Set:
		return makeSetEncodeable(v, cs)
	case String:
		return v.String()
	case Type:
		return makeTypeEncodeable(v, cs)
	default:
		return v
	}
}

func encCompoundBlobFromCompoundBlob(cb compoundBlob, cs chunks.ChunkSink) interface{} {
	refs := make([]ref.Ref, len(cb.blobs))
	for idx, f := range cb.blobs {
		i := processChild(f, cs)
		// All children of compoundBlob must be Blobs, which get encoded and reffed by processChild.
		refs[idx] = i.(ref.Ref)
	}
	return enc.CompoundBlob{Offsets: cb.offsets, Blobs: refs}
}

func encCompoundListFromCompoundList(cl compoundList, cs chunks.ChunkSink) interface{} {
	refs := make([]ref.Ref, len(cl.lists))
	for idx, f := range cl.lists {
		i := processChild(f, cs)
		// All children of compoundList must be Lists, which get encoded and reffed by processChild.
		refs[idx] = i.(ref.Ref)
	}
	return enc.CompoundList{Offsets: cl.offsets, Lists: refs}
}

func makeListEncodeable(l listLeaf, cs chunks.ChunkSink) interface{} {
	items := make([]interface{}, l.Len())
	for idx, f := range l.list {
		items[idx] = processChild(f, cs)
	}
	return items
}

func makeMapEncodeable(m Map, cs chunks.ChunkSink) interface{} {
	j := make([]interface{}, 0, 2*len(m.m))
	for _, r := range m.m {
		j = append(j, processChild(r.key, cs))
		j = append(j, processChild(r.value, cs))
	}
	return enc.MapFromItems(j...)
}

func makeSetEncodeable(s Set, cs chunks.ChunkSink) interface{} {
	items := make([]interface{}, s.Len())
	for idx, f := range s.m {
		items[idx] = processChild(f, cs)
	}
	return enc.SetFromItems(items...)
}

func makeTypeEncodeable(t Type, cs chunks.ChunkSink) interface{} {
	j := make([]interface{}, 0, 2*len(t.Desc.m))
	for _, r := range t.Desc.m {
		j = append(j, processChild(r.key, cs))
		j = append(j, processChild(r.value, cs))
	}
	return enc.Type{Name: t.Name.String(), Kind: uint8(t.Kind), Desc: enc.MapFromItems(j...)}
}

func processChild(f Future, cs chunks.ChunkSink) interface{} {
	if v, ok := f.(*unresolvedFuture); ok {
		return v.Ref()
	}

	v := f.Val()
	d.Exp.NotNil(v)
	switch v := v.(type) {
	// Blobs, lists, maps, and sets are always out-of-line
	case Blob, List, Map, Set, Type:
		return WriteValue(v, cs)
	default:
		// Other types are always inline.
		return toEncodeable(v, cs)
	}
}
