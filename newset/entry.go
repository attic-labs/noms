package newset

import "github.com/attic-labs/noms/ref"

type entry struct {
	start ref.Ref
	set   Set
}

type entrySlice []entry

func (es entrySlice) Len() int {
	return len(es)
}

func (es entrySlice) Less(i, j int) bool {
	return ref.Less(es[i].start, es[j].start)
}

func (es entrySlice) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}
