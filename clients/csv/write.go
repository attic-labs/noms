package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"

	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

func getFieldNamesFromStruct(structDesc types.StructDesc) (fieldNames []string) {
	for _, f := range structDesc.Fields {
		if !types.IsPrimitiveKind(f.T.Desc.Kind()) {
			log.Fatalln("Non-primitive CSV export not supported:",
				f.T.Desc.Describe())
		}
		fieldNames = append(fieldNames, f.Name)
	}
	return
}

func Write(ds *dataset.Dataset, comma rune, concurrency int, output io.Writer) {
	v := ds.Head().Value()
	if v.Type().Desc.Kind() != types.ListKind {
		log.Fatalln("Dataset must be List<>, found", v.Type().Desc.Describe())
	}
	t := v.Type().Desc.(types.CompoundDesc).ElemTypes[0]
	if t.Desc.Kind() != types.RefKind {
		log.Fatalln("List<> must be of Ref, found", t.Desc.Describe())
	}
	u := t.Desc.(types.CompoundDesc).ElemTypes[0]
	if u.Desc.Kind() != types.UnresolvedKind {
		log.Fatalln("Ref must be UnresolvedKind, found", u.Desc.Describe())
	}
	pkg := types.ReadPackage(u.PackageRef(), ds.Store())
	if pkg.Type().Desc.Kind() != types.PackageKind {
		log.Fatalln("Failed to read package:", pkg.Type().Desc.Describe())
	}
	structDesc := pkg.Types()[u.Ordinal()].Desc
	if structDesc.Kind() != types.StructKind {
		log.Fatalln("Did not find Struct:", structDesc.Describe())
	}
	fieldNames := getFieldNamesFromStruct(structDesc.(types.StructDesc))
	nomsList := v.(types.List)

	csvWriter := csv.NewWriter(output)
	csvWriter.Comma = comma

	records := make([][]string, nomsList.Len()+1)
	records[0] = fieldNames // Write header

	nomsList.IterAllP(concurrency, func(v types.Value, index uint64) {
		for _, f := range fieldNames {
			records[index+1] = append(
				records[index+1],
				fmt.Sprintf("%s", v.(types.Ref).TargetValue(ds.Store()).(types.Struct).
					Get(f)))
		}
	})

	csvWriter.WriteAll(records)
	if err := csvWriter.Error(); err != nil {
		log.Fatalln("error flushing csv:", err)
	}
}
