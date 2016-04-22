package flags

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

func TestParseDataStoreFromHTTP(t *testing.T) {
	assert := assert.New(t)
	const port = 8017
	const testString = "A String for testing"
	const dsetId = "testds"
	storeSpec := fmt.Sprintf("http://localhost:%d/", port)

	server := datas.NewRemoteDataStoreServer(chunks.NewTestStoreFactory(), port)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		server.Run()
		wg.Done()
	}()
	time.Sleep(time.Second)

	store1, err := DataStoreFromSpec(storeSpec)
	assert.NoError(err)

	r1 := store1.WriteValue(types.NewString(testString))
	store1, err = store1.Commit(dsetId, datas.NewCommit().SetValue(r1))
	assert.NoError(err)
	store1.Close()

	store2, err := DataStoreFromSpec(storeSpec)
	assert.NoError(err)
	assert.Equal(types.NewString(testString), store2.ReadValue(r1.TargetRef()))

	server.Stop()
	wg.Wait()
}

func TestParseDataStoreFromLDB(t *testing.T) {
	assert := assert.New(t)

	d1 := os.TempDir()
	dir, err := ioutil.TempDir(d1, "flags")
	assert.NoError(err)
	ldbDir := path.Join(dir, "store")
	storeSpec := "ldb:" + ldbDir

	cs := chunks.NewLevelDBStore(ldbDir, "", 24, false)
	ds := datas.NewDataStore(cs)

	s1 := types.NewString("A String")
	ds.WriteValue(s1)
	ds.Commit("testDs", datas.NewCommit().SetValue(types.NewRef(s1.Ref())))
	ds.Close()

	dsTest, errRead := DataStoreFromSpec(storeSpec)
	assert.NoError(errRead)
	assert.Equal(s1.String(), dsTest.ReadValue(s1.Ref()).(types.String).String())
	dsTest.Close()
	os.Remove(dir)
}

func TestParseDataStoreFromMem(t *testing.T) {
	assert := assert.New(t)

	datastoreName := "mem:"
	dsTest, err := DataStoreFromSpec(datastoreName)

	r := dsTest.WriteValue(types.Bool(true))

	assert.NoError(err)
	assert.Equal(types.Bool(true), dsTest.ReadValue(r.TargetRef()))
}

func TestParseDatasetFromHTTP(t *testing.T) {
	assert := assert.New(t)
	const port = 8018
	const datasetId = "dsTest"
	storeSpec := fmt.Sprintf("http://localhost:%d", port)
	dsSpec := fmt.Sprintf("%s:%s", storeSpec, datasetId)

	server := datas.NewRemoteDataStoreServer(chunks.NewTestStoreFactory(), port)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		server.Run()
		wg.Done()
	}()
	time.Sleep(time.Second)

	store, err := DataStoreFromSpec(storeSpec)
	assert.NoError(err)

	dset1 := dataset.NewDataset(store, datasetId)
	s1 := types.NewString("Commit Value")
	dset1, err = dset1.Commit(s1)
	assert.NoError(err)
	store.Close()

	dset2, err := DatasetFromSpec(dsSpec)
	assert.NoError(err)

	assert.EqualValues(s1, dset2.Head().Value())

	server.Stop()
	wg.Wait()
}

func TestParseDatasetFromMem(t *testing.T) {
	assert := assert.New(t)

	datasetName := "mem:datasetTest"
	dsTest, errTest := DatasetFromSpec(datasetName)

	assert.NoError(errTest)

	commit := types.NewString("Commit Value")
	dsTest, err := dsTest.Commit(commit)
	assert.NoError(err)

	assert.EqualValues(commit, dsTest.Head().Value())
}

func TestParseDatasetFromLDB(t *testing.T) {
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
	setTest, errRead := DatasetFromSpec(datasetName)

	assert.NoError(errRead)
	assert.EqualValues(commit, setTest.Head().Value())

	os.Remove(dir)
}

func TestDatasetBadInput(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	badName := "ldb:" + dir + "/name:--bad"
	ds, err := DatasetFromSpec(badName)

	assert.Error(err)
	assert.NotNil(ds)

	badName2 := "ldb:" + dir
	ds, err = DatasetFromSpec(badName2)

	assert.Error(err)
	assert.NotNil(ds)

	badName3 := "mem"
	ds, err = DatasetFromSpec(badName3)

	assert.Error(err)
	assert.NotNil(ds)

	os.Remove(dir)
}

func TxestParseDatasetObjectFromLdb(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	storePath := dir + "/name"
	cs := chunks.NewLevelDBStore(storePath, "", 24, false)
	ds := datas.NewDataStore(cs)
	datasetId := "testdataset"
	datasetSpec := "ldb:" + storePath + ":" + datasetId

	set := dataset.NewDataset(ds, datasetId)
	commit := types.NewString("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)
	ds.Close()

	_, err = DatasetFromSpec(datasetSpec)
	assert.NoError(err)

	objectSpec := "ldb:" + storePath + ":" + commit.Ref().String()
	_, value, err := ObjectFromSpec(objectSpec)
	assert.NoError(err)

	assert.EqualValues(commit, value)

	os.Remove(dir)
}

func TestReadRef(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	storePath := dir + "/name"
	cs := chunks.NewLevelDBStore(storePath, "", 24, false)
	ds := datas.NewDataStore(cs)
	id := "testdataset"

	set := dataset.NewDataset(ds, id)
	commit := types.NewString("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)

	ref := set.Head().Ref()

	ds.Close()

	objectName := "ldb:" + storePath + ":" + ref.String()
	_, value, err := ObjectFromSpec(objectName)
	assert.NoError(err)

	assert.EqualValues(ref.String(), value.Ref().String())
}

func TestParseObjectBadInput(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)

	badName := "ldb:" + dir + "/name:sha2-78888888888"
	_, _, err = ObjectFromSpec(badName)
	assert.Error(err)
}

//need a good way to test this without overwriting any potential $HOME/.noms folder...
func TestDefaultDatastore(t *testing.T) {
	assert := assert.New(t)

	assert.True(true)
}

func TestParseObjectSpec(t *testing.T) {
	assert := assert.New(t)

	pspec, err := ParseObjectSpec("http://localhost:8000")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//localhost:8000"}, pspec)

	pspec, err = ParseObjectSpec("http://localhost:8000/")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//localhost:8000"}, pspec)

	pspec, err = ParseObjectSpec("http://localhost:8000/fff")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//localhost:8000/fff"}, pspec)

	pspec, err = ParseObjectSpec("http://localhost:8000:dsname")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//localhost:8000", DatasetName: "dsname"}, pspec)

	pspec, err = ParseObjectSpec("http://localhost:8000/john/doe/:dsname")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//localhost:8000/john/doe", DatasetName: "dsname"}, pspec)

	pspec, err = ParseObjectSpec("http://local.attic.io/john/doe")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//local.attic.io/john/doe"}, pspec)

	pspec, err = ParseObjectSpec("http://local.attic.io/john/doe:dsname")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//local.attic.io/john/doe", DatasetName: "dsname"}, pspec)

	pspec, err = ParseObjectSpec("http://local.attic.io/john/doe:sha1-234523542345")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "http", Path: "//local.attic.io/john/doe", Ref: "sha1-234523542345"}, pspec)

	pspec, err = ParseObjectSpec("ldb:/filesys/john/doe")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "ldb", Path: "/filesys/john/doe"}, pspec)

	pspec, err = ParseObjectSpec("ldb:/filesys/john/doe:dsname")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "ldb", Path: "/filesys/john/doe", DatasetName: "dsname"}, pspec)

	pspec, err = ParseObjectSpec("ldb:/filesys/john/doe:sha1-234523542345")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "ldb", Path: "/filesys/john/doe", Ref: "sha1-234523542345"}, pspec)

	pspec, err = ParseObjectSpec("mem")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "mem"}, pspec)

	pspec, err = ParseObjectSpec("mem:dsname")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "mem", DatasetName: "dsname"}, pspec)

	pspec, err = ParseObjectSpec("mem:sha1-234523542345")
	assert.NoError(err)
	assert.Equal(ObjectSpec{Protocol: "mem", Ref: "sha1-234523542345"}, pspec)

	_, err = ParseObjectSpec("http://localhost:8000/john:/why:dsname")
	assert.Error(err)

	_, err = ParseObjectSpec("hxtp://localhost:8000/john:/why:dsname")
	assert.Error(err)

	_, err = ParseObjectSpec("http::dsname")
	assert.Error(err)

	_, err = ParseObjectSpec("mem:/a/bogus/path:dsname")
	assert.Error(err)
}
