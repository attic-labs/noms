// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestFetch(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	clienttest.ClientTestSuite
}

func (s *testSuite) TestImportFromStdin() {
	assert := s.Assert()

	oldStdin := os.Stdin
	newStdin, blobOut, err := os.Pipe()
	assert.NoError(err)

	os.Stdin = newStdin
	defer func() {
		os.Stdin = oldStdin
	}()

	go func() {
		blobOut.Write([]byte("abcdef"))
		blobOut.Close()
	}()

	dsName := spec.CreateValueSpecString("ldb", s.LdbDir, "ds")
	// Run() will return when blobOut is closed.
	s.Run(main, []string{"--stdin", dsName})

	db, blob, err := spec.GetPath(dsName + ".value")
	assert.NoError(err)

	expected := types.NewBlob(bytes.NewBufferString("abcdef"))
	assert.True(expected.Equals(blob))

	meta := db.Head("ds").Get(datas.MetaField).(types.Struct)
	assert.Equal("stdin", string(meta.Get("file").(types.String)))

	db.Close()
}
