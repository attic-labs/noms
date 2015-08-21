package main

import (
	"flag"
	"fmt"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

func main() {
	dsFlags := dataset.NewFlags()
	flag.Parse()

	ds := dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}

	lastVal := uint64(0)
	commit := ds.Head()
	if !commit.Equals(datas.EmptyCommit) {
		lastVal = uint64(commit.Value().(types.UInt64))
	}
	newVal := lastVal + 1
	for ok := false; !ok; ds, ok = attemptCommit(types.UInt64(newVal), ds) {
		continue
	}
	fmt.Println(newVal)
}

func attemptCommit(newValue types.Value, ds *dataset.Dataset) (*dataset.Dataset, bool) {
	newDs, ok := ds.Commit(
		datas.NewCommit().SetParents(ds.HeadAsSet()).SetValue(newValue))
	return &newDs, ok
}
