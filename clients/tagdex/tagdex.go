package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	photos "github.com/attic-labs/noms/clients/gen/sha1_ee6ba8b7a1135a4360459b053b68bf5f992bb23e"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

var (
	flags    = datas.NewFlags()
	inputID  = flag.String("input-ds", "", "dataset to find photos within")
	outputID = flag.String("output-ds", "", "dataset to store index in")
)

func main() {
	flag.Parse()

	store, ok := flags.CreateDataStore()
	if !ok || *inputID == "" || *outputID == "" {
		flag.Usage()
		return
	}
	defer store.Close()

	inputDS := dataset.NewDataset(store, *inputID)
	if _, ok := inputDS.MaybeHead(); !ok {
		log.Fatalf("No dataset named %s", *inputID)
	}
	outputDS := dataset.NewDataset(store, *outputID)

	out := NewMapOfStringToSetOfValue()

	t0 := time.Now()
	numRefs := 0
	numPhotos := 0

	photoTypeRef := photos.NewPhoto().TypeRef()
	remotePhotoTypeRef := photos.NewRemotePhoto().TypeRef()

	types.Some(inputDS.Head().Value().Ref(), store, func(f types.Future) (skip bool) {
		numRefs++
		v := f.Deref(store)
		tags := photos.NewSetOfString()
		if v.TypeRef().Equals(photoTypeRef) {
			tags = photos.PhotoFromVal(v).Tags()
			skip = true
		} else if v.TypeRef().Equals(remotePhotoTypeRef) {
			tags = photos.RemotePhotoFromVal(v).Tags()
			skip = true
		}

		if !tags.Empty() {
			numPhotos++
			fmt.Println("Indexing", v.Ref())

			tags.IterAll(func(item string) {
				s := NewSetOfValue()
				if out.Has(item) {
					s = out.Get(item)
				}
				out = out.Set(item, s.Insert(v))
			})
		}
		return
	})

	_, ok = outputDS.Commit(out.NomsValue())
	d.Exp.True(ok, "Could not commit due to conflicting edit")

	fmt.Printf("Indexed %v photos from %v refs in %v\n", numPhotos, numRefs, time.Now().Sub(t0))
}
