// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

func TestLDBDatabase(t *testing.T) {
	assert := assert.New(t)

	d1 := os.TempDir()
	dir, err := ioutil.TempDir(d1, "flags")
	assert.NoError(err)
	ldbDir := path.Join(dir, "store")
	spec := fmt.Sprintf("ldb:%s", path.Join(dir, "store"))

	cs := chunks.NewLevelDBStoreUseFlags(ldbDir, "")
	ds := datas.NewDatabase(cs)

	s1 := types.String("A String")
	s1Hash := ds.WriteValue(s1)
	ds.Commit("testDs", datas.NewCommit().Set(datas.ValueField, s1Hash))
	ds.Close()

	sp, errRead := parseDatabaseSpec(spec)
	assert.NoError(errRead)
	store, err := sp.Database()
	assert.NoError(err)
	assert.Equal(s1, store.ReadValue(s1.Hash()))
	store.Close()
	os.Remove(dir)
}

func TestMemDatabase(t *testing.T) {
	assert := assert.New(t)

	spec := "mem"
	sp, err := parseDatabaseSpec(spec)
	assert.NoError(err)
	store, err := sp.Database()
	assert.NoError(err)
	r := store.WriteValue(types.Bool(true))

	assert.NoError(err)
	assert.Equal(types.Bool(true), store.ReadValue(r.TargetHash()))
}

func TestMemDataset(t *testing.T) {
	assert := assert.New(t)

	spec := "mem::datasetTest"
	sp1, rem, err := parseDatasetSpec(spec)
	assert.Equal("", rem)
	assert.NoError(err)
	dataset1, err := sp1.Dataset()
	assert.NoError(err)
	commit := types.String("Commit Value")
	dsTest, err := dataset1.Commit(commit)
	assert.NoError(err)
	assert.EqualValues(commit, dsTest.HeadValue())
}

func TestLDBDataset(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)
	ldbPath := path.Join(dir, "name")
	cs := chunks.NewLevelDBStoreUseFlags(ldbPath, "")
	ds := datas.NewDatabase(cs)
	id := "dsName"

	set := dataset.NewDataset(ds, id)
	commit := types.String("Commit Value")
	set, err = set.Commit(commit)
	assert.NoError(err)
	ds.Close()

	spec := fmt.Sprintf("ldb:%s::%s", ldbPath, id)
	sp, rem, err := parseDatasetSpec(spec)
	assert.Equal("", rem)
	assert.NoError(err)
	dataset, err := sp.Dataset()
	assert.NoError(err)
	assert.EqualValues(commit, dataset.HeadValue())

	os.Remove(dir)
}

func TestLDBObject(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)
	ldbpath := path.Join(dir, "xx-yy")
	dsId := "dsId"

	cs1 := chunks.NewLevelDBStoreUseFlags(ldbpath, "")
	store1 := datas.NewDatabase(cs1)
	dataset1 := dataset.NewDataset(store1, dsId)
	s1 := types.String("Commit Value")
	r1 := store1.WriteValue(s1)
	_, err = dataset1.Commit(r1)
	assert.NoError(err)
	store1.Close()

	spec2 := fmt.Sprintf("ldb:%s::%s", ldbpath, dsId)
	assert.NoError(err)
	sp1, rem, err := parseDatasetSpec(spec2)
	assert.Equal("", rem)
	assert.NoError(err)
	dataset2, err := sp1.Dataset()
	assert.NoError(err)
	r2 := dataset2.HeadValue()
	s2 := r2.(types.Ref).TargetValue(dataset2.Database())
	assert.Equal(s1, s2)
	dataset2.Database().Close()

	spec3 := fmt.Sprintf("ldb:%s::#%s", ldbpath, s1.Hash().String())
	sp3, err := parseValueSpec(spec3)
	assert.NoError(err)
	database, v3, err := sp3.Value()
	assert.NoError(err)
	assert.Equal(s1, v3)
	database.Close()
}

func TestReadHash(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)
	datasetId := "dsName"

	ldbPath := path.Join(dir, "/name")
	cs1 := chunks.NewLevelDBStoreUseFlags(ldbPath, "")
	database1 := datas.NewDatabase(cs1)
	dataset1 := dataset.NewDataset(database1, datasetId)
	commit := types.String("Commit Value")
	dataset1, err = dataset1.Commit(commit)
	assert.NoError(err)
	r1 := dataset1.Head().Hash()
	dataset1.Database().Close()

	spec2 := fmt.Sprintf("ldb:%s::#%s", ldbPath, r1.String())
	sp2, err := parseValueSpec(spec2)
	assert.NoError(err)
	database, v2, err := sp2.Value()
	assert.NoError(err)

	assert.EqualValues(r1.String(), v2.Hash().String())
	database.Close()
}

func TestdatabaseSpecs(t *testing.T) {
	assert := assert.New(t)

	badSpecs := []string{"mem:stuff", "mem:", "http:", "https:", "random:", "random:random", "http://some.com/wi::rd/path", "/file/ba:d"}
	for _, spec := range badSpecs {
		_, err := parseDatabaseSpec(spec)
		assert.Error(err)
	}

	type testCase struct {
		spec, scheme, path, accessToken string
	}

	testCases := []testCase{
		testCase{"http://localhost:8000", "http", "//localhost:8000", ""},
		testCase{"http://localhost:8000/fff", "http", "//localhost:8000/fff", ""},
		testCase{"https://local.attic.io/john/doe", "https", "//local.attic.io/john/doe", ""},
		testCase{"ldb:/filesys/john/doe", "ldb", "/filesys/john/doe", ""},
		testCase{"./john/doe", "ldb", "./john/doe", ""},
		testCase{"john/doe", "ldb", "john/doe", ""},
		testCase{"/john/doe", "ldb", "/john/doe", ""},
		testCase{"mem", "mem", "", ""},
		testCase{"http://server.com/john/doe?access_token=jane", "http", "//server.com/john/doe?access_token=jane", "jane"},
		testCase{"https://server.com/john/doe/?arg=2&qp1=true&access_token=jane", "https", "//server.com/john/doe/?arg=2&qp1=true&access_token=jane", "jane"},
	}

	for _, tc := range testCases {
		dbSpec, err := parseDatabaseSpec(tc.spec)
		assert.NoError(err)
		assert.Equal(databaseSpec{Protocol: tc.scheme, Path: tc.path, accessToken: tc.accessToken}, dbSpec)
	}
}

func TestdatasetSpecs(t *testing.T) {
	assert := assert.New(t)
	badSpecs := []string{"mem", "mem:", "mem:::ds", "http", "http:", "http://foo", "monkey", "monkey:balls", "http::dsname", "mem:/a/bogus/path:dsname", "http://localhost:8000/one"}

	for _, spec := range badSpecs {
		_, _, err := parseDatasetSpec(spec)
		assert.Error(err)
	}

	invalidDatasetNames := []string{" ", "", "$", "#", ":", "\n", "💩"}
	for _, s := range invalidDatasetNames {
		_, _, err := parseDatasetSpec("mem::" + s)
		assert.Error(err)
	}

	validDatasetNames := []string{"a", "Z", "0", "/", "-", "_"}
	for _, s := range validDatasetNames {
		_, _, err := parseDatasetSpec("mem::" + s)
		assert.NoError(err)
	}

	type testCase struct {
		spec, scheme, path, ds, accessToken string
	}

	testCases := []testCase{
		testCase{"http://localhost:8000::ds1", "http", "//localhost:8000", "ds1", ""},
		testCase{"http://localhost:8000/john/doe/::ds2", "http", "//localhost:8000/john/doe/", "ds2", ""},
		testCase{"https://local.attic.io/john/doe::ds3", "https", "//local.attic.io/john/doe", "ds3", ""},
		testCase{"http://local.attic.io/john/doe::ds1", "http", "//local.attic.io/john/doe", "ds1", ""},
		testCase{"ldb:/filesys/john/doe::ds/one", "ldb", "/filesys/john/doe", "ds/one", ""},
		testCase{"http://localhost:8000/john/doe?access_token=abc::ds/one", "http", "//localhost:8000/john/doe?access_token=abc", "ds/one", "abc"},
		testCase{"https://localhost:8000?qp1=x&access_token=abc&qp2=y::ds/one", "https", "//localhost:8000?qp1=x&access_token=abc&qp2=y", "ds/one", "abc"},
	}

	for _, tc := range testCases {
		dsSpec, rem, err := parseDatasetSpec(tc.spec)
		assert.Equal("", rem)
		assert.NoError(err)
		dbSpec1 := databaseSpec{Protocol: tc.scheme, Path: tc.path, accessToken: tc.accessToken}
		assert.Equal(datasetSpec{DbSpec: dbSpec1, DatasetName: tc.ds}, dsSpec)
	}
}

func TesthashSpec(t *testing.T) {
	assert := assert.New(t)

	badSpecs := []string{"mem::#", "mem::#s", "mem::#sha1-foobarbaz", "mem::#sha1-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}
	for _, bs := range badSpecs {
		_, _, err := parseHashSpec(bs)
		assert.Error(err)
	}

	type testCase struct {
		spec, scheme, path, hash string
	}

	testCases := []testCase{
		testCase{"http://local.attic.io/john/doe::#sha1-0123456789012345678901234567890123456789", "http", "//local.attic.io/john/doe", "sha1-0123456789012345678901234567890123456789"},
		testCase{"ldb:/filesys/john/doe::#sha1-0123456789012345678901234567890123456789", "ldb", "/filesys/john/doe", "sha1-0123456789012345678901234567890123456789"},
		testCase{"mem::#sha1-0123456789012345678901234567890123456789", "mem", "", "sha1-0123456789012345678901234567890123456789"},
	}

	for _, tc := range testCases {
		spec, rem, err := parseHashSpec(tc.spec)
		assert.Equal("", rem)
		assert.NoError(err)
		dbSpec1 := databaseSpec{Protocol: tc.scheme, Path: tc.path, accessToken: ""}
		assert.Equal(hashSpec{DbSpec: dbSpec1, Hash: hash.Parse(tc.hash)}, spec)
	}
}

func TestpathSpec(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		spec, scheme, path, ds, hash string
	}

	testCases := []testCase{
		testCase{"http://local.attic.io/john/doe::#sha1-0123456789012345678901234567890123456789", "http", "//local.attic.io/john/doe", "", "sha1-0123456789012345678901234567890123456789"},
		testCase{"http://localhost:8000/john/doe/::ds1", "http", "//localhost:8000/john/doe/", "ds1", ""},
	}

	for _, tc := range testCases {
		pathSpec, err := parseValueSpec(tc.spec)
		assert.NoError(err)
		dbSpec1 := databaseSpec{Protocol: tc.scheme, Path: tc.path, accessToken: ""}
		if tc.hash != "" {
			assert.Equal(hashSpec{DbSpec: dbSpec1, Hash: hash.Parse(tc.hash)}, pathSpec.(hashSpec))
		} else {
			assert.Equal(datasetSpec{DbSpec: dbSpec1, DatasetName: tc.ds}, pathSpec.(datasetSpec))
		}
	}

	_, err := parseValueSpec("http://local.attic.io")
	assert.Error(err)
}

func TestFullPaths(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "")
	assert.NoError(err)
	ldbPath := path.Join(dir, "TestFullPaths")
	cs := chunks.NewLevelDBStoreUseFlags(ldbPath, "")
	db := datas.NewDatabase(cs)
	ds := dataset.NewDataset(db, "ds")

	s0, s1 := types.String("foo"), types.String("bar")
	list := types.NewList(s0, s1)

	// Make values addressable.
	db.WriteValue(s0)
	db.WriteValue(s1)
	db.WriteValue(list)

	_, err = ds.Commit(list)
	assert.NoError(err)
	db.Close()

	getPath := func(path string, a ...interface{}) types.Value {
		db, val, _ := GetValue(fmt.Sprintf("ldb:%s::%s", ldbPath, fmt.Sprintf(path, a...)))
		if db != nil {
			db.Close()
		}
		return val
	}

	assert.NotNil(list, getPath("ds"))
	assert.Equal(uint64(0), getPath("ds.parents").(types.Set).Len())
	assert.True(list.Equals(getPath("ds.value")))
	assert.True(s0.Equals(getPath("ds.value[0]")))
	assert.True(s1.Equals(getPath("ds.value[1]")))
	assert.True(list.Equals(getPath("#%s", list.Hash().String())))
	assert.True(s0.Equals(getPath("#%s", s0.Hash().String())))
	assert.True(s1.Equals(getPath("#%s", s1.Hash().String())))
	assert.True(s0.Equals(getPath("#%s[0]", list.Hash().String())))
	assert.True(s1.Equals(getPath("#%s[1]", list.Hash().String())))

	assert.Nil(getPath("foo"))
	assert.Nil(getPath("foo.parents"))
	assert.Nil(getPath("foo.value"))
	assert.Nil(getPath("foo.value[0]"))
	assert.Nil(getPath("#%s", types.String("baz").Hash()))
	assert.Nil(getPath("#%s[0]", types.String("baz").Hash()))
}
