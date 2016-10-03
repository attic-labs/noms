// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package noms4ios

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
)

func Version() string {
	return fmt.Sprintf("format version: %v\nbuilt from %v\n", constants.NomsVersion, constants.NomsGitSHA)
}

func CounterGetCurrentValue(dataset string) int64 {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not get dataset: %s\n", err)
		return 0
	}
	defer db.Close()

	if val, ok := ds.MaybeHeadValue(); ok {
		return int64(val.(types.Number))
	}
	return 0
}

func CounterIncrement(dataset string) {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not create dataset: %s\n", err)
		return
	}
	defer db.Close()

	newVal := uint64(1)
	if lastVal, ok := ds.MaybeHeadValue(); ok {
		newVal = uint64(lastVal.(types.Number)) + 1
	}

	_, err = db.CommitValue(ds, types.Number(newVal))
	if err != nil {
		fmt.Printf("Error committing: %s\n", err)
		return
	}
}

func BlobGet(path string) ([]byte, error) {
	var blob types.Blob
	if db, val, err := spec.GetPath(path); err != nil {
		return nil, err
	} else if val == nil {
		fmt.Printf("No value at %s", path)
		return nil, nil
	} else if b, ok := val.(types.Blob); !ok {
		fmt.Printf("Value at %s is not a blob", path)
		return nil, nil
	} else {
		defer db.Close()
		blob = b
	}

	return ioutil.ReadAll(blob.Reader())
}

func BlobPut(url, dataset string) {
	db, ds, err := spec.GetDataset(dataset)
	d.CheckErrorNoUsage(err)
	defer db.Close()

	// assume it's a file
	f, err := os.Open(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid URL %s - fopen error: %s", url, err)
		return
	}

	_, err = f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not stat file %s: %s", url, err)
		return
	}
	b := types.NewStreamingBlob(db, f)

	additionalMetaInfo := map[string]string{"file": url}
	meta, err := spec.CreateCommitMetaStruct(db, "", "", additionalMetaInfo, nil)
	d.CheckErrorNoUsage(err)
	ds, err = db.Commit(ds, b, datas.CommitOptions{Meta: meta})
	if err != nil {
		d.Chk.Equal(datas.ErrMergeNeeded, err)
		fmt.Fprintf(os.Stderr, "Could not commit, optimistic concurrency failed.")
		return
	}
}

func DsList(database string) string {
	cfg := config.NewResolver()
	store, err := cfg.GetDatabase(database)
	d.CheckError(err)
	defer store.Close()

	result := []string{}
	store.Datasets().IterAll(func(k, v types.Value) {
		result = append(result, string(k.(types.String)))
	})
	return strings.Join(result, ",")
}

func DsDelete(dataset string) {
	db, ds, err := spec.GetDataset(dataset) // append local if no local given, check for aliases
	d.CheckError(err)
	defer db.Close()

	oldCommitRef, errBool := ds.MaybeHeadRef()
	if !errBool {
		d.CheckError(fmt.Errorf("Dataset %v not found", ds.ID()))
	}

	_, err = db.Delete(ds)
	d.CheckError(err)

	fmt.Printf("Deleted %v (was #%v)\n", dataset, oldCommitRef.TargetHash().String())
}

func Sync(source, dest string, p int) error {
	cfg := config.NewResolver()
	sourceStore, sourceObj, err := cfg.GetPath(source)
	d.CheckError(err)
	defer sourceStore.Close()

	if sourceObj == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", source))
	}

	sinkDB, sinkDataset, err := spec.GetDataset(dest)
	d.CheckError(err)
	defer sinkDB.Close()

	sourceRef := types.NewRef(sourceObj)
	sinkRef, _ := sinkDataset.MaybeHeadRef()
	nonFF := false
	err = d.Try(func() {
		defer profile.MaybeStartProfile().Stop()
		datas.Pull(sourceStore, sinkDB, sourceRef, sinkRef, p, nil)

		var err error
		sinkDataset, err = sinkDB.FastForward(sinkDataset, sourceRef)
		if err == datas.ErrMergeNeeded {
			sinkDataset, err = sinkDB.SetHead(sinkDataset, sourceRef)
			nonFF = true
		}
		d.PanicIfError(err)
	})

	return err
}

func CountImagesInPhotos(dataset string) int {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not create dataset: %s\n", err)
		return 0
	}
	defer db.Close()

	if lastVal, ok := ds.MaybeHeadValue(); ok {
		photos := lastVal.(types.List)
		return int(photos.Len())
	}
	return 0
}

func WriteImageFromPhotos(dataset string, index int, url string) error {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not create dataset: %s\n", err)
		return err
	}
	defer db.Close()

	var photos types.List
	if lastVal, ok := ds.MaybeHeadValue(); ok {
		photos = lastVal.(types.List)
	} else {
		return errors.New("no images")
	}
	if index > int(photos.Len()) {
		return errors.New("index out of bounds")
	}

	b := photos.Get(uint64(index)).(types.Blob)

	// Note: overwrites any existing file.
	file, err := os.OpenFile(url, os.O_WRONLY|os.O_CREATE, 0644)
	d.CheckErrorNoUsage(err)
	defer file.Close()

	io.Copy(file, b.Reader())
	return nil
}

func AddImageToPhotos(url, msg, dataset string) {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not create dataset: %s\n", err)
		return
	}
	defer db.Close()

	// assume it's a file
	f, err := os.Open(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid URL %s - fopen error: %s", url, err)
		return
	}

	_, err = f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not stat file %s: %s", url, err)
		return
	}
	b := types.NewStreamingBlob(db, f)

	var photos types.List
	if lastVal, ok := ds.MaybeHeadValue(); ok {
		photos = lastVal.(types.List)
	} else {
		photos = types.NewList()
	}
	photos = photos.Append(b)

	additionalMetaInfo := map[string]string{"file": msg}
	meta, err := spec.CreateCommitMetaStruct(db, "", "", additionalMetaInfo, nil)
	d.CheckErrorNoUsage(err)
	ds, err = db.Commit(ds, photos, datas.CommitOptions{Meta: meta})
	if err != nil {
		d.Chk.Equal(datas.ErrMergeNeeded, err)
		fmt.Fprintf(os.Stderr, "Could not commit, optimistic concurrency failed.")
		return
	}
}

func photoIndexGetHeadStruct(dataset string) types.Struct {
	db, ds, err := spec.GetDataset(dataset)
	if err != nil {
		fmt.Printf("Could not create dataset: %s\n", err)
		return types.EmptyStruct
	}
	defer db.Close()

	if lastVal, ok := ds.MaybeHeadValue(); ok {
		return lastVal.(types.Struct)
	}
	return types.EmptyStruct
}

func photoIndexGetByTagMap(dataset string) *types.Map {
	s := photoIndexGetHeadStruct(dataset)
	if !types.EmptyStruct.Equals(s) {
		if imagesVal, ok := s.MaybeGet("byTag"); !ok {
			return nil
		} else {
			if imageSet, ok := imagesVal.(types.Map); !ok {
				return nil
			} else {
				return &imageSet
			}
		}
	}
	return nil
}

func photoIndexGetByDateMap(dataset string) *types.Map {
	s := photoIndexGetHeadStruct(dataset)
	if !types.EmptyStruct.Equals(s) {
		if imagesVal, ok := s.MaybeGet("byDate"); !ok {
			return nil
		} else {
			if imageSet, ok := imagesVal.(types.Map); !ok {
				return nil
			} else {
				return &imageSet
			}
		}
	}
	return nil
}

func PhotosGetCountOfTags(dataset string) int {
	m := photoIndexGetByTagMap(dataset)
	if m != nil {
		return int((*m).Len())
	}
	return 0
}

func PhotoIndexGetCountOfDates(dataset string) int {
	m := photoIndexGetByDateMap(dataset)
	if m != nil {
		return int((*m).Len())
	}
	return 0
}

func PhotoIndexGetDateByIndex(dataset string, index int) int {
	m := photoIndexGetByDateMap(dataset)
	if m != nil {
		i := 0
		var retValue types.Value
		m.Iter(func(key, value types.Value) (stop bool) {
			if i == index {
				retValue = key
				stop = true
			}
			i++
			return
		})
		if retValue != nil {
			return int(retValue.(types.Number))
		}
	}
	return 0
}

func PhotoIndexGetCountAtDate(dataset string, dateKey int) int {
	m := photoIndexGetByDateMap(dataset)
	if m != nil {
		if v, ok := m.MaybeGet(types.Number(dateKey)); ok {
			if s, ok := v.(types.Set); ok {
				return int(s.Len())
			}
		}
	}
	return 0
}

// byDate - Map<num, struct Photo>
// byTag - Map<string, Map<num, Set<struct Photo>>>
/*
	sizeType := types.MakeStructTypeFromFields("", types.FieldMap{
		"width":  types.NumberType,
		"height": types.NumberType,
	})
	dateType := types.MakeStructTypeFromFields("Date", types.FieldMap{
		"nsSinceEpoch": types.NumberType,
	})
	faceType := types.MakeStructTypeFromFields("", types.FieldMap{
		"name": types.StringType,
		"x":    types.NumberType,
		"y":    types.NumberType,
		"w":    types.NumberType,
		"h":    types.NumberType,
	})
	photoType := types.MakeStructTypeFromFields("Photo", types.FieldMap{
		"sizes":         types.MakeMapType(sizeType, types.StringType),
		"tags":          types.MakeSetType(types.StringType),
		"title":         types.StringType,
		"datePublished": dateType,
		"dateUpdated":   dateType,
	})
*/
