package counter

import (
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/integrationtest"
)

const dsName = "test-counter"

func TestIntegration(t *testing.T) {
	integrationtest.RunIntegrationSuite(t, &testSuite{})
}

type testSuite struct {
	integrationtest.IntegrationSuite
}

func (s *testSuite) SetupData(cs chunks.ChunkStore) {
	db := datas.NewDatabase(cs)
	defer db.Close()
	ds := dataset.NewDataset(db, dsName)
	var err error
	ds, err = ds.Commit(types.Number(42))
	s.NoError(err)
}

func (s *testSuite) CheckData(cs chunks.ChunkStore) {
	db := datas.NewDatabase(cs)
	defer db.Close()
	ds := dataset.NewDataset(db, dsName)
	s.True(ds.HeadValue().Equals(types.Number(43)))
}

func (s *testSuite) NodeArgs() []string {
	spec := s.ValueSpecString(dsName)
	return []string{spec}
}

func (s *testSuite) CheckNode(out string) {
	s.Equal("43\n", out)
}
