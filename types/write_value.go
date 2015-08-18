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

func WriteValue(v Value, cs chunks.ChunkSink) (ref.Ref, error) {
	d.Chk.NotNil(cs)

	e, err := toEncodeable(v, cs)
	if err != nil {
		return ref.Ref{}, err
	}

	dst := cs.Put()
	err = enc.Encode(dst, e)
	if err != nil {
		return ref.Ref{}, err
	}
	return dst.Ref(), nil
}

func toEncodeable(v Value, cs chunks.ChunkSink) (interface{}, error) {
	switch v := v.(type) {
	case blobLeaf:
		return v.Reader(), nil
	case compoundBlob:
		cb, err := encCompoundBlobFromCompoundBlob(v, cs)
		if err != nil {
			return nil, err
		}
		return cb, nil
	case List:
		l, err := makeListEncodeable(v, cs)
		if err != nil {
			return nil, err
		}
		return l, nil
	case Map:
		m, err := makeMapEncodeable(v, cs)
		if err != nil {
			return nil, err
		}
		return m, nil
	case primitive:
		return v.ToPrimitive(), nil
	case Ref:
		return v.Ref(), nil
	case Set:
		s, err := makeSetEncodeable(v, cs)
		if err != nil {
			return nil, err
		}
		return s, nil
	case String:
		return v.String(), nil
	default:
		return v, nil
	}
}

func encCompoundBlobFromCompoundBlob(cb compoundBlob, cs chunks.ChunkSink) (interface{}, error) {
	refs := make([]ref.Ref, len(cb.blobs))
	for idx, f := range cb.blobs {
		i, err := processChild(f, cs)
		if err != nil {
			return nil, err
		}
		// All children of compoundBlob must be Blobs, which get encoded and reffed by processChild.
		refs[idx] = i.(ref.Ref)
	}
	return enc.CompoundBlob{Offsets: cb.offsets, Blobs: refs}, nil
}

func makeListEncodeable(l List, cs chunks.ChunkSink) (interface{}, error) {
	items := make([]interface{}, l.Len())
	for idx, f := range l.list {
		i, err := processChild(f, cs)
		if err != nil {
			return nil, err
		}
		items[idx] = i
	}
	return items, nil
}

func makeMapEncodeable(m Map, cs chunks.ChunkSink) (interface{}, error) {
	j := make([]interface{}, 0, 2*len(m.m))
	for _, r := range m.m {
		var cjk, cjv interface{}
		cjk, err := processChild(r.key, cs)
		if err == nil {
			cjv, err = processChild(r.value, cs)
		}
		if err != nil {
			return nil, err
		}
		j = append(j, cjk)
		j = append(j, cjv)
	}
	return enc.MapFromItems(j...), nil
}

func makeSetEncodeable(s Set, cs chunks.ChunkSink) (interface{}, error) {
	items := make([]interface{}, s.Len())
	for idx, f := range s.m {
		i, err := processChild(f, cs)
		if err != nil {
			return nil, err
		}
		items[idx] = i
	}
	return enc.SetFromItems(items...), nil
}

func processChild(f Future, cs chunks.ChunkSink) (interface{}, error) {
	var r ref.Ref
	var err error
	if v, ok := f.(*unresolvedFuture); ok {
		return v.Ref(), nil
	}

	v := f.Val()
	d.Chk.NotNil(v)
	switch v := v.(type) {
	// Blobs, lists, maps, and sets are always out-of-line
	case Blob, List, Map, Set:
		r, err = WriteValue(v, cs)
		if err != nil {
			return nil, err
		}
		return r, nil
	default:
		// Other types are always inline.
		return toEncodeable(v, cs)
	}
}

func (b Bool) ToPrimitive() interface{} {
	return bool(b)
}

func (i Int8) ToPrimitive() interface{} {
	return int8(i)
}

func (i Int16) ToPrimitive() interface{} {
	return int16(i)
}

func (i Int32) ToPrimitive() interface{} {
	return int32(i)
}

func (i Int64) ToPrimitive() interface{} {
	return int64(i)
}

func (f Float32) ToPrimitive() interface{} {
	return float32(f)
}

func (f Float64) ToPrimitive() interface{} {
	return float64(f)
}

func (u UInt8) ToPrimitive() interface{} {
	return uint8(u)
}

func (u UInt16) ToPrimitive() interface{} {
	return uint16(u)
}

func (u UInt32) ToPrimitive() interface{} {
	return uint32(u)
}

func (u UInt64) ToPrimitive() interface{} {
	return uint64(u)
}
