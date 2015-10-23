package newset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/attic-labs/noms/ref"
)

type chunkedSet struct {
	children entrySlice // sorted
}

func (set chunkedSet) Len() (length int) {
	for _, entry := range set.children {
		length += entry.set.Len()
	}
	return
}

func (set chunkedSet) First() ref.Ref {
	return set.children[0].start
}

func (set chunkedSet) Has(r ref.Ref) bool {
	searchIndex := sort.Search(len(set.children), func(i int) bool {
		return ref.Greater(set.children[i].start, r)
	})
	if searchIndex == 0 {
		return false
	}
	searchIndex--
	return set.children[searchIndex].set.Has(r)
}

func (set chunkedSet) Ref() ref.Ref {
	h := ref.NewHash()
	for _, entry := range set.children {
		h.Write(entry.set.Ref().DigestSlice())
	}
	return ref.FromHash(h)
}

func (set chunkedSet) Fmt(indent int) string {
	indentStr := strings.Repeat(" ", indent)
	if len(set.children) == 0 {
		return fmt.Sprintf("%s(empty chunked set)", indentStr)
	}
	s := fmt.Sprintf("%s(chunked with %d chunks)\n", indentStr, len(set.children))
	for i, entry := range set.children {
		s += fmt.Sprintf("%schunk %d (start %s)\n%s\n", indentStr, i, fmtRef(entry.start), entry.set.Fmt(indent+4))
	}
	return s
}
