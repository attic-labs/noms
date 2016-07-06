// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestVersion(t *testing.T) {
	suite.Run(t, &nomsVersionTestSuite{})
}

type nomsVersionTestSuite struct {
	clienttest.ClientTestSuite
}

func (s *nomsVersionTestSuite) TestVersion() {
	val, _ := s.Run(main, []string{"version"})
	expectedVal := fmt.Sprintf("version: %v\nbuilt from %v\n", constants.NomsVersion, constants.NomsGitSHA)
	s.Equal(val, expectedVal)
}
