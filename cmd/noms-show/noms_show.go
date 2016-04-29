package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/attic-labs/noms/clients/flags"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/types"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <object>\n", os.Args[0])
	}

	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	spec, err := flags.ParseObjectSpec(flag.Arg(0))
	if err != nil {
		printError(err.Error())
		return
	}

	var val types.Value
	if spec.IsDatasetSpec() {
		ds, err := flags.GetDataset(spec)
		d.Chk.NoError(err)
		commit, hasHead := ds.MaybeHead()
		if !hasHead {
			printError("Specified dataset does not exist")
			return
		}
		val = commit.Get("value")
	} else {
		_, val, err = flags.GetObject(spec)
		d.Chk.NoError(err)
		if val == nil {
			printError("Specified ref does not exist")
			return
		}
	}

	fmt.Println(types.WriteTaggedHRS(val))
}

func printError(s string) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", s))
	os.Exit(1)
}
