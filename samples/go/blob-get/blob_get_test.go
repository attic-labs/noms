// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/attic-labs/testify/suite"
	"gopkg.in/attic-labs/noms.v7/go/spec"
	"gopkg.in/attic-labs/noms.v7/go/types"
	"gopkg.in/attic-labs/noms.v7/go/util/clienttest"
)

func TestBlobGet(t *testing.T) {
	suite.Run(t, &bgSuite{})
}

type bgSuite struct {
	clienttest.ClientTestSuite
}

func (s *bgSuite) TestBlobGet() {
	blobBytes := []byte("hello")
	blob := types.NewBlob(bytes.NewBuffer(blobBytes))

	sp, err := spec.ForDatabase(s.TempDir)
	s.NoError(err)
	defer sp.Close()
	db := sp.GetDatabase()
	ref := db.WriteValue(blob)
	_, err = db.CommitValue(db.GetDataset("datasetID"), ref)
	s.NoError(err)

	hashSpec := fmt.Sprintf("%s::#%s", s.TempDir, ref.TargetHash().String())
	filePath := filepath.Join(s.TempDir, "out")
	s.MustRun(main, []string{hashSpec, filePath})

	fileBytes, err := ioutil.ReadFile(filePath)
	s.NoError(err)
	s.Equal(blobBytes, fileBytes)

	stdout, _ := s.MustRun(main, []string{hashSpec})
	fmt.Println("stdout:", stdout)
	s.Equal(blobBytes, []byte(stdout))
}
