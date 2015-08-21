package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
	"github.com/clbanning/mxj"
)

var (
	noIO        = flag.Bool("benchmark", false, "Run in 'benchmark' mode, without file-IO")
	customUsage = func() {
		fmtString := `%s walks the given directory, looking for .xml files. When it finds one, the entity inside is parsed into nested Noms maps/lists and committed to the dataset indicated on the command line.`
		fmt.Fprintf(os.Stderr, fmtString, os.Args[0])
		fmt.Fprintf(os.Stderr, "\n\nUsage: %s [options] <path/to/root/directory>\n", os.Args[0])
		flag.PrintDefaults()
	}
)

func main() {
	err := d.Try(func() {
		dsFlags := dataset.NewFlags()
		flag.Usage = customUsage
		flag.Parse()
		ds := dsFlags.CreateDataset()
		dir := flag.Arg(0)
		if ds == nil || dir == "" {
			flag.Usage()
			return
		}

		if util.MaybeStartCPUProfile() {
			defer util.StopCPUProfile()
		}

		list := types.NewList()

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			d.Exp.NoError(err, "Cannot traverse directories")
			if info.IsDir() || filepath.Ext(path) != ".xml" {
				return nil
			}
			file, err := os.Open(path)
			d.Exp.NoError(err, "Error getting XML")
			defer file.Close()

			xmlObject, err := mxj.NewMapXmlReader(file)
			d.Exp.NoError(err, "Error decoding XML")
			object := xmlObject.Old()

			nomsObj := util.NomsValueFromDecodedJSON(object)
			if *noIO {
				return nil
			}

			ref := types.WriteValue(nomsObj, ds)
			list = list.Append(types.Ref{R: ref})
			return nil
		})
		d.Exp.NoError(err)

		if !*noIO {
			for ok := false; !ok; ds, ok = attemptCommit(list, ds) {
				continue
			}
		}

		util.MaybeWriteMemProfile()
	})

	if err != nil {
		log.Fatal(err)
	}
}

func attemptCommit(newValue types.Value, ds *dataset.Dataset) (*dataset.Dataset, bool) {
	newDs, ok := ds.Commit(
		datas.NewCommit().SetParents(ds.HeadAsSet()).SetValue(newValue))
	return &newDs, ok
}
