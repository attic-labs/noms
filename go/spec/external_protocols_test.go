package spec

import (
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

type testProtocol struct {
	name string
}

func (t *testProtocol) NewChunkStore(sp Spec) (chunks.ChunkStore, error) {
	t.name = sp.DatabaseName
	return chunks.NewMemoryStoreFactory().CreateStore(""), nil
}
func (t *testProtocol) NewDatabase(sp Spec) (datas.Database, error) {
	t.name = sp.DatabaseName
	cs, err := t.NewChunkStore(sp)
	d.PanicIfError(err)
	return datas.NewDatabase(cs), nil
}

func TestExternalProtocol(t *testing.T) {
	assert := assert.New(t)
	tp := &testProtocol{}
	RegisterExternalProtocol("test", tp)
	defer UnregisterExternalProtocol("test")

	sp, err := ForDataset("test:foo::bar")
	assert.NoError(err)
	assert.Equal("test", sp.Protocol)
	assert.Equal("foo", sp.DatabaseName)

	cs := sp.NewChunkStore()
	assert.Equal("foo", tp.name)
	c := chunks.NewChunk([]byte("hi!"))
	cs.Put(c)
	assert.True(cs.Has(c.Hash()))

	tp.name = ""
	ds := sp.GetDataset()
	assert.Equal("foo", tp.name)

	ds, err = ds.Database().CommitValue(ds, types.String("hi!"))
	d.PanicIfError(err)

	assert.True(types.String("hi!").Equals(ds.HeadValue()))
}

func TestExternalProtocolRegisterTwice(t *testing.T) {
	assert := assert.New(t)
	tp := &testProtocol{}
	assert.NoError(RegisterExternalProtocol("test", tp), "registering 'test' the first time should not fail")
	defer UnregisterExternalProtocol("test")
	assert.Error(RegisterExternalProtocol("test", tp), "registering 'test' the second time should fail")
}

func TestUnregisterExternalProtocol(t *testing.T) {
	assert := assert.New(t)
	tp := &testProtocol{}
	RegisterExternalProtocol("test", tp)
	assert.True(UnregisterExternalProtocol("test"), "unregistering 'test' should return true")
}

func TestUnregisterExternalProtocolNonexistent(t *testing.T) {
	assert := assert.New(t)
	assert.False(UnregisterExternalProtocol("test"), "unregistering 'test' should return false")
}
