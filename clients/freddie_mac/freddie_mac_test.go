package main

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/suite"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

func TestFreddieMac(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	util.ClientTestSuite
}

func (s *testSuite) TestOrig() {
	oldInputType := inputType
	newInputType := "orig"
	inputType = &newInputType
	defer func() { inputType = oldInputType }()

	input, err := ioutil.TempFile(s.TempDir, "")
	d.Chk.NoError(err)

	inputData := [][]string{
		[]string{"811", "201403", "N", "204402", "30460", "25", "1", "O", "90", "18", "83000", "90", "4.5", "R", "N", "FIX30", "KY", "SF", "40300", "F114Q1000022", "P", "360", "02", "Other sellers", "Other servicers"},
		[]string{"738", "201403", "N", "204402", "14540", "000", "1", "O", "50", "23", "200000", "50", "4.5", "R", "N", "FIX30", "KY", "SF", "42100", "F114Q1000034", "P", "360", "02", "Other sellers", "Other servicers"},
	}

	for _, inputRow := range inputData {
		_, err = input.WriteString(strings.Join(inputRow, "|") + "\n")
		d.Chk.NoError(err)
	}

	_, err = input.Seek(0, 0)
	d.Chk.NoError(err)

	out := s.Run(main, []string{"-ds", "csv", input.Name()})
	s.Equal("", out)

	cs := chunks.NewLevelDBStore(s.LdbDir, 1, false)
	ds := dataset.NewDataset(datas.NewDataStore(cs), "csv")
	defer ds.Close()

	m := ds.Head().Value().(types.Map)
	s.Equal(uint64(len(inputData)), m.Len())

	keyColumn := 19
	for _, inputRow := range inputData {
		st := types.ReadValue(m.Get(types.NewString(inputRow[keyColumn])).(types.Ref).TargetRef(), cs).(types.Struct)
		for i, field := range inputRow {
			s.Equal(field, st.Get(inputSpecs["orig"].fields[i]).(types.String).String())
		}
	}
}
