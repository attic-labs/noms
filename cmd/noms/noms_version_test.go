// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/samples/go/test_util"
	"github.com/attic-labs/testify/suite"
)

func TestVersion(t *testing.T) {
	suite.Run(t, &nomsVersionTestSuite{})
}

type nomsVersionTestSuite struct {
	test_util.ClientTestSuite
}

func (s *nomsVersionTestSuite) TestVersion() {
	val := s.Run(main, []string{"version"})
	expectedVal := fmt.Sprintf("format version: %v\nbuilt from Developer Mode\n", constants.NomsVersion)
	s.Equal(val, expectedVal)
}
