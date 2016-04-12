package flags

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

func TestReadDatastoreFromHTTP(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewHTTPStore("http://localhost:8000", "")
	server := datas.NewDataStoreServer(datas.NewTestFactory(chunks.NewTestStoreFactory()), 8000)
	go server.Run()

	ds := datas.NewRemoteDataStore(cs)

	r := ds.WriteValue(types.Bool(true))

	ds.Close()

	datastoreName := "http://localhost:8000"
	dsTest, err := ReadDataStore(datastoreName)

	assert.NoError(err)
	assert.Equal(types.Bool(true), dsTest.ReadValue(r))

	dsTest.Close()
	server.Stop()
}

func TestReadDatastoreFromLDB(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	cs := chunks.NewLevelDBStore(dir+"/name", "", 24, false)
	ds := datas.NewDataStore(cs)

	r := ds.WriteValue(types.Bool(true))

	ds.Close()

	datastoreName := "ldb:" + dir + "/name"
	dsTest, errRead := ReadDataStore(datastoreName)

	assert.NoError(errRead)
	assert.Equal(types.Bool(true), dsTest.ReadValue(r))

	dsTest.Close()
	os.Remove(dir)
}

func TestReadDatastoreFromMem(t *testing.T) {
	assert := assert.New(t)

	datastoreName := "mem:"
	dsTest, err := ReadDataStore(datastoreName)

	r := dsTest.WriteValue(types.Bool(true))

	assert.NoError(err)
	assert.Equal(types.Bool(true), dsTest.ReadValue(r))
}

func TestDatastoreBadInput(t *testing.T) {
	assert := assert.New(t)

	badName1 := "mem"
	ds, err := ReadDataStore(badName1)

	assert.Error(err)
	assert.Nil(ds)
}

func TestReadDatasetFromHTTP(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewHTTPStore("http://localhost:8001", "")
	server := datas.NewDataStoreServer(datas.NewTestFactory(chunks.NewTestStoreFactory()), 8001)
	go server.Run()

	ds := datas.NewRemoteDataStore(cs)
	id := "datasetTest"

	set := dataset.NewDataset(ds, id)
	commit := types.NewString("Commit Value")
	set, err := set.Commit(commit)
	assert.NoError(err)

	ds.Close()

	datasetName := "http://localhost:8001:datasetTest"
	setTest, err := ReadDataset(datasetName)

	assert.NoError(err)
	assert.EqualValues(commit, setTest.Head().Value())

	server.Stop()
}

func TestReadDatasetFromMem(t *testing.T) {
	assert := assert.New(t)

	datasetName := "mem::datasetTest"
	dsTest, errTest := ReadDataset(datasetName)

	assert.NoError(errTest)

	commit := types.NewString("Commit Value")
	dsTest, err := dsTest.Commit(commit)
	assert.NoError(err)

	assert.EqualValues(commit, dsTest.Head().Value())
}

func TestReadDatasetFromLDB(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	cs := chunks.NewLevelDBStore(dir+"/name", "", 24, false)
	ds := datas.NewDataStore(cs)
	id := "testdataset"

	set := dataset.NewDataset(ds, id)
	commit := types.NewString("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)

	ds.Close()

	datasetName := "ldb:" + dir + "/name:" + id
	setTest, errRead := ReadDataset(datasetName)

	assert.NoError(errRead)
	assert.EqualValues(commit, setTest.Head().Value())

	os.Remove(dir)
}

func TestDatasetBadInput(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	badName := "ldb:" + dir + "/name:--bad"
	ds, err := ReadDataset(badName)

	assert.Error(err)
	assert.NotNil(ds)

	badName2 := "ldb:" + dir
	ds, err = ReadDataset(badName2)

	assert.Error(err)
	assert.NotNil(ds)

	badName3 := "mem"
	ds, err = ReadDataset(badName3)

	assert.Error(err)
	assert.NotNil(ds)

	os.Remove(dir)
}

func TestReadDatasetObjectFromLdb(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	cs := chunks.NewLevelDBStore(dir+"/name", "", 24, false)
	ds := datas.NewDataStore(cs)
	id := "testdataset"

	set := dataset.NewDataset(ds, id)
	commit := types.NewString("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)

	ds.Close()

	datasetName := "ldb:" + dir + "/name:" + id
	setTest, ref, isDs, errRead := ReadObject(datasetName)

	assert.Zero(ref)
	assert.True(isDs)
	assert.NoError(errRead)
	assert.EqualValues(commit, setTest.Head().Value())

	os.Remove(dir)
}

func TestReadRef(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	cs := chunks.NewLevelDBStore(dir+"/name", "", 24, false)
	ds := datas.NewDataStore(cs)
	id := "testdataset"

	set := dataset.NewDataset(ds, id)
	commit := types.NewString("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)

	ref := set.Head().Ref()

	ds.Close()

	objectName := "ldb:" + dir + "/name:" + ref.String()

	set, refTest, isDs, errRead := ReadObject(objectName)

	assert.EqualValues(ref.String(), refTest.String())
	assert.False(isDs)
	assert.NoError(errRead)
	assert.Zero(set)
}

func TestReadObjectBadInput(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	badName := "ldb:" + dir + "/name:sha2-78888888888"
	ds, ref, isDs, err := ReadObject(badName)

	//it interprets it as a dataset id

	assert.NoError(err)
	assert.NotNil(ds)
	assert.Zero(ref)
	assert.True(isDs)
}

//need a good way to test this without overwriting any potential $HOME/.noms folder...
func TestDefaultDatastore(t *testing.T) {
	assert := assert.New(t)

	assert.True(true)
}
