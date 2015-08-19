package chunks

import (
	"github.com/attic-labs/noms/ref"
	"github.com/stretchr/testify/assert"
)

type ChunkStoreSuite interface {
	SetupTest()
	TearDownTest()
	Assert() *assert.Assertions
	Store() ChunkStore
	PutCountFn() func() int
}

func ChunkStoreTestCommon(suite ChunkStoreSuite) {
	suite.SetupTest()
	ChunkStoreTestPut(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestWriteAfterCloseFails(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestWriteAfterRefFails(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestPutWithRefAfterClose(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestPutWithMultipleRef(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestRoot(suite)
	suite.TearDownTest()

	suite.SetupTest()
	ChunkStoreTestGetNonExisting(suite)
	suite.TearDownTest()
}

func ChunkStoreTestPut(suite ChunkStoreSuite) {
	putCountFn := suite.PutCountFn()

	input := "abc"
	w := suite.Store().Put()
	_, err := w.Write([]byte(input))
	suite.Assert().NoError(err)
	ref := w.Ref()

	// See http://www.di-mgt.com.au/sha_testvectors.html
	suite.Assert().Equal("sha1-a9993e364706816aba3e25717850c26c9cd0d89d", ref.String())

	// And reading it via the API should work...
	assertInputInStore(input, ref, suite.Store(), suite.Assert())
	if putCountFn != nil {
		suite.Assert().Equal(1, putCountFn())
	}

	// Re-writing the same data should be idempotent and should not result in a second put
	w = suite.Store().Put()
	_, err = w.Write([]byte(input))
	suite.Assert().NoError(err)
	suite.Assert().Equal(ref, w.Ref())
	assertInputInStore(input, ref, suite.Store(), suite.Assert())

	if putCountFn != nil {
		suite.Assert().Equal(1, putCountFn())
	}
}

func ChunkStoreTestWriteAfterCloseFails(suite ChunkStoreSuite) {
	input := "abc"
	w := suite.Store().Put()
	_, err := w.Write([]byte(input))
	suite.Assert().NoError(err)

	suite.Assert().NoError(w.Close())
	suite.Assert().Panics(func() { w.Write([]byte(input)) }, "Write() after Close() should barf!")
}

func ChunkStoreTestWriteAfterRefFails(suite ChunkStoreSuite) {
	input := "abc"
	w := suite.Store().Put()
	_, err := w.Write([]byte(input))
	suite.Assert().NoError(err)

	_ = w.Ref()
	suite.Assert().NoError(err)
	suite.Assert().Panics(func() { w.Write([]byte(input)) }, "Write() after Close() should barf!")
}

func ChunkStoreTestPutWithRefAfterClose(suite ChunkStoreSuite) {
	input := "abc"
	w := suite.Store().Put()
	_, err := w.Write([]byte(input))
	suite.Assert().NoError(err)

	suite.Assert().NoError(w.Close())
	ref := w.Ref() // Ref() after Close() should work...

	// And reading the data via the API should work...
	assertInputInStore(input, ref, suite.Store(), suite.Assert())
}

func ChunkStoreTestPutWithMultipleRef(suite ChunkStoreSuite) {
	input := "abc"
	w := suite.Store().Put()
	_, err := w.Write([]byte(input))
	suite.Assert().NoError(err)

	w.Ref()
	ref := w.Ref() // Multiple calls to Ref() should work...

	// And reading the data via the API should work...
	assertInputInStore(input, ref, suite.Store(), suite.Assert())
}

func ChunkStoreTestRoot(suite ChunkStoreSuite) {
	oldRoot := suite.Store().Root()
	suite.Assert().Equal(oldRoot, ref.Ref{})

	bogusRoot := ref.Parse("sha1-81c870618113ba29b6f2b396ea3a69c6f1d626c5") // sha1("Bogus, Dude")
	newRoot := ref.Parse("sha1-907d14fb3af2b0d4f18c2d46abe8aedce17367bd")   // sha1("Hello, World")

	// Try to update root with bogus oldRoot
	result := suite.Store().UpdateRoot(newRoot, bogusRoot)
	suite.Assert().False(result)

	// Now do a valid root update
	result = suite.Store().UpdateRoot(newRoot, oldRoot)
	suite.Assert().True(result)
}

func ChunkStoreTestGetNonExisting(suite ChunkStoreSuite) {
	ref := ref.Parse("sha1-1111111111111111111111111111111111111111")
	r := suite.Store().Get(ref)
	suite.Assert().Nil(r)
}
