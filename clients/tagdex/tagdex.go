package main

import (
	"flag"
	"fmt"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var (
	csFlags     = chunks.NewFlags()
	inputRefStr = flag.String("input-ref", "", "ref to find photos from within input chunkstore")
	outputID    = flag.String("output-ds", "", "dataset to store data in.")
)

func main() {
	flag.Parse()

	cs := csFlags.CreateStore()
	if cs == nil || *inputRefStr == "" || *outputID == "" {
		flag.Usage()
		return
	}

	var inputRef ref.Ref
	err := d.Try(func() {
		inputRef = ref.Parse(*inputRefStr)
	})
	if err != nil {
		flag.Usage()
		return
	}

	ds := dataset.NewDataset(datas.NewDataStore(cs), *outputID)
	out := NewMapOfStringToSetOfPhoto()

	types.All(inputRef, cs, func(f types.Future) {
		v, err := f.Deref(cs)
		d.Chk.NoError(err)
		if v, ok := v.(types.Map); ok && types.NewString("Photo").Equals(v.Get(types.NewString("$name"))) {
			p := PhotoFromVal(v)
			p.Tags().Iter(func(item types.String) (stop bool) {
				var s SetOfPhoto
				if out.Has(item) {
					s = out.Get(item)
				} else {
					s = NewSetOfPhoto()
				}
				out = out.Set(item, s.Insert(p))
				return
			})
		}
	})

	ds = ds.Commit(datas.NewSetOfCommit().Insert(
		datas.NewCommit().SetParents(ds.Heads().NomsValue()).SetValue(out.NomsValue())))

	fmt.Println(ds.Root().String())
}
