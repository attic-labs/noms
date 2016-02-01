package csv

import (
	"encoding/csv"
	"io"
	"log"

	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

func getFieldNamesFromStruct(s types.Struct) (fieldNames []string) {
	for _, f := range s.Desc().Fields {
		fieldNames = append(fieldNames, f.Name)
	}
	return
}

func Write(ds *dataset.Dataset, comma rune, output io.Writer) {
	rows := ds.Head().Value().(types.List)
	// Gather field names from first Struct in rows.
	fieldNames := getFieldNamesFromStruct(
		rows.Get(0).(types.Ref).TargetValue(ds.Store()).(types.Struct))
	csvWriter := csv.NewWriter(output)
	csvWriter.Comma = comma

	csvWriter.Write(fieldNames) // Write headers

	rows.IterAll(func(v types.Value, index uint64) {
		values := []string{}

		for _, f := range fieldNames {
			values = append(
				values,
				v.(types.Ref).TargetValue(ds.Store()).(types.Struct).
					Get(f).(types.String).String())
		}

		if err := csvWriter.Write(values); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	})

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		log.Fatalln("error flushing csv:", err)
	}
}
