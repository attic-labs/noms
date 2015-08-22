package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

func main() {
	dsFlags := dataset.NewFlags()
	flag.Parse()
	ds := dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}

	url := flag.Arg(0)
	if ds == nil || url == "" {
		flag.Usage()
		return
	}

	res, err := http.Get(url)
	defer res.Body.Close()
	if err != nil {
		log.Fatalf("Error fetching %s: %+v\n", url, err)
	} else if res.StatusCode != 200 {
		log.Fatalf("Error fetching %s: %s\n", url, res.Status)
	}

	var jsonObject interface{}
	err = json.NewDecoder(res.Body).Decode(&jsonObject)
	if err != nil {
		log.Fatalln("Error decoding JSON: ", err)
	}

	value := util.NomsValueFromDecodedJSON(jsonObject)
	for ok := false; !ok; ds, ok = attemptCommit(value, ds) {
		continue
	}
}

func attemptCommit(newValue types.Value, ds *dataset.Dataset) (*dataset.Dataset, bool) {
	newDs, ok := ds.Commit(
		datas.NewCommit().SetParents(ds.HeadAsSet()).SetValue(newValue))
	return &newDs, ok
}
