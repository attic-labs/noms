package main

import (
	"fmt"
	"strings"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/types"
)

type CommitIterator struct {
	db       datas.Database
	branches branchList
}

func NewCommitIterator(db datas.Database, commit types.Struct) *CommitIterator {
	cr := types.NewTypedRefFromValue(commit)
	return &CommitIterator{db: db, branches: branchList{branch{cr: cr, commit: commit}}}
}

func (iter *CommitIterator) Next() (LogNode, bool) {
	if iter.branches.IsEmpty() {
		return LogNode{}, false
	}

	cols := iter.branches.ColumnsWithMaxHeight()
	col := cols[0]
	colsToDelete := cols[1:]
	br := iter.branches[col]
	startingColCount := len(iter.branches)

	for _, colToDelete := range colsToDelete {
		iter.branches = iter.branches.Splice(colToDelete, 1)
	}

	branches := branchList{}
	parents := commitRefsFromSet(br.commit.Get(datas.ParentsField).(types.Set))
	for _, p := range parents {
		b := branch{cr: p, commit: iter.db.ReadValue(p.TargetRef()).(types.Struct)}
		branches = append(branches, b)
	}
	iter.branches = iter.branches.Splice(col, 1, branches...)

	newCols := []int{}
	for cnt := 1; cnt < len(parents); cnt++ {
		newCols = append(newCols, col+cnt)
	}

	cols = iter.branches.ColumnsWithMaxHeight()
	return LogNode{
		cr:               br.cr,
		commit:           br.commit,
		startingColCount: startingColCount,
		endingColCount:   len(iter.branches),
		col:              col,
		newCols:          newCols,
		foldedCols:       cols,
		lastCommit:       iter.branches.IsEmpty(),
	}, true
}

type LogNode struct {
	cr               types.Ref
	commit           types.Struct
	startingColCount int
	endingColCount   int
	col              int
	newCols          []int
	foldedCols       []int
	lastCommit       bool
}

func (n LogNode) String() string {
	return fmt.Sprintf("cr: %s, startingColCount: %d, endingColCount: %d, col: %d, newCols: %v, foldedCols: %v", n.cr.TargetRef(), n.startingColCount, n.endingColCount, n.col, n.newCols, n.foldedCols)
}

type branch struct {
	cr     types.Ref
	commit types.Struct
}

type branchList []branch

func (bl branchList) IsEmpty() bool {
	return len(bl) == 0
}

func (bl branchList) String() string {
	res := []string{}
	for _, b := range bl {
		res = append(res, b.cr.TargetRef().String())
	}
	return strings.Join(res, " ")
}

func (bl branchList) ColumnsWithMaxHeight() []int {
	maxHeight := uint64(0)
	var cr types.Ref
	cols := []int{}
	for i, b := range bl {
		if b.cr.Height() > maxHeight {
			maxHeight = b.cr.Height()
			cr = b.cr
			cols = []int{i}
		} else if b.cr.Height() == maxHeight && b.cr.Equals(cr) {
			cols = append(cols, i)
		}
	}
	return cols
}

func (bl branchList) Splice(start int, deleteCount int, branches ...branch) branchList {
	l := append(bl[:start], branches...)
	return append(l, bl[start+deleteCount:]...)
}

func commitRefsFromSet(set types.Set) []types.Ref {
	res := []types.Ref{}
	set.IterAll(func(v types.Value) {
		res = append(res, v.(types.Ref))
	})
	return res
}
