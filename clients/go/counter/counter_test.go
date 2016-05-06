package main

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/clients/go/test_util"
	"github.com/stretchr/testify/suite"
)

func TestCounter(t *testing.T) {
	suite.Run(t, &counterTestSuite{})
}

type counterTestSuite struct {
	test_util.ClientTestSuite
}

func (s *counterTestSuite) TestCounter() {
	spec := fmt.Sprintf("ldb:%s:%s", s.LdbDir, "counter")
	args := []string{spec}
	s.Equal("1\n", s.Run(main, args))
	s.Equal("2\n", s.Run(main, args))
	s.Equal("3\n", s.Run(main, args))
}
