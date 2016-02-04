package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/attic-labs/noms/clients/csv"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/dataset"
)

var (
	p       = flag.Int("p", 512, "parallelism")
	dsFlags = dataset.NewFlags()
	// Actually the delimiter uses runes, which can be multiple characters long.
	// https://blog.golang.org/strings
	delimiter = flag.String("delimiter", ",", "field delimiter for csv file, must be exactly one character long.")
)

func main() {
	cpuCount := runtime.NumCPU()
	runtime.GOMAXPROCS(cpuCount)

	flag.Usage = func() {
		fmt.Println("Usage: csv_exporter [options] filename\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	ds := dsFlags.GetDataset()
	if ds == nil {
		flag.Usage()
		return
	}
	defer ds.Store().Close()

	if flag.NArg() != 1 {
		fmt.Printf("Expected exactly one parameter (path) after flags, but you have %d. Maybe you put a flag after the path?\n", flag.NArg())
		flag.Usage()
		return
	}

	path := flag.Arg(0)
	if path == "" {
		flag.Usage()
		return
	}
	f, err := os.Create(path)
	d.Exp.NoError(err)
	defer f.Close()

	comma, err := csv.StringToRune(*delimiter)
	if err != nil {
		fmt.Println(err.Error())
		flag.Usage()
		return
	}

	csv.Write(ds, comma, *p, f)
}
