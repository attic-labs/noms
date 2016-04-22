package flags

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

const (
	maxFileHandles = 24
)

var (
	validDatasetNameRegexp = regexp.MustCompile("^[a-zA-Z0-9]+([/\\-_][a-zA-Z0-9]+)*$")
)

type ObjectSpec struct {
	Protocol    string
	Path        string
	DatasetName string
	Ref         string
}

func ParseObjectSpec(spec string) (ObjectSpec, error) {
	comps := strings.Split(spec, ":")
	err := errors.New("Incorrect syntax for data spec")
	protocol, rest := comps[0], comps[1:]
	path, end := "", ""
	switch protocol {
	case "http":
		if len(rest) == 1 {
			path, end = rest[0], ""
		} else if len(rest) == 2 {
			re := regexp.MustCompile("^\\d+([/].*)*")
			if re.MatchString(rest[1]) {
				path, end = rest[0]+":"+rest[1], ""
			} else {
				path, end = rest[0], rest[1]
			}
		} else if len(rest) == 3 {
			path, end = rest[0]+":"+rest[1], rest[2]
		} else {
			return ObjectSpec{}, err
		}
		if path == "" {
			return ObjectSpec{}, errors.New("Illegal empty path in 'http' dataspec")
		}
	case "ldb":
		if len(rest) == 1 {
			path, end = rest[0], ""
		} else if len(rest) == 2 {
			path, end = rest[0], rest[1]
		} else {
			return ObjectSpec{}, err
		}
		if path == "" {
			return ObjectSpec{}, errors.New("Illegal empty path in 'ldb' dataspec")
		}
	case "mem":
		if len(rest) == 0 {
			path, end = "", ""
		} else if len(rest) == 1 {
			path, end = "", rest[0]
		} else {
			return ObjectSpec{}, err
		}
		if path != "" {
		}
	default: // ldb:$HOME/.noms
		if len(rest) == 0 {
			protocol = "ldb"
			path, end = "ldb:$HOME/.noms", ""
		} else if len(rest) == 1 {
			path, end = "ldb:$HOME/.noms", rest[0]
		} else if len(rest) == 2 {
			path, end = rest[0], rest[1]
		} else {
			return ObjectSpec{}, fmt.Errorf("Unknown protocol in data spec: %s", protocol)
		}
	}

	path = strings.TrimRight(path, "/")
	re := regexp.MustCompile("^sha1-.*")
	if re.MatchString(end) {
		return ObjectSpec{Protocol: protocol, Path: path, DatasetName: "", Ref: end}, nil
	}

	if end != "" && !validDatasetNameRegexp.MatchString(end) {
		return ObjectSpec{}, fmt.Errorf("Illegal dataset name: %s", end)
	}
	return ObjectSpec{Protocol: protocol, Path: path, DatasetName: end, Ref: ""}, nil
}

func (spec ObjectSpec) IsObjectSpec() bool {
	return spec.Ref != ""
}

func (spec ObjectSpec) IsDatasetSpec() bool {
	return spec.DatasetName != ""
}

func (spec ObjectSpec) IsDatastoreSpec() bool {
	return spec.Ref == "" && spec.DatasetName == ""
}

func GetDataStore(spec ObjectSpec) (ds datas.DataStore, err error) {
	switch spec.Protocol {
	case "http":
		ds = datas.NewRemoteDataStore(spec.Protocol+":"+spec.Path, "")
	case "ldb":
		ds = datas.NewDataStore(chunks.NewLevelDBStore(spec.Path, "", maxFileHandles, false))
	case "mem":
		ds = datas.NewDataStore(chunks.NewMemoryStore())
	}

	return
}

func GetDataset(spec ObjectSpec) (dataset.Dataset, error) {
	store, err := GetDataStore(spec)
	if err != nil {
		return dataset.Dataset{}, err
	}

	return dataset.NewDataset(store, spec.DatasetName), nil
}

func GetObject(spec ObjectSpec) (datas.DataStore, types.Value, error) {
	store, err := GetDataStore(spec)
	if err != nil {
		return store, nil, err
	}

	if r, isRef := ref.MaybeParse(spec.Ref); isRef {
		v := store.ReadValue(r)
		return store, v, nil
	}

	return store, nil, nil
}

func DataStoreFromSpec(spec string) (datas.DataStore, error) {
	objSpec, err := ParseObjectSpec(spec)
	if err != nil {
		return nil, err
	}

	if !objSpec.IsDatastoreSpec() {
		return nil, errors.New("Illegal datastore spec")
	}

	return GetDataStore(objSpec)
}

func DatasetFromSpec(spec string) (dataset.Dataset, error) {
	objSpec, err := ParseObjectSpec(spec)
	if err != nil {
		return dataset.Dataset{}, err
	}

	if !objSpec.IsDatasetSpec() {
		return dataset.Dataset{}, errors.New("Illegal dataset spec")
	}

	return GetDataset(objSpec)
}

func ObjectFromSpec(spec string) (datas.DataStore, types.Value, error) {
	objSpec, err := ParseObjectSpec(spec)
	if err != nil {
		return nil, nil, err
	}

	if !objSpec.IsObjectSpec() {
		return nil, nil, errors.New("Illegal object spec")
	}

	return GetObject(objSpec)
}
