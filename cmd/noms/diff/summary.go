package diff

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/status"
	"github.com/dustin/go-humanize"
)

// Summary prints a summary of the diff between two values to stdout.
func Summary(value1, value2 types.Value) {
	if datas.IsCommitType(value1.Type()) && datas.IsCommitType(value2.Type()) {
		fmt.Println("Commits detected. Comparing values instead.")
		value1 = value1.(types.Struct).Get(datas.ValueField)
		value2 = value2.(types.Struct).Get(datas.ValueField)
	}

	var valueString string
	if value1.Type().Kind() == value2.Type().Kind() {
		switch value1.Type().Kind() {
		case types.StructKind:
			valueString = "field"
		case types.MapKind:
			valueString = "entry"
		default:
			valueString = "value"
		}
	}
	waitChan := make(chan struct{})
	ch := make(chan diffSummaryProgress)
	go func() {
		acc := diffSummaryProgress{}
		for p := range ch {
			acc.Adds += p.Adds
			acc.Removes += p.Removes
			acc.Changes += p.Changes
			acc.NewSize += p.NewSize
			acc.OldSize += p.OldSize
			formatStatus(acc, valueString)
		}
		status.Done()
		waitChan <- struct{}{}
	}()
	diffSummary(ch, value1, value2)

	<-waitChan
}

type diffSummaryProgress struct {
	Adds, Removes, Changes, NewSize, OldSize uint64
}

func diffSummary(ch chan diffSummaryProgress, v1, v2 types.Value) {
	if !v1.Equals(v2) {
		if shouldDescend(v1, v2) {
			switch v1.Type().Kind() {
			case types.ListKind:
				diffSummaryList(ch, v1.(types.List), v2.(types.List))
			case types.MapKind:
				diffSummaryMap(ch, v1.(types.Map), v2.(types.Map))
			case types.SetKind:
				diffSummarySet(ch, v1.(types.Set), v2.(types.Set))
			case types.StructKind:
				diffSummaryStructs(ch, v1.(types.Struct), v2.(types.Struct))
			default:
				panic("Unrecognized type in diff function: " + v1.Type().Describe() + " and " + v2.Type().Describe())
			}
		} else {
			ch <- diffSummaryProgress{Adds: 1, Removes: 1, NewSize: 1, OldSize: 1}
			close(ch)
		}
	}
}

func diffSummaryList(ch chan<- diffSummaryProgress, v1, v2 types.List) {
	ch <- diffSummaryProgress{OldSize: v1.Len(), NewSize: v2.Len()}

	spliceChan := make(chan types.Splice)
	closeChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		v2.Diff(v1, spliceChan, closeChan)
		close(spliceChan)
		doneChan <- struct{}{}
	}()
	defer waitForCloseOrDone(closeChan, doneChan) // see comment for explanation

	for splice := range spliceChan {
		if splice.SpRemoved == splice.SpAdded {
			ch <- diffSummaryProgress{Changes: splice.SpRemoved}
		} else {
			ch <- diffSummaryProgress{Adds: splice.SpAdded, Removes: splice.SpRemoved}
		}
	}
	close(ch)
}

func diffSummaryMap(ch chan<- diffSummaryProgress, v1, v2 types.Map) {
	diffSummaryGeneric(ch, v1.Len(), v2.Len(), func(changeChan chan<- types.ValueChanged, closeChan <-chan struct{}) {
		v2.Diff(v1, changeChan, closeChan)
	})
}

func diffSummarySet(ch chan<- diffSummaryProgress, v1, v2 types.Set) {
	diffSummaryGeneric(ch, v1.Len(), v2.Len(), func(changeChan chan<- types.ValueChanged, closeChan <-chan struct{}) {
		v2.Diff(v1, changeChan, closeChan)
	})
}

func diffSummaryStructs(ch chan<- diffSummaryProgress, v1, v2 types.Struct) {
	size1 := uint64(v1.Type().Desc.(types.StructDesc).Len())
	size2 := uint64(v2.Type().Desc.(types.StructDesc).Len())
	diffSummaryGeneric(ch, size1, size2, func(changeChan chan<- types.ValueChanged, closeChan <-chan struct{}) {
		v2.Diff(v1, changeChan, closeChan)
	})
}

type diffFunc func(changeChan chan<- types.ValueChanged, closeChan <-chan struct{})

func diffSummaryGeneric(ch chan<- diffSummaryProgress, oldSize, newSize uint64, f diffFunc) {
	ch <- diffSummaryProgress{OldSize: oldSize, NewSize: newSize}

	changeChan := make(chan types.ValueChanged)
	closeChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		f(changeChan, closeChan)
		close(changeChan)
		doneChan <- struct{}{}
	}()
	defer waitForCloseOrDone(closeChan, doneChan) // see comment for explanation
	reportChanges(ch, changeChan)
	close(ch)
}

func reportChanges(ch chan<- diffSummaryProgress, changeChan chan types.ValueChanged) {
	for change := range changeChan {
		switch change.ChangeType {
		case types.DiffChangeAdded:
			ch <- diffSummaryProgress{Adds: 1}
		case types.DiffChangeRemoved:
			ch <- diffSummaryProgress{Removes: 1}
		case types.DiffChangeModified:
			ch <- diffSummaryProgress{Changes: 1}
		default:
			panic("unknown change type")
		}
	}
}

func formatStatus(acc diffSummaryProgress, valueString string) {
	pluralize := func(noun string, n uint64) string {
		// This only hanldes what we care about.
		if noun[len(noun)-1] == 'y' {
			noun = noun[:len(noun)-1] + "ie"
		}
		pattern := "%s %s"
		if n != 1 {
			pattern += "s"
		}
		return fmt.Sprintf(pattern, humanize.Comma(int64(n)), noun)
	}

	insertions := pluralize("insertion", acc.Adds)
	deletions := pluralize("deletion", acc.Removes)
	changes := pluralize("change", acc.Changes)

	oldValues := pluralize(valueString, acc.OldSize)
	newValues := pluralize(valueString, acc.NewSize)

	status.Printf("%s, %s, %s, (%s vs %s)", insertions, deletions, changes, oldValues, newValues)
}
