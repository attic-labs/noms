// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

var datasetCapturePrefixRe = regexp.MustCompile("^(" + datas.DatasetRe.String() + ")")

type AbsolutePath struct {
	Dataset string
	Hash    hash.Hash
	Path    types.Path
}

func NewAbsolutePath(str string) (AbsolutePath, error) {
	if len(str) == 0 {
		return AbsolutePath{}, errors.New("Empty path")
	}

	var h hash.Hash
	var dataset string
	var pathStr string

	if str[0] == '#' {
		tail := str[1:]
		if len(tail) < hash.StringLen {
			return AbsolutePath{}, errors.New("Invalid hash: " + tail)
		}

		hashStr := tail[:hash.StringLen]
		if h2, ok := hash.MaybeParse(hashStr); ok {
			h = h2
		} else {
			return AbsolutePath{}, errors.New("Invalid hash: " + hashStr)
		}

		pathStr = tail[hash.StringLen:]
	} else {
		datasetParts := datasetCapturePrefixRe.FindStringSubmatch(str)
		if datasetParts == nil {
			return AbsolutePath{}, fmt.Errorf("Invalid dataset name: %s", str)
		}

		dataset = datasetParts[1]
		pathStr = str[len(dataset):]
	}

	if len(pathStr) == 0 {
		return AbsolutePath{Hash: h, Dataset: dataset}, nil
	}

	path, err := types.ParsePath(pathStr)
	if err != nil {
		return AbsolutePath{}, err
	}

	return AbsolutePath{Hash: h, Dataset: dataset, Path: path}, nil
}

func (p AbsolutePath) Resolve(db datas.Database) (val types.Value) {
	if len(p.Dataset) > 0 {
		var ok bool
		ds := db.GetDataset(p.Dataset)
		if val, ok = ds.MaybeHead(); !ok {
			val = nil
		}
	} else if !p.Hash.IsEmpty() {
		val = db.ReadValue(p.Hash)
	} else {
		d.Chk.Fail("Unreachable")
	}

	if val != nil && p.Path != nil {
		val = p.Path.Resolve(val)
	}
	return
}

func (p AbsolutePath) String() (str string) {
	if len(p.Dataset) > 0 {
		str = p.Dataset
	} else if !p.Hash.IsEmpty() {
		str = "#" + p.Hash.String()
	} else {
		d.Chk.Fail("Unreachable")
	}

	return str + p.Path.String()
}

func ReadAbsolutePaths(db datas.Database, paths ...string) ([]types.Value, error) {
	r := make([]types.Value, 0, len(paths))
	for _, ps := range paths {
		p, err := NewAbsolutePath(ps)
		if err != nil {
			return nil, fmt.Errorf("Invalid input path '%s'", ps)
		}

		v := p.Resolve(db)
		if v == nil {
			return nil, fmt.Errorf("Input path '%s' does not exist in database", ps)
		}

		r = append(r, v)
	}
	return r, nil
}
