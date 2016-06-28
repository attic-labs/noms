// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
)

var datasetCapturePrefixRe = regexp.MustCompile("^(" + constants.DatasetRe.String() + ")")

type Path struct {
	rootDataset string
	rootHash    hash.Hash
	parts       []pathPart
}

type PathRootGetter interface {
	GetDatasetHead(id string) Value
	GetHash(h hash.Hash) Value
}

type pathPart interface {
	Resolve(v Value) Value
	String() string
}

func NewPath() Path {
	return Path{}
}

func ParsePath(path string) (Path, error) {
	return NewPath().AddPath(path)
}

func (p Path) SetRootDataset(id string) Path {
	d.Chk.False(p.HasRoot(), "Path already has a root")
	return Path{rootDataset: id, parts: p.copyParts(0)}
}

func (p Path) SetRootHash(h hash.Hash) Path {
	d.Chk.False(p.HasRoot(), "Path already has a root")
	return Path{rootHash: h, parts: p.copyParts(0)}
}

func (p Path) HasRoot() bool {
	return len(p.rootDataset) > 0 || !p.rootHash.IsEmpty()
}

func (p Path) AddField(name string) Path {
	return p.appendPart(newFieldPart(name))
}

func (p Path) AddIndex(idx Value) Path {
	return p.appendPart(newIndexPart(idx))
}

func (p Path) AddHashIndex(h hash.Hash) Path {
	return p.appendPart(newHashIndexPart(h))
}

func (p Path) appendPart(part pathPart) Path {
	return Path{p.rootDataset, p.rootHash, append(p.copyParts(1), part)}
}

func (p Path) copyParts(n int) []pathPart {
	parts := make([]pathPart, len(p.parts), len(p.parts)+n)
	copy(parts, p.parts)
	return parts
}

func (p Path) AddPath(str string) (Path, error) {
	if len(str) == 0 {
		return Path{}, errors.New("Empty path")
	}

	return p.addPath(str, true)
}

func (p Path) addPath(str string, isRoot bool) (Path, error) {
	if len(str) == 0 {
		return p, nil
	}

	op, tail := str[0], str[1:]

	switch op {
	case '#':
		if !isRoot {
			return Path{}, errors.New("# operator can only be the first character")
		}

		if len(tail) < hash.StringLen {
			return Path{}, errors.New("Invalid hash: " + tail)
		}

		hashstr := tail[:hash.StringLen]
		h, ok := hash.MaybeParse(hashstr)
		if !ok {
			return Path{}, errors.New("Invalid hash: " + hashstr)
		}

		return p.SetRootHash(h).addPath(tail[hash.StringLen:], false)

	case '.':
		idx := fieldNameComponentRe.FindIndex([]byte(tail))
		if idx == nil {
			return Path{}, errors.New("Invalid field: " + tail)
		}

		return p.AddField(tail[:idx[1]]).addPath(tail[idx[1]:], false)

	case '[':
		if len(tail) == 0 {
			return Path{}, errors.New("Path ends in [")
		}

		idx, h, rem, err := parsePathIndex(tail)
		if err != nil {
			return Path{}, err
		}

		d.Chk.NotEqual(idx == nil, h.IsEmpty())
		if idx != nil {
			return p.AddIndex(idx).addPath(rem, false)
		} else {
			return p.AddHashIndex(h).addPath(rem, false)
		}

	case ']':
		return Path{}, errors.New("] is missing opening [")

	default:
		// Operator isn't recognised, try to parse the whole string as a dataset root.
		if !isRoot {
			return Path{}, fmt.Errorf("Invalid operator: %c", op)
		}

		datasetIdParts := datasetCapturePrefixRe.FindStringSubmatch(str)
		if datasetIdParts == nil {
			return Path{}, fmt.Errorf("Invalid dataset name: %s", str)
		}

		datasetId := datasetIdParts[1]
		return p.SetRootDataset(datasetId).addPath(str[len(datasetId):], false)
	}
}

func (p Path) Resolve(v Value) (resolved Value) {
	resolved = v
	for _, part := range p.parts {
		if resolved == nil {
			break
		}
		resolved = part.Resolve(resolved)
	}

	return
}

func (p Path) ResolveFromRoot(getter PathRootGetter) (val Value) {
	if len(p.rootDataset) > 0 {
		val = getter.GetDatasetHead(p.rootDataset)
	} else if !p.rootHash.IsEmpty() {
		val = getter.GetHash(p.rootHash)
	} else {
		d.Chk.Fail("Path does not have a root")
	}

	if val != nil {
		val = p.Resolve(val)
	}
	return
}

func (p Path) String() string {
	nparts := len(p.parts)
	if p.HasRoot() {
		nparts++
	}

	strs := make([]string, 0, nparts)

	if len(p.rootDataset) > 0 {
		strs = append(strs, p.rootDataset)
	} else if !p.rootHash.IsEmpty() {
		strs = append(strs, "#"+p.rootHash.String())
	}

	for _, part := range p.parts {
		strs = append(strs, part.String())
	}

	return strings.Join(strs, "")
}

type fieldPart struct {
	name string
}

func newFieldPart(name string) fieldPart {
	return fieldPart{name}
}

func (fp fieldPart) Resolve(v Value) Value {
	if s, ok := v.(Struct); ok {
		if fv, ok := s.MaybeGet(fp.name); ok {
			return fv
		}
	}

	return nil
}

func (fp fieldPart) String() string {
	return fmt.Sprintf(".%s", fp.name)
}

type indexPart struct {
	idx Value
}

func newIndexPart(idx Value) indexPart {
	k := idx.Type().Kind()
	d.Chk.True(k == StringKind || k == BoolKind || k == NumberKind)
	return indexPart{idx}
}

func (ip indexPart) Resolve(v Value) Value {
	if l, ok := v.(List); ok {
		if n, ok := ip.idx.(Number); ok {
			f := float64(n)
			if f == math.Trunc(f) && f >= 0 {
				u := uint64(f)
				if u < l.Len() {
					return l.Get(u)
				}
			}
		}
	}

	if m, ok := v.(Map); ok {
		return m.Get(ip.idx)
	}

	return nil
}

func (ip indexPart) String() string {
	return fmt.Sprintf("[%s]", EncodedValue(ip.idx))
}

type hashIndexPart struct {
	h hash.Hash
}

func newHashIndexPart(h hash.Hash) hashIndexPart {
	return hashIndexPart{h}
}

func (hip hashIndexPart) Resolve(v Value) (res Value) {
	var seq orderedSequence
	var getCurrentValue func(cur *sequenceCursor) Value

	switch v := v.(type) {
	case Set:
		seq = v.seq
		getCurrentValue = func(cur *sequenceCursor) Value { return cur.current().(Value) }
	case Map:
		seq = v.seq
		getCurrentValue = func(cur *sequenceCursor) Value { return cur.current().(mapEntry).value }
	default:
		return nil
	}

	cur := newCursorAt(seq, orderedKeyFromHash(hip.h), false, false)
	if !cur.valid() {
		return nil
	}

	if getCurrentKey(cur).h != hip.h {
		return nil
	}

	return getCurrentValue(cur)
}

func (hip hashIndexPart) String() string {
	return fmt.Sprintf("[#%s]", hip.h.String())
}

func parsePathIndex(str string) (idx Value, h hash.Hash, rem string, err error) {
Switch:
	switch str[0] {
	case '"':
		// String is complicated because ] might be quoted, and " or \ might be escaped.
		stringBuf := bytes.Buffer{}
		i := 1

		for ; i < len(str); i++ {
			c := str[i]
			if c == '"' {
				break
			}
			if c == '\\' && i < len(str)-1 {
				i++
				c = str[i]
				if c != '\\' && c != '"' {
					err = errors.New(`Only " and \ can be escaped`)
					break Switch
				}
			}
			stringBuf.WriteByte(c)
		}

		if i == len(str) {
			err = errors.New("[ is missing closing ]")
		} else {
			idx = String(stringBuf.String())
			rem = str[i+2:]
		}

	default:
		split := strings.SplitN(str, "]", 2)
		if len(split) < 2 {
			err = errors.New("[ is missing closing ]")
			break Switch
		}

		idxStr := split[0]
		rem = split[1]

		if len(idxStr) == 0 {
			err = errors.New("Empty index value")
		} else if idxStr[0] == '#' {
			hashStr := idxStr[1:]
			h, _ = hash.MaybeParse(hashStr)
			if h.IsEmpty() {
				err = errors.New("Invalid hash: " + hashStr)
			}
		} else if idxStr == "true" {
			idx = Bool(true)
		} else if idxStr == "false" {
			idx = Bool(false)
		} else if i, err2 := strconv.ParseFloat(idxStr, 64); err2 == nil {
			// Should we be more strict here? ParseFloat allows leading and trailing dots, and exponents.
			idx = Number(i)
		} else {
			err = errors.New("Invalid index: " + idxStr)
		}
	}

	return
}
