package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

var datasetID = "testdataset"

func createTestStore(assert *assert.Assertions) chunks.ChunkStore {
	ms := &chunks.MemoryStore{}
	datasetDs := dataset.NewDataset(datas.NewDataStore(ms), datasetID)
	datasetValue := types.NewString("Value for " + datasetID)
	datasetDs, ok := datasetDs.Commit(
		datas.NewCommit().SetParents(
			datas.NewSetOfCommit().Insert(datasetDs.Head()).NomsValue()).SetValue(datasetValue))
	assert.True(ok)
	return ms
}

func TestBadRequest(t *testing.T) {
	assert := assert.New(t)

	req, _ := http.NewRequest("GET", "/bad", nil)
	w := httptest.NewRecorder()

	ms := &chunks.MemoryStore{}
	s := server{ms}
	s.handle(w, req)
	assert.Equal(w.Code, http.StatusBadRequest)
}

func TestRoot(t *testing.T) {
	assert := assert.New(t)

	req, _ := http.NewRequest("GET", "/root", nil)
	w := httptest.NewRecorder()
	ms := createTestStore(assert)
	s := server{ms}
	s.handle(w, req)
	assert.Equal(w.Code, http.StatusOK)
	ref := ref.Parse(w.Body.String())
	assert.Equal(ms.Root(), ref)
}

func TestGetRef(t *testing.T) {
	assert := assert.New(t)

	ms := createTestStore(assert)
	rootRef := ms.Root().String()

	req, _ := http.NewRequest("GET", "/get?ref="+rootRef, nil)
	w := httptest.NewRecorder()
	s := server{ms}
	s.handle(w, req)
	assert.Equal(w.Code, http.StatusOK)
	assert.Equal(`j {"map":["parents",{"ref":"sha1-e7a6cc434e244d62262786678197643397c8139e"},"value",{"ref":"sha1-b0b44852c7048beab261086a135a8eda4c3e11c8"},"$name","Commit"]}
`, w.Body.String())
}

func TestGetInvalidRef(t *testing.T) {
	assert := assert.New(t)

	ms := createTestStore(assert)
	rootRef := "sha1-xxx"

	req, _ := http.NewRequest("GET", "/get?ref="+rootRef, nil)
	w := httptest.NewRecorder()
	s := server{ms}
	s.handle(w, req)
	assert.Equal(w.Code, http.StatusBadRequest)
}

func TestGetNonExistingRef(t *testing.T) {
	assert := assert.New(t)

	ms := createTestStore(assert)
	ref := "sha1-1111111111111111111111111111111111111111"

	req, _ := http.NewRequest("GET", "/get?ref="+ref, nil)
	w := httptest.NewRecorder()
	s := server{ms}
	s.handle(w, req)
	assert.Equal(w.Code, http.StatusNotFound)
}
