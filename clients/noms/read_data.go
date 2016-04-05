package noms

import (
	"strings"
	"regexp"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
)

//TODO: default
func ReadDatastore(in string) (data datas.DataStore, err bool) {
	input := strings.Split(in, ":")
	err = false

	var cs chunks.ChunkStore

	switch input[0] {
	case "http":
		//get from server and path
		cs = chunks.NewHTTPStore(input[1], "")
		err = true

	/*case "ldb":
		//create/access from path
		path := strings.Split(input[1], "/")
		dir := strings.Join(path[0:len(path) - 2], "")
		name := path[len(path) - 1]
		cs = NewLevelDBStore(dir, name, 24, false)
		err = true
*/
	case "mem":
		//get from in memory
		cs = chunks.NewMemoryStore()
		err = true

	default:
		return
	}

	data = datas.NewDataStore(cs)
	return


}

func ReadDataset(in string) (data dataset.Dataset, err bool) {
	input := strings.Split(in, ":")

	ds, validDatastore := ReadDatastore(strings.Join(input[0:1], ""))

	if (!validDatastore) {
		err = false
		return
	}

	validIn := regexp.MustCompile("^[a-zA-Z0-9]+([/\\-_][a-zA-Z0-9]+)*$")

	if (!validIn.MatchString(input[2])) {
		err = false
		return
	}

	data = dataset.NewDataset(ds, input[2])

	return
}

func ReadObject(in string) (ds dataset.Dataset, r ref.Ref, err bool, isDs bool) {
	err = false
	r, isRef := ref.MaybeParse(in)

	if (isRef) {
		err = true
		isDs = false
		return
	}

	ds, isValid := ReadDataset(in)
	isDs = true

	err = isValid

	return
}