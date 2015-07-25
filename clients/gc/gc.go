package main

import (
	"flag"
	"fmt"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/walk"
)

// Currently this has to be here because we're iterating over values to get all the refs. This can move
// into the chunk implementation when "walk" works over chunks.
func allValueRefs(root ref.Ref, cs *chunks.FileStore) map[ref.Ref]bool {
	m := map[ref.Ref]bool{}
	walk.Some(root, cs, func(r ref.Ref) bool {
		if m[r] {
			return true;
		} else {
			m[r] = true
			return false
		}
	})

	return m
}

func main() {
	fsFlags := chunks.NewFlags()
	flag.Parse()

	fs := fsFlags.CreateStore()
	if fs == nil {
		flag.Usage()
		return
	}

	root := fs.Root()
	cs := fs.(*chunks.FileStore)
	refs := allValueRefs(root, cs)
	numDeleted := cs.GarbageCollect(refs)
	fmt.Printf("Garbage collected %d chunks", numDeleted)
}