// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/attic-labs/noms/go/types"
)

type expr interface {
	ranges() vrangeslice
	dbgPrintTree(w io.Writer, level int)
}

type logExpr struct {
	op    boolop
	expr1 expr
	expr2 expr
}

type relExpr struct {
	idxPath string
	op      relop
	v1      types.Value
}

func (le logExpr) ranges() (ranges vrangeslice) {
	rslice1 := le.expr1.ranges()
	rslice2 := le.expr2.ranges()
	rslice := vrangeslice{}

	switch le.op {
	case and:
		if len(rslice1) == 0 || len(rslice2) == 0 {
			return rslice
		}
		for _, r1 := range rslice1 {
			for _, r2 := range rslice2 {
				rslice = append(rslice, r1.and(r2)...)
			}
		}
		sort.Sort(rslice)
		return rslice
	case or:
		if len(rslice1) == 0 {
			return rslice2
		}
		if len(rslice2) == 0 {
			return rslice1
		}
		for _, r1 := range rslice1 {
			for _, r2 := range rslice2 {
				rslice = append(rslice, r1.or(r2)...)
			}
		}
		sort.Sort(rslice)
		return rslice
	}
	return []vrange{}
}

func (le logExpr) dbgPrintTree(w io.Writer, level int) {
	fmt.Fprintf(w, "%*s%s\n", 2*level, "", le.op)
	if le.expr1 != nil {
		le.expr1.dbgPrintTree(w, level+1)
	}
	if le.expr2 != nil {
		le.expr2.dbgPrintTree(w, level+1)
	}
}

func (re relExpr) ranges() (ranges vrangeslice) {
	var r vrange
	switch re.op {
	case equals:
		e := ventry{v: re.v1, incl: true}
		r = vrange{lower: e, upper: e}
	case gt:
		r = vrange{lower: ventry{re.v1, false, 0}, upper: ventry{nil, true, 1}}
	case gte:
		r = vrange{lower: ventry{re.v1, true, 0}, upper: ventry{nil, true, 1}}
	case lt:
		r = vrange{lower: ventry{nil, true, -1}, upper: ventry{re.v1, false, 0}}
	case lte:
		r = vrange{lower: ventry{nil, true, -1}, upper: ventry{re.v1, true, 0}}
	}
	return vrangeslice{r}
}

func (re relExpr) dbgPrintTree(w io.Writer, level int) {
	buf := bytes.Buffer{}
	types.WriteEncodedValue(&buf, re.v1)
	fmt.Fprintf(w, "%*s%s %s %s\n", 2*level, "", re.idxPath, re.op, buf.String())
}
