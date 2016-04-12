package flags

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
)

func ReadDataStore(in string) (ds datas.DataStore, err error) {
	input := strings.Split(in, ":")

	if len(input) < 2 {
		err = errors.New("Improper datastore name")
		return
	}

	var cs chunks.ChunkStore

	switch input[0] {
	case "http":
		//get from server and path
		cs = chunks.NewHTTPStore(in, "")
		ds = datas.NewRemoteDataStore(cs)

	case "ldb":
		//create/access from path
		cs = chunks.NewLevelDBStore(input[1], "", 24, false)
		ds = datas.NewDataStore(cs)

	case "mem":
		cs = chunks.NewMemoryStore()
		ds = datas.NewDataStore(cs)

	case "":
		cs = chunks.NewLevelDBStore("$HOME/.noms", "", 24, false)
		ds = datas.NewDataStore(cs)

	default:
		err = fmt.Errorf("Improper datastore name: %s", in)
		ds = datas.NewDataStore(cs)
		return

	}

	return
}

func ReadDataset(in string) (data dataset.Dataset, err error) {
	input := strings.Split(in, ":")

	if len(input) < 3 {
		err = fmt.Errorf("Improper dataset name: %s", in)
		return
	}

	ds, errStore := ReadDataStore(strings.Join(input[0:len(input)-1], ":"))
	name := input[len(input)-1]

	d.Chk.NoError(errStore)

	validIn := regexp.MustCompile("^[a-zA-Z0-9]+([/\\-_][a-zA-Z0-9]+)*$")

	if !validIn.MatchString(name) {
		err = fmt.Errorf("Improper dataset name: %s", in)
		return
	}

	data = dataset.NewDataset(ds, name)

	return
}

func ReadObject(in string) (ds dataset.Dataset, r ref.Ref, isDs bool, err error) {
	input := strings.Split(in, ":")

	if len(input) < 3 {
		err = fmt.Errorf("Improper object name: %s", in)
		return
	}

	objectName := input[len(input)-1]

	isDs = true

	r, isRef := ref.MaybeParse(objectName)

	if isRef {
		isDs = false
		return
	}

	ds, isValid := ReadDataset(in)
	isDs = true

	d.Chk.NoError(isValid)

	return
}
