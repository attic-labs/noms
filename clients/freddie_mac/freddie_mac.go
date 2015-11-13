package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var (
	p         = flag.Uint("p", 512, "parallelism")
	dsFlags   = dataset.NewFlags()
	inputType = flag.String("type", "", "The type of the data we are importing [orig|payment]")
	delimRune = '|'
)

type valuesFromCSV struct {
	values []string
}

type refKeys struct {
	ref     types.Ref
	seq     types.Value
	quarter types.Value
}

type inputSpec struct {
	structName string   // The name of the struct type
	fields     []string // The name for each field (the delimiter | should not appear anywhere in these strings
}

var (
	inputSpecs = map[string]inputSpec{
		"orig": inputSpec{
			structName: "LoanOrigination",
			fields:     []string{"creditscore", "firstpayment", "firsttime", "maturity", "msa", "insurance", "units", "occupancy", "cltv", "dti", "upb", "ltv", "interest", "channel", "ppm", "product", "state", "property", "zip", "seq", "purpose", "term", "borrowers", "seller", "servicer"},
		},
	}
)

func main() {
	cpuCount := runtime.NumCPU()
	runtime.GOMAXPROCS(cpuCount)

	flag.Usage = func() {
		fmt.Println("Usage: freddie_mac [options] file\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	ds := dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}
	defer ds.Close()

	if flag.NArg() != 1 {
		fmt.Printf("Expected exactly one parameter (path) after flags, but you have %d. Maybe you put a flag after the path?\n", flag.NArg())
		flag.Usage()
		return
	}

	path := flag.Arg(0)
	if ds == nil || path == "" {
		flag.Usage()
		return
	}

	res, err := os.Open(path)
	d.Exp.NoError(err)
	defer res.Close()

	inputSpec, ok := inputSpecs[*inputType]
	d.Exp.True(ok, fmt.Sprintf("Invalid input type (%s)", *inputType))

	// var delimStr string
	// {
	// 	buf := make([]byte, utf8.UTFMax)
	// 	delimStr = string(buf[:utf8.EncodeRune(buf, delimRune)])
	// }

	r := csv.NewReader(res)
	r.Comma = '|'
	r.FieldsPerRecord = 0 // Let first row determine the number of fields.

	fields := make([]types.Field, 0, len(inputSpec.fields))
	for _, key := range inputSpec.fields {
		fields = append(fields, types.Field{
			Name: key,
			T:    types.MakePrimitiveTypeRef(types.StringKind),
			// TODO(misha): Think about whether we need fields to be optional.
			Optional: false,
		})
	}

	typeDef := types.MakeStructTypeRef(inputSpec.structName, fields, types.Choices{})
	pkg := types.NewPackage([]types.Type{typeDef}, []ref.Ref{})
	pkgRef := types.RegisterPackage(&pkg)
	typeRef := types.MakeTypeRef(pkgRef, 0)

	recordChan := make(chan valuesFromCSV, 4096)
	refChan := make(chan refKeys, 4096)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatalln("Error decoding CSV: ", err)
			}

			wg.Add(1)
			recordChan <- valuesFromCSV{row}
		}

		wg.Done()
		close(recordChan)
	}()

	rowsToNoms := func() {
		for row := range recordChan {
			fields := make(map[string]types.Value)
			for i, v := range row.values {
				fields[inputSpec.fields[i]] = types.NewString(v)
			}
			newStruct := types.NewStruct(typeRef, typeDef, fields)
			r := types.NewRef(types.WriteValue(newStruct, ds.Store()))
			seq, _ := newStruct.MaybeGet("seq")
			quarter, _ := newStruct.MaybeGet("quarter")
			refChan <- refKeys{
				ref:     r,
				seq:     seq,
				quarter: quarter,
			}
		}
	}

	for i := uint(0); i < *p; i++ {
		go rowsToNoms()
	}

	refList := []refKeys{}

	go func() {
		for r := range refChan {
			refList = append(refList, r)
			wg.Done()
		}
	}()

	wg.Wait()

	refs := make([]types.Value, 0, 2*len(refList))
	for _, r := range refList {
		refs = append(refs, r.seq, r.ref)
	}

	value := types.NewMap(refs...)
	_, ok = ds.Commit(value)
	d.Exp.True(ok, "Could not commit due to conflicting edit")
}
