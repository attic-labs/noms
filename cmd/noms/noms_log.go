// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/attic-labs/noms/cmd/noms/diff"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/orderedparallel"
	"github.com/attic-labs/noms/go/util/outputpager"
	"github.com/mgutz/ansi"
	flag "github.com/tsuru/gnuflag"
)

var (
	useColor   = false
	color      int
	maxLines   int
	maxCommits int
	oneline    bool
	showGraph  bool
	showValue  bool
)

const parallelism = 16

var nomsLog = &nomsCommand{
	Run:       runLog,
	UsageLine: "log [options] <commitObject>",
	Short:     "Displays the history of a Noms dataset",
	Long:      "commitObject must be a dataset or object spec that refers to a commit. See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details.",
	Flags:     setupLogFlags,
	Nargs:     1,
}

func setupLogFlags() *flag.FlagSet {
	logFlagSet := flag.NewFlagSet("log", flag.ExitOnError)
	logFlagSet.IntVar(&color, "color", -1, "value of 1 forces color on, 2 forces color off")
	logFlagSet.IntVar(&maxLines, "max-lines", 10, "max number of lines to show per commit (-1 for all lines)")
	logFlagSet.IntVar(&maxCommits, "n", 0, "max number of commits to display (0 for all commits)")
	logFlagSet.BoolVar(&oneline, "oneline", false, "show a summary of each commit on a single line")
	logFlagSet.BoolVar(&showGraph, "graph", false, "show ascii-based commit hierarcy on left side of output")
	logFlagSet.BoolVar(&showValue, "show-value", false, "show commit value rather than diff information -- this is temporary")
	outputpager.RegisterOutputpagerFlags(logFlagSet)
	return logFlagSet
}

func runLog(args []string) int {
	useColor = shouldUseColor()

	database, value, err := spec.GetPath(args[0])
	if err != nil {
		d.CheckErrorNoUsage(err)
	}
	defer database.Close()

	if value == nil {
		d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", args[0]))
	}

	origCommit, ok := value.(types.Struct)
	if !ok || !datas.IsCommitType(origCommit.Type()) {
		d.CheckError(fmt.Errorf("%s does not reference a Commit object", args[0]))
	}

	iter := NewCommitIterator(database, origCommit)
	displayed := 0
	if maxCommits <= 0 {
		maxCommits = math.MaxInt32
	}

	inChan := make(chan interface{}, parallelism)
	outChan := orderedparallel.New(inChan, func(node interface{}) interface{} {
		buff := &bytes.Buffer{}
		printCommit(node.(LogNode), buff, database)
		return buff.Bytes()
	}, parallelism)

	go func() {
		for ln, ok := iter.Next(); ok && displayed < maxCommits; ln, ok = iter.Next() {
			inChan <- ln
			displayed++
		}
		close(inChan)
	}()

	pgr := outputpager.Start()
	defer pgr.Stop()

	for commitBuff := range outChan {
		io.Copy(pgr.Writer, bytes.NewReader(commitBuff.([]byte)))
	}
	return 0
}

// Prints the information for one commit in the log, including ascii graph on left side of commits if
// -graph arg is true.
func printCommit(node LogNode, w io.Writer, db datas.Database) (err error) {
	maxMetaFieldNameLength := func(commit types.Struct) int {
		maxLen := 0
		if m, ok := commit.MaybeGet(datas.MetaField); ok {
			meta := m.(types.Struct)
			meta.Type().Desc.(types.StructDesc).IterFields(func(name string, t *types.Type) {
				maxLen = max(maxLen, len(name))
			})
		}
		return maxLen
	}

	hashStr := node.commit.Hash().String()
	if useColor {
		hashStr = ansi.Color("commit "+hashStr, "red+h")
	}

	maxFieldNameLen := maxMetaFieldNameLength(node.commit)

	parentLabel := "Parent"
	parentValue := "None"
	parents := commitRefsFromSet(node.commit.Get(datas.ParentsField).(types.Set))
	if len(parents) > 1 {
		pstrings := make([]string, len(parents))
		for i, p := range parents {
			pstrings[i] = p.TargetHash().String()
		}
		parentLabel = "Merge"
		parentValue = strings.Join(pstrings, " ")
	} else if len(parents) == 1 {
		parentValue = parents[0].TargetHash().String()
	}

	if oneline {
		parentStr := fmt.Sprintf("%s %s", parentLabel+":", parentValue)
		fmt.Fprintf(w, "%s (%s)\n", hashStr, parentStr)
		return
	}

	maxFieldNameLen = max(maxFieldNameLen, len(parentLabel))
	parentStr := fmt.Sprintf("%-*s %s", maxFieldNameLen+1, parentLabel+":", parentValue)
	fmt.Fprintf(w, "%s%s\n", genGraph(node, 0), hashStr)
	fmt.Fprintf(w, "%s%s\n", genGraph(node, 1), parentStr)
	lineno := 1

	if maxLines != 0 {
		lineno, err = writeMetaLines(node, maxLines, lineno, maxFieldNameLen, w)
		if err != nil && err != MaxLinesErr {
			fmt.Fprintf(w, "error: %s\n", err)
			return
		}

		if showValue {
			_, err = writeCommitLines(node, maxLines, lineno, w)
		} else {
			_, err = writeDiffLines(node, db, maxLines, lineno, w)
		}
	}
	return
}

// Generates ascii graph chars to display on the left side of the commit info if -graph arg is true.
func genGraph(node LogNode, lineno int) string {
	if !showGraph {
		return ""
	}

	// branchCount is the number of branches that we need to graph for this commit and determines the
	// length of prefix string. The string will change from line to line to indicate whether the new
	// branches are getting created or currently displayed branches need to be merged with other branches.
	// Normally we want the maximum number of branches so we have enough room to display them all, however
	// if node.Shrunk() is true, we only need to display the minimum number of branches.
	branchCount := max(node.startingColCount, node.endingColCount)
	if node.Shrunk() {
		branchCount = min(node.startingColCount, node.endingColCount)
	}

	// Create the basic prefix string indicating the number of branches that are being tracked.
	p := strings.Repeat("| ", max(branchCount, 1))
	buf := []rune(p)

	// The first line of a commit has a '*' in the graph to indicate what branch it resides in.
	if lineno == 0 {
		if node.Expanding() {
			buf[(branchCount-1)*2] = ' '
		}
		buf[node.col*2] = '*'
		return string(buf)
	}

	// If expanding, change all the '|' chars to '\' chars after the inserted branch
	if node.Expanding() && lineno == 1 {
		for i := node.newCols[0]; i < branchCount; i++ {
			buf[(i*2)-1] = '\\'
			buf[i*2] = ' '
		}
	}

	// if one branch is getting folded into another, show '/' where necessary to indicate that.
	if node.Shrinking() {
		foldingDistance := node.foldedCols[1] - node.foldedCols[0]
		ch := ' '
		if lineno < foldingDistance+1 {
			ch = '/'
		}
		for _, col := range node.foldedCols[1:] {
			buf[(col*2)-1] = ch
			buf[(col * 2)] = ' '
		}
	}

	return string(buf)
}

func writeMetaLines(node LogNode, maxLines, lineno, maxLabelLen int, w io.Writer) (int, error) {
	if m, ok := node.commit.MaybeGet(datas.MetaField); ok {
		meta := m.(types.Struct)
		mlw := &maxLineWriter{numLines: lineno, maxLines: maxLines, node: node, dest: w, needsPrefix: true, showGraph: showGraph}
		err := d.Try(func() {
			meta.Type().Desc.(types.StructDesc).IterFields(func(fieldName string, t *types.Type) {
				v := meta.Get(fieldName)
				fmt.Fprintf(mlw, "%-*s", maxLabelLen+2, strings.Title(fieldName)+":")
				types.WriteEncodedValue(mlw, v)
				fmt.Fprintf(mlw, "\n")
			})
		})
		return mlw.numLines, err
	}
	return lineno, nil
}

func writeCommitLines(node LogNode, maxLines, lineno int, w io.Writer) (lineCnt int, err error) {
	mlw := &maxLineWriter{numLines: lineno, maxLines: maxLines, node: node, dest: w, needsPrefix: true, showGraph: showGraph}
	err = types.WriteEncodedValueWithTags(mlw, node.commit.Get(datas.ValueField))
	d.PanicIfNotType(err, MaxLinesErr)
	if err != nil {
		mlw.forceWrite([]byte("..."))
		mlw.numLines++
		err = nil
	}
	mlw.forceWrite([]byte("\n"))
	if !node.lastCommit {
		mlw.forceWrite([]byte("\n"))
	}
	return mlw.numLines, err
}

func writeDiffLines(node LogNode, db datas.Database, maxLines, lineno int, w io.Writer) (lineCnt int, err error) {
	mlw := &maxLineWriter{numLines: lineno, maxLines: maxLines, node: node, dest: w, needsPrefix: true, showGraph: showGraph}
	parents := node.commit.Get(datas.ParentsField).(types.Set)
	var parent types.Value = nil
	if parents.Len() > 0 {
		parent = parents.First()
	}
	if parent == nil {
		_, err = fmt.Fprint(mlw, "\n")
		return 1, err
	}

	parentCommit := parent.(types.Ref).TargetValue(db).(types.Struct)
	err = diff.Diff(mlw, parentCommit.Get(datas.ValueField), node.commit.Get(datas.ValueField))
	d.PanicIfNotType(err, MaxLinesErr)
	if err != nil {
		mlw.forceWrite([]byte("...\n"))
		mlw.numLines++
		err = nil
	}
	if !node.lastCommit {
		mlw.forceWrite([]byte("\n"))
	}
	return mlw.numLines, err
}

func shouldUseColor() bool {
	if color != 1 && color != 0 {
		return outputpager.IsStdoutTty()
	}
	return color == 1
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
