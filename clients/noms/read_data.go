package read_data

import (
	"strings"
	"regexp"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/datasets"
)

//TODO: default
func ReadDatastore(in string) datas.DataStore, bool {
	input := strings.Split(in, ":")

	switch input[:0] {
	case "http":
		//get from server and path
		http_cs := NewHTTPStore(input[:1], nil)
		return newRemoteDataStore(http_cs), true

	case "ldb":
		//create/access from path
		ldb_cs := 
		return newLocalDataStore(ldb_cs), true

	case "mem":
		//get from in memory
		mem_cs := NewMemoryStore()
		return newLocalDataStore(mem_cs), true

	default:
		return nil, false
	}


}

func ReadDataset(in string) dataset.Dataset, bool {
	input := strings.Split(in, ":")

	ds := ReadDatastore(Join(input[0:1])

	validIn := regexp.MustCompile('^[a-zA-Z0-9]+([/\-_][a-zA-Z0-9]+)*$')

	if (!validIn.MatchString(input[:2])) {
		return nil, false
	}

}

func ReadObject(in string) (dataset.Dataset ds, ref.Ref r, bool err) {
	err = false
	r, isRef := ref.MaybeParse(in)

	if (isRef) {
		err = true
		return
	}

	ds, isValid := ReadDataset(in)

	err = isValid

	return
}