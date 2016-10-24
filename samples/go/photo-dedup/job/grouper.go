// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package job

import (
	"github.com/attic-labs/noms/go/util/status"
	"github.com/attic-labs/noms/samples/go/photo-dedup/dhash"
	"github.com/attic-labs/noms/samples/go/photo-dedup/model"
)

// photoGrouper is a data structure used to group similar photos into PhotoGroups
//
// The current implementation is a simple map. Photo inserts are O(n^2).
// TODO: Replace the map with VP/MVP tree (https://en.wikipedia.org/wiki/Vantage-point_tree).
type photoGrouper struct {
	groups         map[model.ID]model.PhotoGroup
	photoCount     int
	duplicateCount int
}

func newPhotoGrouper() *photoGrouper {
	return &photoGrouper{make(map[model.ID]model.PhotoGroup), 0, 0}
}

func (g *photoGrouper) insertGroup(group model.PhotoGroup) {
	status.Printf("Grouping - %d duplicates found in %d photos", g.duplicateCount, g.photoCount)
	g.groups[group.ID()] = group
}

// insertPhoto places the photo into an existing group if there is one that contains
// duplicate photos. Otherwise it creates a new group.
//
// The current implementation is a brute force n^2 comparision. A more efficient
// implementation would be to build an VP/MVP tree. A VP tree is a binary search
// tree that works in a geometric space. Each node defines a center point and a
// radius. Dhashes within the radius can be found to the left; those outside the
// radius can be found to the right. An MVP is the k-tree equivalent.
func (g *photoGrouper) insertPhoto(photo model.Photo) {
	//
	const similarityThreshold = 10
	for _, group := range g.groups {
		if group.Dhash() != dhash.NilHash {
			if dhash.Distance(photo.Dhash(), group.Dhash()) < similarityThreshold {
				if group.Add(photo) {
					g.duplicateCount++
					g.photoCount++
				}
				return
			}
		}
	}
	g.insertGroup(model.NewPhotoGroup(photo))
	g.photoCount++
}

// iterGroups iterator through all the photo groups
func (g *photoGrouper) iterGroups(cb func(pg model.PhotoGroup)) {
	for _, group := range g.groups {
		cb(group)
	}
}
