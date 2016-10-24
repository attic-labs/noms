// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package model

import (
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/photo-dedup/dhash"
	"github.com/attic-labs/noms/go/d"
)

type PhotoGroup interface {
	ID() ID
	Dhash() dhash.Hash
	Cover() Photo
	Add(photo Photo) bool
	Marshal() types.Struct
}

type photoGroup struct {
	id     ID
	dhash  dhash.Hash
	photos map[Photo]bool
}

func NewPhotoGroup(initialPhoto Photo) PhotoGroup {
	return &photoGroup{NewAtticID(), initialPhoto.Dhash(), map[Photo]bool{initialPhoto: true}}
}

func (pg *photoGroup) ID() ID {
	return pg.id
}

func (pg *photoGroup) Dhash() dhash.Hash {
	return pg.dhash
}

func (pg *photoGroup) Cover() Photo {
	return pg.pickCover()
}

func (pg *photoGroup) Add(photo Photo) bool {
	if pg.photos[photo] {
		return false
	}
	pg.photos[photo] = true
	return true
}

func (pg *photoGroup) Marshal() types.Struct {
	photos := []types.Value{}
	cover := pg.Cover()
	for p, _ := range pg.photos {
		if p != cover {
			photos = append(photos, p.Marshal())
		}
	}
	data := map[string]types.Value{
		field.id:     pg.id.Marshal(),
		field.dhash:  types.String(pg.dhash.String()),
		field.cover:  cover.Marshal(),
		field.photos: types.NewSet(photos...),
	}
	return types.NewStruct("PhotoGroup", data)
}

func (pg *photoGroup) pickCover() Photo {
	for p, _ := range pg.photos {
		return p
	}
	d.PanicIfTrue(true, "unreachable")
	return nil
}
