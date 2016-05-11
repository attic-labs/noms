package main

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/clients/go/flags"
	"github.com/attic-labs/noms/clients/go/test_util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
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

func addCommit(ds dataset.Dataset, v string) (dataset.Dataset, error) {
	return ds.Commit(types.NewString(v))
}

func addBranchedDataset(newDs, parentDs dataset.Dataset, v string) (dataset.Dataset, error) {
	return newDs.CommitWithParents(types.NewString(v), datas.NewSetOfRefOfCommit().Insert(parentDs.HeadRef()))
}

func mergeDatasets(ds1, ds2 dataset.Dataset, v string) (dataset.Dataset, error) {
	return ds1.CommitWithParents(types.NewString(v), datas.NewSetOfRefOfCommit().Insert(ds1.HeadRef(), ds2.HeadRef()))
}

func (s *nomsShowTestSuite) TestNomsGraph1() {
	spec := fmt.Sprintf("ldb:%s", s.LdbDir)
	dbSpec, err := flags.ParseDatabaseSpec(spec)
	d.Chk.NoError(err)
	db, err := dbSpec.Database()
	d.Chk.NoError(err)

	b1 := dataset.NewDataset(db, "b1")

	b1, err = addCommit(b1, "1")
	d.Chk.NoError(err)
	b1, err = addCommit(b1, "2")
	d.Chk.NoError(err)
	b1, err = addCommit(b1, "3")
	d.Chk.NoError(err)

	b2 := dataset.NewDataset(db, "b2")
	b2, err = addBranchedDataset(b2, b1, "3.1")
	d.Chk.NoError(err)

	b1, err = addCommit(b1, "3.2")
	d.Chk.NoError(err)
	b1, err = addCommit(b1, "3.6")
	d.Chk.NoError(err)

	b3 := dataset.NewDataset(db, "b3")
	b3, err = addBranchedDataset(b3, b2, "3.1.3")
	d.Chk.NoError(err)
	b3, err = addCommit(b3, "3.1.5")
	d.Chk.NoError(err)
	b3, err = addCommit(b3, "3.1.7")
	d.Chk.NoError(err)

	b2, err = mergeDatasets(b2, b3, "3.5")
	d.Chk.NoError(err)
	b2, err = addCommit(b2, "3.7")
	d.Chk.NoError(err)

	b1, err = mergeDatasets(b1, b2, "4")
	d.Chk.NoError(err)

	b1, err = addCommit(b1, "5")
	d.Chk.NoError(err)
	b1, err = addCommit(b1, "6")
	d.Chk.NoError(err)
	b1, err = addCommit(b1, "7")
	d.Chk.NoError(err)

	b1.Store().Close()
	s.Equal(graphRes1, s.Run(main, []string{"-graph", spec + ":b1"}))
}

func (s *nomsShowTestSuite) TestNomsGraph2() {
	spec := fmt.Sprintf("ldb:%s", s.LdbDir)
	dbSpec, err := flags.ParseDatabaseSpec(spec)
	d.Chk.NoError(err)
	db, err := dbSpec.Database()
	d.Chk.NoError(err)

	ba := dataset.NewDataset(db, "ba")

	ba, err = addCommit(ba, "1")
	d.Chk.NoError(err)

	bb := dataset.NewDataset(db, "bb")
	bb, err = addCommit(bb, "10")
	d.Chk.NoError(err)

	bc := dataset.NewDataset(db, "bc")
	bc, err = addCommit(bc, "100")
	d.Chk.NoError(err)

	ba, err = mergeDatasets(ba, bb, "11")
	d.Chk.NoError(err)

	ba, err = mergeDatasets(ba, bc, "101")
	d.Chk.NoError(err)

	db.Close()
	s.Equal(graphRes2, s.Run(main, []string{"-graph", spec + ":ba"}))
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

const (
	graphRes1 = "* sha1-722bf19c3412e46fbc78edfa94a009b6dac17bd8\n| Parent: sha1-8018223be9e73326d4" +
		"a82b072aa91913750a9c21\n| \"7\"\n| \n* sha1-8018223be9e73326d4a82b072aa91913750a9c21\n| Parent: sha" +
		"1-9fe245f3561fd5f80cdb58fcc03e4179c0cad700\n| \"6\"\n| \n* sha1-9fe245f3561fd5f80cdb58fcc03e4179c0c" +
		"ad700\n| Parent: sha1-5d116063509d38e68c4bd528dfda4d9de741416c\n| \"5\"\n| \n*   sha1-5d116063509d3" +
		"8e68c4bd528dfda4d9de741416c\n|\\  Merge: sha1-ee04cc1dda6b010ca3488e4996d14a54176b169d sha1-ff3e171" +
		"5b44d1e05ee047abf18703e52cf61024d\n| | \"4\"\n| | \n| * sha1-ff3e1715b44d1e05ee047abf18703e52cf6102" +
		"4d\n| | Parent: sha1-46481de358be65e6e58fefb8db7e680a15f90297\n| | \"3.7\"\n| | \n| *   sha1-46481d" +
		"e358be65e6e58fefb8db7e680a15f90297\n| |\\  Merge: sha1-4da4474c6c032fe3d5a7f6d051ac61142cc0777d sha" +
		"1-d392854a7b3fea30b8f30cb1db70e27d03c23c6e\n| | | \"3.5\"\n| | | \n| | * sha1-d392854a7b3fea30b8f30" +
		"cb1db70e27d03c23c6e\n| | | Parent: sha1-dee9a653168b3b8ef274b2ec60c1b2524306591d\n| | | \"3.1.7\"\n" +
		"| | | \n| | * sha1-dee9a653168b3b8ef274b2ec60c1b2524306591d\n| | | Parent: sha1-242e659af0ef0e376b8" +
		"77b64ffac1ba42f70df69\n| | | \"3.1.5\"\n| | | \n* | | sha1-ee04cc1dda6b010ca3488e4996d14a54176b169d" +
		"\n| | | Parent: sha1-9b247a8497322fd362f98fd4f990bb175ca03908\n| | | \"3.6\"\n| | | \n| | * sha1-24" +
		"2e659af0ef0e376b877b64ffac1ba42f70df69\n| | | Parent: sha1-4da4474c6c032fe3d5a7f6d051ac61142cc0777d" +
		"\n| | | \"3.1.3\"\n| | | \n* | | sha1-9b247a8497322fd362f98fd4f990bb175ca03908\n| |/  Parent: sha1-" +
		"9443faf02f7495e53f3f1e87b180e328424f2830\n| |   \"3.2\"\n| |   \n| * sha1-4da4474c6c032fe3d5a7f6d05" +
		"1ac61142cc0777d\n|/  Parent: sha1-9443faf02f7495e53f3f1e87b180e328424f2830\n|   \"3.1\"\n|   \n* sh" +
		"a1-9443faf02f7495e53f3f1e87b180e328424f2830\n| Parent: sha1-c2961e584d41e98a7c735e399eef6c618e0431b" +
		"6\n| \"3\"\n| \n* sha1-c2961e584d41e98a7c735e399eef6c618e0431b6\n| Parent: sha1-4a1a4e051327f02c1be" +
		"502ac7ce9e7bf04fbf729\n| \"2\"\n| \n* sha1-4a1a4e051327f02c1be502ac7ce9e7bf04fbf729\n| Parent: None" +
		"\n| \"1\"\n"
	graphRes2 = "*   sha1-a7f6c6b7f0db1f9d2448bf23c4aa70d983dfecb2\n|\\  Merge: sha1-10473a7892604ff88d9" +
		"149e3cbb9dd9dc123d194 sha1-d37384e9e9cf2f9a0abd5968151c246fdd8cf9dd\n| | \"101\"\n| | \n| *   sha1-" +
		"d37384e9e9cf2f9a0abd5968151c246fdd8cf9dd\n| |\\  Merge: sha1-07cec20929f80a1fd923991683f4bf3adad099" +
		"03 sha1-4a1a4e051327f02c1be502ac7ce9e7bf04fbf729\n| | | \"11\"\n| | | \n* | sha1-10473a7892604ff88d" +
		"9149e3cbb9dd9dc123d194\n| | Parent: None\n| | \"100\"\n| | \n* sha1-07cec20929f80a1fd923991683f4bf3" +
		"adad09903\n| Parent: None\n| \"10\"\n| \n* sha1-4a1a4e051327f02c1be502ac7ce9e7bf04fbf729\n| Parent:" +
		" None\n| \"1\"\n"
)
