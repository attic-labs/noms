// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package model

import (
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/photo-dedup/dhash"
)

type Photo interface {
	ID() ID
	Dhash() dhash.Hash
	SetDhash(dhash.Hash)
	IterSizes(cb func(width int, height int, url string))
	Marshal() types.Struct
}

type lazyPhoto struct {
	strct types.Struct
}

func UnmarshalPhoto(value types.Value) (Photo, bool) {
	sizeType := types.MakeStructTypeFromFields("", types.FieldMap{})
	photoType := types.MakeStructTypeFromFields("", types.FieldMap{
		field.id: types.StringType,
		field.sizes: types.MakeMapType(sizeType, types.StringType),
		field.title: types.StringType,
	})

	if types.IsSubtype(photoType, value.Type()) {
		return &lazyPhoto{value.(types.Struct)}, true
	}
	return nil, false
}

func (p *lazyPhoto) Dhash() dhash.Hash {
	v, ok := p.strct.MaybeGet(field.dhash)
	if !ok {
		return dhash.NilHash
	}
	h, err := dhash.Parse(string(v.(types.String)))
	if err != nil {
		return dhash.NilHash
	}
	return h
}

func (p *lazyPhoto) ID() ID {
	return UnmarshalID(p.strct.Get(field.id))
}

func (p *lazyPhoto) SetDhash(hash dhash.Hash) {
	p.strct = p.strct.Set(field.dhash, types.String(hash.String()))
}

func (p *lazyPhoto) IterSizes(cb func(width int, height int, url string)) {
	sizeMap := p.strct.Get(field.sizes).(types.Map)
	sizeMap.IterAll(func(k, v types.Value) {
		sz := k.(types.Struct)
		w := int(sz.Get(field.width).(types.Number))
		h := int(sz.Get(field.height).(types.Number))
		url := string(v.(types.String))
		cb(w, h, url)
	})

}

func (p *lazyPhoto) Marshal() types.Struct {
	return p.strct
}

