package fs

import (
	"io/ioutil"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/integrationtest"
)

const dsName = "test-fs"

func TestIntegration(t *testing.T) {
	integrationtest.RunIntegrationSuite(t, &testSuite{})
}

type testSuite struct {
	integrationtest.IntegrationSuite
}

func (s *testSuite) CheckData(cs chunks.ChunkStore) {
	db := datas.NewDatabase(cs)
	defer db.Close()
	ds := dataset.NewDataset(db, dsName)
	v := ds.HeadValue()
	s.True(v.Type().Equals(types.MakeStructType("File", map[string]*types.Type{
		"content": types.MakeRefType(types.BlobType),
	})))
	s.Equal("File", v.(types.Struct).Type().Desc.(types.StructDesc).Name)
	b := v.(types.Struct).Get("content").(types.Ref).TargetValue(db).(types.Blob)

	bs, err := ioutil.ReadAll(b.Reader())
	s.NoError(err)
	s.Equal([]byte("Hello World!\n"), bs)
}

func (s *testSuite) NodeArgs() []string {
	return []string{"./test-data.txt", s.ValueSpecString(dsName)}
}

func (s *testSuite) CheckNode(out string) {
	s.Contains(out, "1 of 1 entries")
	s.Contains(out, "done")
}
