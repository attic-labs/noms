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

type ventry struct {
	v    types.Value
	incl bool
	inf  int8
}

type vrange struct {
	lower ventry
	upper ventry
}

type vrangeslice []vrange

func (vs vrangeslice) Len() int {
	return len(vs)
}

func (vs vrangeslice) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs vrangeslice) Less(i, j int) bool {
	return !vs[i].lower.equals(vs[j].lower) && vs[i].lower.isLessThanOrEqual(vs[j].lower)
}

func (vs vrangeslice) dbgPrint(w io.Writer) {
	for i, r := range vs {
		if i == 0 {
			fmt.Fprintf(w, "\n#################\n")
		}
		fmt.Fprintf(w, "range %d: %s\n", i, r)
	}
	if len(vs) > 0 {
		fmt.Fprintf(w, "\n")
	}
}
func (r vrange) and(o vrange) (vslice vrangeslice) {
	if !r.intersects(o) {
		return []vrange{}
	}

	lower := r.lower.maxValue(o.lower)
	upper := r.upper.minValue(o.upper)
	return []vrange{vrange{lower, upper}}
}

func (v vrange) or(o vrange) (vslice vrangeslice) {
	if v.intersects(o) {
		v1 := v.lower.minValue(o.lower)
		v2 := v.upper.maxValue(o.upper)
		return []vrange{vrange{v1, v2}}
	}
	res := vrangeslice{v, o}
	sort.Sort(res)
	return res
}

func (r vrange) intersects(o vrange) (res bool) {
	if r.lower.isGreaterThanOrEqual(o.lower) && r.lower.isLessThanOrEqual(o.upper) {
		return true
	}
	if r.upper.isGreaterThanOrEqual(o.lower) && r.upper.isLessThanOrEqual(o.upper) {
		return true
	}
	if o.lower.isGreaterThanOrEqual(r.lower) && o.lower.isLessThanOrEqual(r.upper) {
		return true
	}
	if o.upper.isGreaterThanOrEqual(r.lower) && o.upper.isLessThanOrEqual(r.upper) {
		return true
	}
	return false
}

func (v ventry) isLessThanOrEqual(o ventry) (res bool) {
	return v.equals(o) || v.isLessThan(o)
}

//
func (v ventry) isLessThan(o ventry) (res bool) {
	if v.equals(o) {
		return false
	}

	if v.inf < o.inf {
		return true
	}

	if v.inf > o.inf {
		return false
	}

	if v.v.Less(o.v) {
		return true
	}

	if v.v.Equals(o.v) {
		return o.incl
	}
	return false
}

func (v ventry) isGreaterThanOrEqual(o ventry) (res bool) {
	return !v.isLessThan(o)
}

func (v ventry) isGreaterThan(o ventry) (res bool) {
	return !v.equals(o) || !v.isLessThan(o)
}

func (v ventry) equals(o ventry) bool {
	return v.inf == o.inf && v.incl == o.incl &&
		(v.v == nil && o.v == nil || (v.v != nil && o.v != nil && v.v.Equals(o.v)))
}

func (v vrange) String() string {
	return fmt.Sprintf("vrange{lower: %s, upper: %s", v.lower, v.upper)
}

func (v ventry) String() string {
	var s1 string
	if v.v == nil {
		s1 = "<nil>"
	} else {
		buf := bytes.Buffer{}
		types.WriteEncodedValue(&buf, v.v)
		s1 = buf.String()
	}
	return fmt.Sprintf("ventry{v: %s, incl: %t, inf: %d}", s1, v.incl, v.inf)
}

func (v ventry) minValue(o ventry) (res ventry) {
	if v.isLessThan(o) {
		return v
	}
	return o
}

func (v ventry) maxValue(o ventry) (res ventry) {
	if v.isLessThan(o) {
		return o
	}
	return v
}
