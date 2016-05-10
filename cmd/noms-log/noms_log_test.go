package main

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/clients/go/flags"
	"github.com/attic-labs/noms/clients/go/test_util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestNomsShow(t *testing.T) {
	suite.Run(t, &nomsShowTestSuite{})
}

type nomsShowTestSuite struct {
	test_util.ClientTestSuite
}

func testCommitInResults(s *nomsShowTestSuite, spec string, i int) {
	sp, err := flags.ParseDatasetSpec(spec)
	s.NoError(err)
	ds, err := sp.Dataset()
	s.NoError(err)
	ds, err = ds.Commit(types.Number(1))
	s.NoError(err)
	commit := ds.Head()
	fmt.Printf("commit ref: %s, type: %s\n", commit.Ref(), commit.Type().Name())
	ds.Store().Close()
	s.Contains(s.Run(main, []string{spec}), commit.Ref().String())
}

func (s *nomsShowTestSuite) TestNomsLog() {
	datasetName := "dsTest"
	spec := fmt.Sprintf("ldb:%s:%s", s.LdbDir, datasetName)
	sp, err := flags.ParseDatasetSpec(spec)
	d.Chk.NoError(err)

	ds, err := sp.Dataset()
	d.Chk.NoError(err)
	ds.Store().Close()
	s.Equal("", s.Run(main, []string{spec}))

	testCommitInResults(s, spec, 1)
	testCommitInResults(s, spec, 2)
}

func TestTruncateLines(t *testing.T) {
	assert := assert.New(t)
	t1 := "one"
	s1 := truncateLines(t1, 10)
	assert.Equal([]string{"one"}, s1)

	t2 := "one\ntwo\nthree\nfour\nfive\nsix\nseven\n"
	s1 = truncateLines(t2, 3)
	assert.Equal([]string{"one", "two", "three"}, s1)

	s1 = truncateLines(t2, 10)
	assert.Equal([]string{"one", "two", "three", "four", "five", "six", "seven"}, s1)

	s1 = truncateLines(t2, 0)
	assert.Empty(s1)
}
