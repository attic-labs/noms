// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

var (
	databaseRegex = regexp.MustCompile("^([^:]+)(:.+)?$")
	datasetRegex  = regexp.MustCompile(`^([a-zA-Z0-9\-_/]+)`)
)

func GetDatabase(str string) (datas.Database, error) {
	sp, err := parseDatabaseSpec(str)
	if err != nil {
		return nil, err
	}
	return sp.Database()
}

func GetChunkStore(str string) (chunks.ChunkStore, error) {
	sp, err := parseDatabaseSpec(str)
	if err != nil {
		return nil, err
	}

	switch sp.Protocol {
	case "ldb":
		return chunks.NewLevelDBStoreUseFlags(sp.Path, ""), nil
	case "mem":
		return chunks.NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("Unable to create chunkstore for protocol: %s", str)
	}
}

func GetDataset(str string) (dataset.Dataset, error) {
	sp, rem, err := parseDatasetSpec(str)
	if err != nil {
		return dataset.Dataset{}, err
	}
	if len(rem) > 0 {
		return dataset.Dataset{}, fmt.Errorf("Dataset %s has trailing characters %s", str, rem)
	}
	return sp.Dataset()
}

func GetValue(str string) (datas.Database, types.Value, error) {
	sp, err := parseValueSpec(str)
	if err != nil {
		return nil, nil, err
	}
	return sp.Value()
}

type databaseSpec struct {
	Protocol    string
	Path        string
	accessToken string
}

type datasetSpec struct {
	DbSpec      databaseSpec
	DatasetName string
}

type hashSpec struct {
	DbSpec databaseSpec
	Hash   hash.Hash
}

type pathSpec struct {
	Rel  valueSpec
	Path types.Path
}

type valueSpec interface {
	Value() (datas.Database, types.Value, error)
}

func parseDatabaseSpec(spec string) (databaseSpec, error) {
	parts := databaseRegex.FindStringSubmatch(spec)
	if len(parts) != 3 {
		return databaseSpec{}, fmt.Errorf("Invalid database spec: %s", spec)
	}
	protocol, path := parts[1], parts[2]
	if strings.Contains(path, "::") {
		return databaseSpec{}, fmt.Errorf("Invalid database spec: %s", spec)
	}
	switch protocol {
	case "http", "https":
		if strings.HasPrefix(path, ":") {
			path = path[1:]
		}
		if len(path) == 0 {
			return databaseSpec{}, fmt.Errorf("Invalid database spec: %s", spec)
		}
		u, err := url.Parse(protocol + ":" + path)
		if err != nil {
			return databaseSpec{}, fmt.Errorf("Invalid path for %s protocol, spec: %s\n", protocol, spec)
		}
		token := u.Query().Get("access_token")
		return databaseSpec{Protocol: protocol, Path: path, accessToken: token}, nil
	case "ldb":
		if strings.HasPrefix(path, ":") {
			path = path[1:]
		}
		if len(path) == 0 {
			return databaseSpec{}, fmt.Errorf("Invalid database spec: %s", spec)
		}
		return databaseSpec{Protocol: protocol, Path: path}, nil
	case "mem":
		if len(path) > 0 && path != ":" {
			return databaseSpec{}, fmt.Errorf("Invalid database spec (mem path must be empty): %s", spec)
		}
		return databaseSpec{Protocol: protocol, Path: ""}, nil
	default:
		if len(path) != 0 {
			return databaseSpec{}, fmt.Errorf("Invalid protocol for spec: %s", spec)
		}
		return databaseSpec{Protocol: "ldb", Path: protocol}, nil
	}
	return databaseSpec{}, fmt.Errorf("Invalid database spec: %s", spec)
}

func splitDbSpec(spec string, expectHash bool) (dbSpec databaseSpec, rem string, err error) {
	sep := "::"
	if expectHash {
		sep += "#"
	}

	parts := strings.SplitN(spec, sep, 2)
	if len(parts) != 2 {
		err = fmt.Errorf("Missing %s separator in dataset spec %s", sep, spec)
		return
	}

	dbSpec, err = parseDatabaseSpec(parts[0])
	if err != nil {
		return
	}

	rem = parts[1]
	return
}

func parseDatasetSpec(spec string) (dsSpec datasetSpec, rem string, err error) {
	dbSpec, dsPart, dbErr := splitDbSpec(spec, false)
	if dbErr != nil {
		err = dbErr
		return
	}

	remParts := datasetRegex.FindStringSubmatch(dsPart)
	if remParts == nil {
		return datasetSpec{}, "", fmt.Errorf("Invalid dataset spec: component %s of %s must match %s", dsPart, spec, datasetRegex.String())
	}

	dsName := remParts[1]
	dsSpec = datasetSpec{DbSpec: dbSpec, DatasetName: dsName}
	rem = dsPart[len(dsName):]
	return
}

func parseHashSpec(spec string) (hSpec hashSpec, rem string, err error) {
	dbSpec, hashPart, dbErr := splitDbSpec(spec, true)
	if dbErr != nil {
		err = dbErr
		return
	}

	// Hash arbitrary value to figure out hash length.
	hashlen := len(hash.FromData([]byte{}).String())
	if len(hashPart) < hashlen {
		return hashSpec{}, "", fmt.Errorf("Hash %s must be %d characters", hashPart, hashlen)
	}

	hashSpecStr := hashPart[:hashlen]
	h, ok := hash.MaybeParse(hashSpecStr)
	if !ok {
		err = fmt.Errorf("Failed to parse hash: %s", hashSpecStr)
		return
	}

	hSpec = hashSpec{dbSpec, h}
	rem = hashPart[hashlen:]
	return
}

func parseValueSpec(spec string) (valSpec valueSpec, err error) {
	// Try to extract the database from the spec so the error message can be better.
	if _, _, err = splitDbSpec(spec, false); err != nil {
		return
	}

	var rem string

	if s, r, e := parseDatasetSpec(spec); e == nil {
		valSpec = s
		rem = r
	} else if s, r, e := parseHashSpec(spec); e == nil {
		valSpec = s
		rem = r
	} else {
		err = fmt.Errorf("Failed to parse path to value %s", spec)
		return
	}

	if len(rem) == 0 {
		return
	}

	if path, err := types.ParsePath(rem); err == nil {
		valSpec = pathSpec{valSpec, path}
	}
	return
}

func (s databaseSpec) String() string {
	return s.Protocol + ":" + s.Path
}

func (spec databaseSpec) Database() (ds datas.Database, err error) {
	switch spec.Protocol {
	case "http", "https":
		err = d.Unwrap(d.Try(func() {
			ds = datas.NewRemoteDatabase(spec.String(), "Bearer "+spec.accessToken)
		}))
	case "ldb":
		err = d.Unwrap(d.Try(func() {
			ds = datas.NewDatabase(chunks.NewLevelDBStoreUseFlags(spec.Path, ""))
		}))
	case "mem":
		ds = datas.NewDatabase(chunks.NewMemoryStore())
	default:
		err = fmt.Errorf("Invalid path prototocol: %s", spec.Protocol)
	}
	return
}

func (spec datasetSpec) Dataset() (dataset.Dataset, error) {
	store, err := spec.DbSpec.Database()
	if err != nil {
		return dataset.Dataset{}, err
	}

	return dataset.NewDataset(store, spec.DatasetName), nil
}

func (s datasetSpec) String() string {
	return s.DbSpec.String() + "::" + s.DatasetName
}

func (spec datasetSpec) Value() (datas.Database, types.Value, error) {
	dataset, err := spec.Dataset()
	if err != nil {
		return nil, nil, err
	}

	commit, ok := dataset.MaybeHead()
	if !ok {
		dataset.Database().Close()
		return nil, nil, fmt.Errorf("No head value for dataset: %s", spec.DatasetName)
	}

	return dataset.Database(), commit, nil
}

func (spec hashSpec) Value() (datas.Database, types.Value, error) {
	db, err := spec.DbSpec.Database()
	if err != nil {
		return nil, nil, err
	}
	return db, db.ReadValue(spec.Hash), nil
}

func RegisterDatabaseFlags() {
	chunks.RegisterLevelDBFlags()
}

func (spec pathSpec) Value() (datas.Database, types.Value, error) {
	db, root, err := spec.Rel.Value()
	if err != nil {
		return db, root, err
	}

	return db, spec.Path.Resolve(root), nil
}
