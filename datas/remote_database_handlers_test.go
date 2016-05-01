package datas

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

func TestHandleWriteValue(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewTestStore()
	db := NewDatabase(cs)

	l := types.NewList(
		db.WriteValue(types.Bool(true)),
		db.WriteValue(types.Bool(false)),
	)
	db.WriteValue(l)

	hint := l.Ref()
	newItem := types.NewEmptyBlob()
	itemChunk := types.EncodeValue(newItem, nil)
	l2 := l.Insert(1, types.NewTypedRefFromValue(newItem))
	listChunk := types.EncodeValue(l2, nil)

	body := &bytes.Buffer{}
	serializeHints(body, map[ref.Ref]struct{}{hint: struct{}{}})
	sz := chunks.NewSerializer(body)
	sz.Put(itemChunk)
	sz.Put(listChunk)
	sz.Close()

	w := httptest.NewRecorder()
	HandleWriteValue(w, &http.Request{Body: ioutil.NopCloser(body), Method: "POST"}, params{}, cs)

	if assert.Equal(http.StatusCreated, w.Code, "Handler error:\n%s", string(w.Body.Bytes())) {
		db2 := NewDatabase(cs)
		v := db2.ReadValue(l2.Ref())
		if assert.NotNil(v) {
			assert.True(v.Equals(l2), "%+v != %+v", v, l2)
		}
	}
}

func TestHandleWriteValueBackpressure(t *testing.T) {
	assert := assert.New(t)
	cs := &backpressureCS{ChunkStore: chunks.NewMemoryStore()}
	db := NewDatabase(cs)

	l := types.NewList(
		db.WriteValue(types.Bool(true)),
		db.WriteValue(types.Bool(false)),
	)
	db.WriteValue(l)

	hint := l.Ref()
	newItem := types.NewEmptyBlob()
	itemChunk := types.EncodeValue(newItem, nil)
	l2 := l.Insert(1, types.NewTypedRefFromValue(newItem))
	listChunk := types.EncodeValue(l2, nil)

	body := &bytes.Buffer{}
	serializeHints(body, map[ref.Ref]struct{}{hint: struct{}{}})
	sz := chunks.NewSerializer(body)
	sz.Put(itemChunk)
	sz.Put(listChunk)
	sz.Close()

	w := httptest.NewRecorder()
	HandleWriteValue(w, &http.Request{Body: ioutil.NopCloser(body), Method: "POST"}, params{}, cs)

	if assert.Equal(httpStatusTooManyRequests, w.Code, "Handler error:\n%s", string(w.Body.Bytes())) {
		hashes := deserializeHashes(w.Body)
		assert.Len(hashes, 1)
		assert.Equal(l2.Ref(), hashes[0])
	}
}

func TestBuildWriteValueRequest(t *testing.T) {
	assert := assert.New(t)
	input1, input2 := "abc", "def"
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte(input1)),
		chunks.NewChunk([]byte(input2)),
	}

	hints := map[ref.Ref]struct{}{
		ref.Parse("sha1-0000000000000000000000000000000000000002"): struct{}{},
		ref.Parse("sha1-0000000000000000000000000000000000000003"): struct{}{},
	}
	compressed := buildWriteValueRequest(serializeChunks(chnx, assert), hints)
	gr, err := gzip.NewReader(compressed)
	d.Exp.NoError(err)
	defer gr.Close()

	count := 0
	for hint := range deserializeHints(gr) {
		count++
		_, present := hints[hint]
		assert.True(present)
	}
	assert.Equal(len(hints), count)

	chunkChan := make(chan chunks.Chunk, 16)
	go chunks.DeserializeToChan(gr, chunkChan)
	for c := range chunkChan {
		assert.Equal(chnx[0].Ref(), c.Ref())
		chnx = chnx[1:]
	}
	assert.Empty(chnx)
}

func serializeChunks(chnx []chunks.Chunk, assert *assert.Assertions) io.Reader {
	body := &bytes.Buffer{}
	gw := gzip.NewWriter(body)
	sz := chunks.NewSerializer(gw)
	assert.NoError(sz.PutMany(chnx))
	assert.NoError(sz.Close())
	assert.NoError(gw.Close())
	return body
}

func TestBuildGetRefsRequest(t *testing.T) {
	assert := assert.New(t)
	refs := map[ref.Ref]struct{}{
		ref.Parse("sha1-0000000000000000000000000000000000000002"): struct{}{},
		ref.Parse("sha1-0000000000000000000000000000000000000003"): struct{}{},
	}
	r := buildGetRefsRequest(refs)
	b, err := ioutil.ReadAll(r)
	assert.NoError(err)

	urlValues, err := url.ParseQuery(string(b))
	assert.NoError(err)
	assert.NotEmpty(urlValues)

	queryRefs := urlValues["ref"]
	assert.Len(queryRefs, len(refs))
	for _, r := range queryRefs {
		_, present := refs[ref.Parse(r)]
		assert.True(present, "Query contains %s, which is not in initial refs", r)
	}
}

func TestHandleGetRefs(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewTestStore()
	input1, input2 := "abc", "def"
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte(input1)),
		chunks.NewChunk([]byte(input2)),
	}
	err := cs.PutMany(chnx)
	assert.NoError(err)

	body := strings.NewReader(fmt.Sprintf("ref=%s&ref=%s", chnx[0].Ref(), chnx[1].Ref()))

	w := httptest.NewRecorder()
	HandleGetRefs(w,
		&http.Request{Body: ioutil.NopCloser(body), Method: "POST", Header: http.Header{
			"Content-Type": {"application/x-www-form-urlencoded"},
		}},
		params{},
		cs,
	)

	if assert.Equal(http.StatusOK, w.Code, "Handler error:\n%s", string(w.Body.Bytes())) {
		chunkChan := make(chan chunks.Chunk)
		go chunks.DeserializeToChan(w.Body, chunkChan)
		for c := range chunkChan {
			assert.Equal(chnx[0].Ref(), c.Ref())
			chnx = chnx[1:]
		}
		assert.Empty(chnx)
	}
}

func TestHandleGetRoot(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewTestStore()
	c := chunks.NewChunk([]byte("abc"))
	cs.Put(c)
	assert.True(cs.UpdateRoot(c.Ref(), ref.Ref{}))

	w := httptest.NewRecorder()
	HandleRootGet(w, &http.Request{Method: "GET"}, params{}, cs)

	if assert.Equal(http.StatusOK, w.Code, "Handler error:\n%s", string(w.Body.Bytes())) {
		root := ref.Parse(string(w.Body.Bytes()))
		assert.Equal(c.Ref(), root)
	}
}

func TestHandlePostRoot(t *testing.T) {
	assert := assert.New(t)
	cs := chunks.NewTestStore()
	input1, input2 := "abc", "def"
	chnx := []chunks.Chunk{
		chunks.NewChunk([]byte(input1)),
		chunks.NewChunk([]byte(input2)),
	}
	err := cs.PutMany(chnx)
	assert.NoError(err)

	// First attempt should fail, as 'last' won't match.
	u := &url.URL{}
	queryParams := url.Values{}
	queryParams.Add("last", chnx[0].Ref().String())
	queryParams.Add("current", chnx[1].Ref().String())
	u.RawQuery = queryParams.Encode()

	w := httptest.NewRecorder()
	HandleRootPost(w, &http.Request{URL: u, Method: "POST"}, params{}, cs)
	assert.Equal(http.StatusConflict, w.Code, "Handler error:\n%s", string(w.Body.Bytes()))

	// Now, update the root manually to 'last' and try again.
	assert.True(cs.UpdateRoot(chnx[0].Ref(), ref.Ref{}))
	w = httptest.NewRecorder()
	HandleRootPost(w, &http.Request{URL: u, Method: "POST"}, params{}, cs)
	assert.Equal(http.StatusOK, w.Code, "Handler error:\n%s", string(w.Body.Bytes()))
}

type params map[string]string

func (p params) ByName(k string) string {
	return p[k]
}
