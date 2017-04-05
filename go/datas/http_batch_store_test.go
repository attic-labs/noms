// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/suite"
	"github.com/julienschmidt/httprouter"
)

const testAuthToken = "aToken123"

func TestHTTPBatchStore(t *testing.T) {
	suite.Run(t, &HTTPBatchStoreSuite{})
}

type HTTPBatchStoreSuite struct {
	suite.Suite
	cs    *chunks.TestStore
	store *httpBatchStore
}

type inlineServer struct {
	*httprouter.Router
}

func (serv inlineServer) Do(req *http.Request) (resp *http.Response, err error) {
	w := httptest.NewRecorder()
	serv.ServeHTTP(w, req)
	return &http.Response{
			StatusCode: w.Code,
			Status:     http.StatusText(w.Code),
			Header:     w.HeaderMap,
			Body:       ioutil.NopCloser(w.Body),
		},
		nil
}

func (suite *HTTPBatchStoreSuite) SetupTest() {
	suite.cs = chunks.NewTestStore()
	suite.store = NewHTTPBatchStoreForTest(suite.cs)
}

func NewHTTPBatchStoreForTest(cs chunks.ChunkStore) *httpBatchStore {
	serv := inlineServer{httprouter.New()}
	serv.POST(
		constants.WriteValuePath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleWriteValue(w, req, ps, cs)
		},
	)
	serv.POST(
		constants.GetRefsPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleGetRefs(w, req, ps, cs)
		},
	)
	serv.POST(
		constants.HasRefsPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleHasRefs(w, req, ps, cs)
		},
	)
	serv.POST(
		constants.RootPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleRootPost(w, req, ps, cs)
		},
	)
	serv.GET(
		constants.RootPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleRootGet(w, req, ps, cs)
		},
	)
	hcs := NewHTTPBatchStore("http://localhost:9000", "")
	hcs.httpClient = serv
	return hcs
}

func newAuthenticatingHTTPBatchStoreForTest(suite *HTTPBatchStoreSuite, hostUrl string) *httpBatchStore {
	authenticate := func(req *http.Request) {
		suite.Equal(testAuthToken, req.URL.Query().Get("access_token"))
	}

	serv := inlineServer{httprouter.New()}
	serv.POST(
		constants.RootPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			authenticate(req)
			HandleRootPost(w, req, ps, suite.cs)
		},
	)
	hcs := NewHTTPBatchStore(hostUrl, "")
	hcs.httpClient = serv
	return hcs
}

func newBadVersionHTTPBatchStoreForTest(suite *HTTPBatchStoreSuite) *httpBatchStore {
	serv := inlineServer{httprouter.New()}
	serv.POST(
		constants.RootPath,
		func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			HandleRootPost(w, req, ps, suite.cs)
			w.Header().Set(NomsVersionHeader, "BAD")
		},
	)
	hcs := NewHTTPBatchStore("http://localhost", "")
	hcs.httpClient = serv
	return hcs
}

func (suite *HTTPBatchStoreSuite) TearDownTest() {
	suite.store.Close()
	suite.cs.Close()
}

func (suite *HTTPBatchStoreSuite) TestPutChunk() {
	c := types.EncodeValue(types.String("abc"), nil)
	suite.store.SchedulePut(c, 1)
	suite.store.Flush()

	suite.Equal(1, suite.cs.Writes)
}

func (suite *HTTPBatchStoreSuite) TestPutChunksInOrder() {
	vals := []types.Value{
		types.String("abc"),
		types.String("def"),
	}
	l := types.NewList()
	for _, val := range vals {
		suite.store.SchedulePut(types.EncodeValue(val, nil), 1)
		l = l.Append(types.NewRef(val))
	}
	suite.store.SchedulePut(types.EncodeValue(l, nil), 2)
	suite.store.Flush()

	suite.Equal(3, suite.cs.Writes)
}

func (suite *HTTPBatchStoreSuite) TestPutChunksReverseOrder() {
	val := types.String("abc")
	l := types.NewList(types.NewRef(val))

	suite.store.SchedulePut(types.EncodeValue(l, nil), 2)
	suite.store.SchedulePut(types.EncodeValue(val, nil), 1)
	suite.store.SetReverseFlushOrder()
	suite.store.Flush()

	suite.Equal(2, suite.cs.Writes)
}

func (suite *HTTPBatchStoreSuite) TestRoot() {
	c := types.EncodeValue(types.NewMap(), nil)
	suite.cs.Put(c)
	suite.True(suite.store.UpdateRoot(c.Hash(), hash.Hash{}))
	suite.Equal(c.Hash(), suite.cs.Root())
}

func (suite *HTTPBatchStoreSuite) TestVersionMismatch() {
	store := newBadVersionHTTPBatchStoreForTest(suite)
	defer store.Close()
	c := types.EncodeValue(types.NewMap(), nil)
	suite.cs.Put(c)
	suite.Panics(func() { store.UpdateRoot(c.Hash(), hash.Hash{}) })
}

func (suite *HTTPBatchStoreSuite) TestUpdateRoot() {
	c := types.EncodeValue(types.NewMap(), nil)
	suite.cs.Put(c)
	suite.True(suite.store.UpdateRoot(c.Hash(), hash.Hash{}))
	suite.Equal(c.Hash(), suite.cs.Root())
}

func (suite *HTTPBatchStoreSuite) TestUpdateRootWithParams() {
	u := fmt.Sprintf("http://localhost:9000?access_token=%s&other=19", testAuthToken)
	store := newAuthenticatingHTTPBatchStoreForTest(suite, u)
	defer store.Close()
	c := types.EncodeValue(types.NewMap(), nil)
	suite.cs.Put(c)
	suite.True(store.UpdateRoot(c.Hash(), hash.Hash{}))
	suite.Equal(c.Hash(), suite.cs.Root())
}

func (suite *HTTPBatchStoreSuite) TestGet() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	suite.cs.PutMany(chnx)
	got := suite.store.Get(chnx[0].Hash())
	suite.Equal(chnx[0].Hash(), got.Hash())
	got = suite.store.Get(chnx[1].Hash())
	suite.Equal(chnx[1].Hash(), got.Hash())
}

func (suite *HTTPBatchStoreSuite) TestGetMany() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	notPresent := chunks.NewChunk([]byte("ghi")).Hash()
	suite.cs.PutMany(chnx)

	hashes := hash.NewHashSet(chnx[0].Hash(), chnx[1].Hash(), notPresent)
	foundChunks := make(chan *chunks.Chunk)
	go func() { suite.store.GetMany(hashes, foundChunks); close(foundChunks) }()

	for c := range foundChunks {
		hashes.Remove(c.Hash())
	}
	suite.Len(hashes, 1)
	suite.True(hashes.Has(notPresent))
}

func (suite *HTTPBatchStoreSuite) TestGetManyAllCached() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	suite.store.SchedulePut(chnx[0], 1)
	suite.store.SchedulePut(chnx[1], 1)

	hashes := hash.NewHashSet(chnx[0].Hash(), chnx[1].Hash())
	foundChunks := make(chan *chunks.Chunk)
	go func() { suite.store.GetMany(hashes, foundChunks); close(foundChunks) }()

	for c := range foundChunks {
		hashes.Remove(c.Hash())
	}
	suite.Len(hashes, 0)
}

func (suite *HTTPBatchStoreSuite) TestGetManySomeCached() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	cached := chunks.NewChunk([]byte("ghi"))
	suite.cs.PutMany(chnx)
	suite.store.SchedulePut(cached, 1)

	hashes := hash.NewHashSet(chnx[0].Hash(), chnx[1].Hash(), cached.Hash())
	foundChunks := make(chan *chunks.Chunk)
	go func() { suite.store.GetMany(hashes, foundChunks); close(foundChunks) }()

	for c := range foundChunks {
		hashes.Remove(c.Hash())
	}
	suite.Len(hashes, 0)
}

func (suite *HTTPBatchStoreSuite) TestGetSame() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("def")),
		chunks.NewChunk([]byte("def")),
	}
	suite.cs.PutMany(chnx)
	got := suite.store.Get(chnx[0].Hash())
	suite.Equal(chnx[0].Hash(), got.Hash())
	got = suite.store.Get(chnx[1].Hash())
	suite.Equal(chnx[1].Hash(), got.Hash())
}

func (suite *HTTPBatchStoreSuite) TestHas() {
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte("abc")),
		chunks.NewChunk([]byte("def")),
	}
	suite.cs.PutMany(chnx)
	suite.True(suite.store.Has(chnx[0].Hash()))
	suite.True(suite.store.Has(chnx[1].Hash()))
}
