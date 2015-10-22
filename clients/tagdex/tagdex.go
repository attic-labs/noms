package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	photo "github.com/attic-labs/noms/clients/gen/sha1_7f65b04b0f60c7c529f3c5b716ec87e5c09e4b73"
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

	out := NewMapOfStringToSetOfPhotoUnion()

	t0 := time.Now()
	numRefs := 0
	numPhotos := 0

	types.Some(inputDS.Head().Value().Ref(), store, func(f types.Future) (skip bool) {
		numRefs++

		u := NewPhotoUnion()
		tags := photo.NewSetOfString()
		v := f.Deref(store)

		if p, ok := v.(photo.Photo); ok {
			fmt.Println("Found ", p.Title())

			tags = p.Tags()
			u = u.SetPhoto(p)
			skip = true
		} else if r, ok := v.(photo.RemotePhoto); ok {
			fmt.Println("Found ", r.Title())

			tags = r.Tags()
			u = u.SetRemote(r)
			skip = true
		}

		if !tags.Empty() {
			numPhotos++
			fmt.Println("Indexing", v.Ref())

			tags.IterAll(func(item string) {
				s, _ := out.MaybeGet(item)
				out = out.Set(item, s.Insert(u))
				return
			})
		}

		return
	})

	_, ok = outputDS.Commit(out)
	d.Exp.True(ok, "Could not commit due to conflicting edit")

	fmt.Printf("Indexed %v photos from %v refs in %v\n", numPhotos, numRefs, time.Now().Sub(t0))
}
