// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package job

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/walk"
	"github.com/attic-labs/noms/samples/go/photo-dedup/model"
	"github.com/attic-labs/noms/go/util/status"
)

var grouper *photoGrouper

// DeduplicateJob reads Set<Photo>'s (annotated with dhash) and writes Set<PhotoGroup> to
// outDS where each group contains all duplicates.
func DeduplicateJob(db datas.Database, photoSets []types.Value, outDS datas.Dataset) error {
	return commitPhotoGroups(db, outDS, groupPhotos(db, photoSets))
}

// groupPhotos reads Set<Photo>'s and sorts them into groups containing all photos that
// are deemed duplicates by comparing dhash's
func groupPhotos(db datas.Database, photoSets []types.Value) <-chan types.Struct {
	grouper = newPhotoGrouper()
	for _, set := range photoSets {
		walk.WalkValues(set, db, func(cv types.Value) (stop bool) {
			if photo, ok := model.UnmarshalPhoto(cv); ok {
				grouper.insertPhoto(photo)
			}
			return false
		})
	}
	photoGroupsCh := make(chan types.Struct)
	go func() {
		defer close(photoGroupsCh)
		grouper.iterGroups(func(pg model.PhotoGroup) {
			photoGroupsCh <- pg.Marshal()
		})
	}()
	return photoGroupsCh
}

// commitPhotoGroups commits the new groups to ds
func commitPhotoGroups(db datas.Database, ds datas.Dataset, groups <-chan types.Struct) error {
	newSet := types.NewGraphBuilder(db, types.SetKind, true)
	for group := range groups {
		newSet.SetInsert(nil, group)
	}
	status.Done()
	fmt.Printf("\nCommitting %d PhotoGroups\n", grouper.photoCount)
	commit := newSet.Build()
	_, err := db.Commit(ds, commit, datas.CommitOptions{
		Meta: model.NewCommitMeta().Marshal(),
	})
	return err
}
