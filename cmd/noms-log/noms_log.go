package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/attic-labs/noms/clients/go/flags"
	"github.com/attic-labs/noms/clients/go/util"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/types"
	goisatty "github.com/mattn/go-isatty"
	"github.com/mgutz/ansi"
)

var (
	forceColor = flag.Bool("force-color", false, "respect no-color arg regardless of whether printing to terminal")
	lines      = flag.Int("lines", 10, "max number of lines to show per commit (-1 for all lines)")
	noColor    = flag.Bool("no-color", false, "don't use color in output")
	showHelp   = flag.Bool("help", false, "show help text")
	showGraph  = flag.Bool("graph", false, "show ascii-based commit hierarcy on left side of output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <dataset>\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSee \"Spelling Objects\" at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object argument.\n\n")
	}

	flag.Parse()
	if *showHelp {
		flag.Usage()
		return
	}

	if len(flag.Args()) != 1 {
		util.CheckError(errors.New("expected exactly one argument"))
	}

	spec, err := flags.ParseDatasetSpec(flag.Arg(0))
	util.CheckError(err)
	dataset, err := spec.Dataset()
	util.CheckError(err)

	origCommit, ok := dataset.MaybeHead()

	if ok {
		iter := NewCommitIterator(dataset.Store(), origCommit)
		for ln, ok := iter.Next(); ok; ln, ok = iter.Next() {
			printCommit(ln)
		}
	}

	dataset.Store().Close()
}

func useColor() bool {
	if *noColor {
		return false
	}
	if *forceColor {
		return !*noColor
	}
	return goisatty.IsTerminal(1)
}

func printCommit(ln LogNode) {
	lineno := 0
	doColor := func(s string) string { return s }
	if useColor() {
		doColor = ansi.ColorFunc("red+h")
	}

	fmt.Printf("%s%s\n", genPrefix(ln, lineno), doColor(ln.commit.Ref().String()))
	parents := commitRefsFromSet(ln.commit.Get(datas.ParentsField).(types.Set))
	lineno++
	if len(parents) > 1 {
		pstrings := []string{}
		for _, cr := range parents {
			pstrings = append(pstrings, cr.TargetRef().String())
		}
		fmt.Printf("%sMerge: %s\n", genPrefix(ln, lineno), strings.Join(pstrings, " "))
	} else if len(parents) == 1 {
		fmt.Printf("%sParent: %s\n", genPrefix(ln, lineno), parents[0].TargetRef().String())
	} else {
		fmt.Printf("%sParent: None\n", genPrefix(ln, lineno))
	}
	lines := truncateLines(types.EncodedValueWithTags(ln.commit.Get(datas.ValueField)), *lines)
	for _, line := range lines {
		lineno++
		fmt.Printf("%s%s\n", genPrefix(ln, lineno), line)
	}
	lineno++
	if !ln.lastCommit {
		fmt.Printf("%s\n", genPrefix(ln, lineno))
	}
}

func genPrefix(ln LogNode, lineno int) string {
	if !*showGraph {
		return ""
	}
	expanding := ln.startingColCount < ln.endingColCount
	shrunk := ln.startingColCount > ln.endingColCount
	shrinking := len(ln.foldedCols) > 1

	maxColCount := max(ln.startingColCount, ln.endingColCount)
	minColCount := min(ln.startingColCount, ln.endingColCount)
	colCount := maxColCount
	if shrunk {
		colCount = minColCount
	}

	p := strings.Repeat("| ", max(colCount, 1))
	buf := []rune(p)

	if lineno == 0 {
		if expanding {
			buf[(colCount-1)*2] = ' '
		}
		buf[ln.col*2] = '*'
		return string(buf)
	}

	if expanding && lineno == 1 {
		for i := ln.newCols[0]; i < colCount; i++ {
			buf[(i*2)-1] = '\\'
			buf[i*2] = ' '
		}
	}

	if shrinking {
		foldingDistance := ln.foldedCols[1] - ln.foldedCols[0]
		ch := ' '
		if lineno < foldingDistance+1 {
			ch = '/'
		}
		for _, col := range ln.foldedCols[1:] {
			buf[(col*2)-1] = ch
			buf[(col * 2)] = ' '
		}
	}
	return string(buf)
}

func truncateLines(s1 string, maxLines int) []string {
	s1 = strings.TrimSpace(s1)
	var res = []string{}
	switch {
	case maxLines == 0:
	case maxLines < 0:
		res = strings.Split(s1, "\n")
	default:
		x := strings.SplitN(s1, "\n", maxLines+1)
		if len(x) < maxLines {
			maxLines = len(x)
		}
		res = x[:maxLines]
	}
	return res
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
