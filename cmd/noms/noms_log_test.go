// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"testing"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/noms/go/util/test"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

func TestNomsLog(t *testing.T) {
	suite.Run(t, &nomsLogTestSuite{})
}

type nomsLogTestSuite struct {
	clienttest.ClientTestSuite
}

func testCommitInResults(s *nomsLogTestSuite, str string, i int) {
	sp, err := spec.ForDataset(str)
	s.NoError(err)
	defer sp.Close()

	sp.GetDatabase().CommitValue(sp.GetDataset(), types.Number(i))
	s.NoError(err)

	commit := sp.GetDataset().Head()
	res, _ := s.MustRun(main, []string{"log", str})
	s.Contains(res, commit.Hash().String())
}

func (s *nomsLogTestSuite) TestNomsLog() {
	sp, err := spec.ForDataset(spec.CreateValueSpecString("nbs", s.DBDir, "dsTest"))
	s.NoError(err)
	defer sp.Close()

	sp.GetDatabase() // create the database
	s.Panics(func() { s.MustRun(main, []string{"log", sp.String()}) })

	testCommitInResults(s, sp.String(), 1)
	testCommitInResults(s, sp.String(), 2)
}

func (s *nomsLogTestSuite) TestNomsLogPath() {
	sp, err := spec.ForPath(spec.CreateValueSpecString("nbs", s.DBDir, "dsTest.value.bar"))
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()
	ds := sp.GetDataset()
	for i := 0; i < 3; i++ {
		data := types.NewStruct("", types.StructData{
			"bar": types.Number(i),
		})
		ds, err = db.CommitValue(ds, data)
		s.NoError(err)
	}

	stdout, stderr := s.MustRun(main, []string{"log", "--show-value", sp.String()})
	s.Empty(stderr)
	test.EqualsIgnoreHashes(s.T(), pathValue, stdout)

	stdout, stderr = s.MustRun(main, []string{"log", sp.String()})
	s.Empty(stderr)
	test.EqualsIgnoreHashes(s.T(), pathDiff, stdout)
}

func addCommit(ds datas.Dataset, v string) (datas.Dataset, error) {
	return ds.Database().CommitValue(ds, types.String(v))
}

func addCommitWithValue(ds datas.Dataset, v types.Value) (datas.Dataset, error) {
	return ds.Database().CommitValue(ds, v)
}

func addBranchedDataset(newDs, parentDs datas.Dataset, v string) (datas.Dataset, error) {
	p := types.NewSet(parentDs.HeadRef())
	return newDs.Database().Commit(newDs, types.String(v), datas.CommitOptions{Parents: p})
}

func mergeDatasets(ds1, ds2 datas.Dataset, v string) (datas.Dataset, error) {
	p := types.NewSet(ds1.HeadRef(), ds2.HeadRef())
	return ds1.Database().Commit(ds1, types.String(v), datas.CommitOptions{Parents: p})
}

func (s *nomsLogTestSuite) TestNArg() {
	dsName := "nArgTest"

	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	ds := sp.GetDatabase().GetDataset(dsName)

	ds, err = addCommit(ds, "1")
	h1 := ds.Head().Hash()
	s.NoError(err)
	ds, err = addCommit(ds, "2")
	s.NoError(err)
	h2 := ds.Head().Hash()
	ds, err = addCommit(ds, "3")
	s.NoError(err)
	h3 := ds.Head().Hash()

	dsSpec := spec.CreateValueSpecString("nbs", s.DBDir, dsName)
	res, _ := s.MustRun(main, []string{"log", "-n1", dsSpec})
	s.NotContains(res, h1.String())
	res, _ = s.MustRun(main, []string{"log", "-n0", dsSpec})
	s.Contains(res, h3.String())
	s.Contains(res, h2.String())
	s.Contains(res, h1.String())

	vSpec := spec.CreateValueSpecString("nbs", s.DBDir, "#"+h3.String())
	res, _ = s.MustRun(main, []string{"log", "-n1", vSpec})
	s.NotContains(res, h1.String())
	res, _ = s.MustRun(main, []string{"log", "-n0", vSpec})
	s.Contains(res, h3.String())
	s.Contains(res, h2.String())
	s.Contains(res, h1.String())
}

func (s *nomsLogTestSuite) TestEmptyCommit() {
	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()
	ds := db.GetDataset("ds1")

	meta := types.NewStruct("Meta", map[string]types.Value{
		"longNameForTest": types.String("Yoo"),
		"test2":           types.String("Hoo"),
	})
	ds, err = db.Commit(ds, types.String("1"), datas.CommitOptions{Meta: meta})
	s.NoError(err)

	ds, err = db.Commit(ds, types.String("2"), datas.CommitOptions{})
	s.NoError(err)

	dsSpec := spec.CreateValueSpecString("nbs", s.DBDir, "ds1")
	res, _ := s.MustRun(main, []string{"log", "--show-value=false", dsSpec})
	test.EqualsIgnoreHashes(s.T(), metaRes1, res)

	res, _ = s.MustRun(main, []string{"log", "--show-value=false", "--oneline", dsSpec})
	test.EqualsIgnoreHashes(s.T(), metaRes2, res)
}

func (s *nomsLogTestSuite) TestNomsGraph1() {
	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()

	b1 := db.GetDataset("b1")
	b1, err = addCommit(b1, "1")
	s.NoError(err)
	b1, err = addCommit(b1, "2")
	s.NoError(err)
	b1, err = addCommit(b1, "3")
	s.NoError(err)

	b2 := db.GetDataset("b2")
	b2, err = addBranchedDataset(b2, b1, "3.1")
	s.NoError(err)

	b1, err = addCommit(b1, "3.2")
	s.NoError(err)
	b1, err = addCommit(b1, "3.6")
	s.NoError(err)

	b3 := db.GetDataset("b3")
	b3, err = addBranchedDataset(b3, b2, "3.1.3")
	s.NoError(err)
	b3, err = addCommit(b3, "3.1.5")
	s.NoError(err)
	b3, err = addCommit(b3, "3.1.7")
	s.NoError(err)

	b2, err = mergeDatasets(b2, b3, "3.5")
	s.NoError(err)
	b2, err = addCommit(b2, "3.7")
	s.NoError(err)

	b1, err = mergeDatasets(b1, b2, "4")
	s.NoError(err)

	b1, err = addCommit(b1, "5")
	s.NoError(err)
	b1, err = addCommit(b1, "6")
	s.NoError(err)
	b1, err = addCommit(b1, "7")
	s.NoError(err)

	res, _ := s.MustRun(main, []string{"log", "--graph", "--show-value=true", spec.CreateValueSpecString("nbs", s.DBDir, "b1")})
	s.Equal(graphRes1, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", spec.CreateValueSpecString("nbs", s.DBDir, "b1")})
	s.Equal(diffRes1, res)
}

func (s *nomsLogTestSuite) TestNomsGraph2() {
	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()

	ba := db.GetDataset("ba")
	ba, err = addCommit(ba, "1")
	s.NoError(err)

	bb := db.GetDataset("bb")
	bb, err = addCommit(bb, "10")
	s.NoError(err)

	bc := db.GetDataset("bc")
	bc, err = addCommit(bc, "100")
	s.NoError(err)

	ba, err = mergeDatasets(ba, bb, "11")
	s.NoError(err)

	_, err = mergeDatasets(ba, bc, "101")
	s.NoError(err)

	res, _ := s.MustRun(main, []string{"log", "--graph", "--show-value=true", spec.CreateValueSpecString("nbs", s.DBDir, "ba")})
	s.Equal(graphRes2, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", spec.CreateValueSpecString("nbs", s.DBDir, "ba")})
	s.Equal(diffRes2, res)
}

func (s *nomsLogTestSuite) TestNomsGraph3() {
	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	db := sp.GetDatabase()

	w := db.GetDataset("w")

	w, err = addCommit(w, "1")
	s.NoError(err)

	w, err = addCommit(w, "2")
	s.NoError(err)

	x := db.GetDataset("x")
	x, err = addBranchedDataset(x, w, "20-x")
	s.NoError(err)

	y := db.GetDataset("y")
	y, err = addBranchedDataset(y, w, "200-y")
	s.NoError(err)

	z := db.GetDataset("z")
	z, err = addBranchedDataset(z, w, "2000-z")
	s.NoError(err)

	w, err = mergeDatasets(w, x, "22-wx")
	s.NoError(err)

	w, err = mergeDatasets(w, y, "222-wy")
	s.NoError(err)

	_, err = mergeDatasets(w, z, "2222-wz")
	s.NoError(err)

	res, _ := s.MustRun(main, []string{"log", "--graph", "--show-value=true", spec.CreateValueSpecString("nbs", s.DBDir, "w")})
	test.EqualsIgnoreHashes(s.T(), graphRes3, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", spec.CreateValueSpecString("nbs", s.DBDir, "w")})
	test.EqualsIgnoreHashes(s.T(), diffRes3, res)
}

func (s *nomsLogTestSuite) TestTruncation() {
	toNomsList := func(l []string) types.List {
		nv := []types.Value{}
		for _, v := range l {
			nv = append(nv, types.String(v))
		}
		return types.NewList(nv...)
	}

	sp, err := spec.ForDatabase(spec.CreateDatabaseSpecString("nbs", s.DBDir))
	s.NoError(err)
	defer sp.Close()

	t := sp.GetDatabase().GetDataset("truncate")

	t, err = addCommit(t, "the first line")
	s.NoError(err)

	l := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven"}
	_, err = addCommitWithValue(t, toNomsList(l))
	s.NoError(err)

	dsSpec := spec.CreateValueSpecString("nbs", s.DBDir, "truncate")
	res, _ := s.MustRun(main, []string{"log", "--graph", "--show-value=true", dsSpec})
	test.EqualsIgnoreHashes(s.T(), truncRes1, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", dsSpec})
	test.EqualsIgnoreHashes(s.T(), diffTrunc1, res)

	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=true", "--max-lines=-1", dsSpec})
	test.EqualsIgnoreHashes(s.T(), truncRes2, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", "--max-lines=-1", dsSpec})
	test.EqualsIgnoreHashes(s.T(), diffTrunc2, res)

	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=true", "--max-lines=0", dsSpec})
	test.EqualsIgnoreHashes(s.T(), truncRes3, res)
	res, _ = s.MustRun(main, []string{"log", "--graph", "--show-value=false", "--max-lines=0", dsSpec})
	test.EqualsIgnoreHashes(s.T(), diffTrunc3, res)
}

func TestBranchlistSplice(t *testing.T) {
	assert := assert.New(t)
	bl := branchList{}
	for i := 0; i < 4; i++ {
		bl = bl.Splice(0, 0, branch{})
	}
	assert.Equal(4, len(bl))
	bl = bl.Splice(3, 1)
	bl = bl.Splice(0, 1)
	bl = bl.Splice(1, 1)
	bl = bl.Splice(0, 1)
	assert.Zero(len(bl))

	for i := 0; i < 4; i++ {
		bl = bl.Splice(0, 0, branch{})
	}
	assert.Equal(4, len(bl))

	branchesToDelete := []int{1, 2, 3}
	bl = bl.RemoveBranches(branchesToDelete)
	assert.Equal(1, len(bl))
}

const (
	graphRes1 = "* 7h4be5ngf1kvp35r94agj5a6vh9afu19\n| Parent: anjt1fu2mod83gojg9tb690rtauf03rn\n| \"7\"\n| \n* anjt1fu2mod83gojg9tb690rtauf03rn\n| Parent: id209cmfd3pouu19m0absg3066239v49\n| \"6\"\n| \n* id209cmfd3pouu19m0absg3066239v49\n| Parent: or0snhnimj9gqetrr2m5csnv5d3vs3n3\n| \"5\"\n| \n*   or0snhnimj9gqetrr2m5csnv5d3vs3n3\n|\\  Merge: ldimfvtvabe2ss78mca5cnu7rrr0vbfq cbej2j0jnudfvhg092h36b8f6f703oa8\n| | \"4\"\n| | \n* | ldimfvtvabe2ss78mca5cnu7rrr0vbfq\n| | Parent: ipce5nghuig5mam8e08gqtm85r05o02f\n| | \"3.7\"\n| | \n* |   ipce5nghuig5mam8e08gqtm85r05o02f\n|\\ \\  Merge: g8kp9qp5ur98uoko5b3de5t4lbe1o3e8 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | | \"3.5\"\n| | | \n* | | g8kp9qp5ur98uoko5b3de5t4lbe1o3e8\n| | | Parent: 52eo9rc1715smklpb9h12fbv77i10lma\n| | | \"3.1.7\"\n| | | \n* | | 52eo9rc1715smklpb9h12fbv77i10lma\n| | | Parent: 0d3eb218bu1coaacpahjguuh7pas1sl3\n| | | \"3.1.5\"\n| | | \n* | | 0d3eb218bu1coaacpahjguuh7pas1sl3\n| | | Parent: 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | | \"3.1.3\"\n| | | \n| | * cbej2j0jnudfvhg092h36b8f6f703oa8\n|/  | Parent: fdd94rg56c6sln658omo3q6560d038ft\n|   | \"3.6\"\n|   | \n* | 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | Parent: 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n| | \"3.1\"\n| | \n| * fdd94rg56c6sln658omo3q6560d038ft\n|/  Parent: 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n|   \"3.2\"\n|   \n* 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n| Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| \"3\"\n| \n* ipht90juefbs13gpvjnjtdpkn83rjb77\n| Parent: nf8shs15uvbl18et3ai05dogd0ddfvnb\n| \"2\"\n| \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \"1\"\n"
	diffRes1  = "* 7h4be5ngf1kvp35r94agj5a6vh9afu19\n| Parent: anjt1fu2mod83gojg9tb690rtauf03rn\n| -   \"6\"\n| +   \"7\"\n| \n* anjt1fu2mod83gojg9tb690rtauf03rn\n| Parent: id209cmfd3pouu19m0absg3066239v49\n| -   \"5\"\n| +   \"6\"\n| \n* id209cmfd3pouu19m0absg3066239v49\n| Parent: or0snhnimj9gqetrr2m5csnv5d3vs3n3\n| -   \"4\"\n| +   \"5\"\n| \n*   or0snhnimj9gqetrr2m5csnv5d3vs3n3\n|\\  Merge: ldimfvtvabe2ss78mca5cnu7rrr0vbfq cbej2j0jnudfvhg092h36b8f6f703oa8\n| | -   \"3.7\"\n| | +   \"4\"\n| | \n* | ldimfvtvabe2ss78mca5cnu7rrr0vbfq\n| | Parent: ipce5nghuig5mam8e08gqtm85r05o02f\n| | -   \"3.5\"\n| | +   \"3.7\"\n| | \n* |   ipce5nghuig5mam8e08gqtm85r05o02f\n|\\ \\  Merge: g8kp9qp5ur98uoko5b3de5t4lbe1o3e8 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | | -   \"3.1.7\"\n| | | +   \"3.5\"\n| | | \n* | | g8kp9qp5ur98uoko5b3de5t4lbe1o3e8\n| | | Parent: 52eo9rc1715smklpb9h12fbv77i10lma\n| | | -   \"3.1.5\"\n| | | +   \"3.1.7\"\n| | | \n* | | 52eo9rc1715smklpb9h12fbv77i10lma\n| | | Parent: 0d3eb218bu1coaacpahjguuh7pas1sl3\n| | | -   \"3.1.3\"\n| | | +   \"3.1.5\"\n| | | \n* | | 0d3eb218bu1coaacpahjguuh7pas1sl3\n| | | Parent: 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | | -   \"3.1\"\n| | | +   \"3.1.3\"\n| | | \n| | * cbej2j0jnudfvhg092h36b8f6f703oa8\n|/  | Parent: fdd94rg56c6sln658omo3q6560d038ft\n|   | -   \"3.2\"\n|   | +   \"3.6\"\n|   | \n* | 6lhinb213vhpr4r37f3t0hsjooetsmdl\n| | Parent: 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n| | -   \"3\"\n| | +   \"3.1\"\n| | \n| * fdd94rg56c6sln658omo3q6560d038ft\n|/  Parent: 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n|   -   \"3\"\n|   +   \"3.2\"\n|   \n* 6st4l3o1pebqnvtq5vbfa0a2jhb8m8lg\n| Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| -   \"2\"\n| +   \"3\"\n| \n* ipht90juefbs13gpvjnjtdpkn83rjb77\n| Parent: nf8shs15uvbl18et3ai05dogd0ddfvnb\n| -   \"1\"\n| +   \"2\"\n| \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \n"

	graphRes2 = "*   7sgafqdm5mvorjlpnnoo22m8mtdbf1ih\n|\\  Merge: u3mmk88at2ur2tpu2tv8spim650cu19c 6ctvkat0af1nrmsdienl920ptu6l1lot\n| | \"101\"\n| | \n* |   u3mmk88at2ur2tpu2tv8spim650cu19c\n|\\ \\  Merge: c7rkjsu2lluhni44k955nhupko7g7ibs nf8shs15uvbl18et3ai05dogd0ddfvnb\n| | | \"11\"\n| | | \n* | c7rkjsu2lluhni44k955nhupko7g7ibs\n| | Parent: None\n| | \"10\"\n| | \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \"1\"\n| \n* 6ctvkat0af1nrmsdienl920ptu6l1lot\n| Parent: None\n| \"100\"\n"
	diffRes2  = "*   7sgafqdm5mvorjlpnnoo22m8mtdbf1ih\n|\\  Merge: u3mmk88at2ur2tpu2tv8spim650cu19c 6ctvkat0af1nrmsdienl920ptu6l1lot\n| | -   \"11\"\n| | +   \"101\"\n| | \n* |   u3mmk88at2ur2tpu2tv8spim650cu19c\n|\\ \\  Merge: c7rkjsu2lluhni44k955nhupko7g7ibs nf8shs15uvbl18et3ai05dogd0ddfvnb\n| | | -   \"10\"\n| | | +   \"11\"\n| | | \n* | c7rkjsu2lluhni44k955nhupko7g7ibs\n| | Parent: None\n| | \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \n* 6ctvkat0af1nrmsdienl920ptu6l1lot\n| Parent: None\n| \n"

	graphRes3 = "*   k60mr8o17qkuggmb2q7qhoeu282k79h8\n|\\  Merge: 5sofa8nask2mb0pjomfoil8pa8ro8c9p 18kut3ou8bho6dsontb0caqn2hes58k9\n| | \"2222-wz\"\n| | \n* |   5sofa8nask2mb0pjomfoil8pa8ro8c9p\n|\\ \\  Merge: sgdu5b2b83tpm2tabh43ai6e27rhaknu brviub3og7753cb2n1ojvrlc23kpn0ce\n| | | \"222-wy\"\n| | | \n| * |   brviub3og7753cb2n1ojvrlc23kpn0ce\n| |\\ \\  Merge: f6gi9e2365884otnbftqh74u3is0uep5 ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | \"22-wx\"\n| | | | \n* | | | sgdu5b2b83tpm2tabh43ai6e27rhaknu\n| | | | Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | \"200-y\"\n| | | | \n| * | | f6gi9e2365884otnbftqh74u3is0uep5\n| | | | Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | \"20-x\"\n| | | | \n| | | * 18kut3ou8bho6dsontb0caqn2hes58k9\n|/ / /  Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n|       \"2000-z\"\n|       \n* ipht90juefbs13gpvjnjtdpkn83rjb77\n| Parent: nf8shs15uvbl18et3ai05dogd0ddfvnb\n| \"2\"\n| \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \"1\"\n"
	diffRes3  = "*   k60mr8o17qkuggmb2q7qhoeu282k79h8\n|\\  Merge: 5sofa8nask2mb0pjomfoil8pa8ro8c9p 18kut3ou8bho6dsontb0caqn2hes58k9\n| | -   \"222-wy\"\n| | +   \"2222-wz\"\n| | \n* |   5sofa8nask2mb0pjomfoil8pa8ro8c9p\n|\\ \\  Merge: sgdu5b2b83tpm2tabh43ai6e27rhaknu brviub3og7753cb2n1ojvrlc23kpn0ce\n| | | -   \"200-y\"\n| | | +   \"222-wy\"\n| | | \n| * |   brviub3og7753cb2n1ojvrlc23kpn0ce\n| |\\ \\  Merge: f6gi9e2365884otnbftqh74u3is0uep5 ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | -   \"20-x\"\n| | | | +   \"22-wx\"\n| | | | \n* | | | sgdu5b2b83tpm2tabh43ai6e27rhaknu\n| | | | Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | -   \"2\"\n| | | | +   \"200-y\"\n| | | | \n| * | | f6gi9e2365884otnbftqh74u3is0uep5\n| | | | Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n| | | | -   \"2\"\n| | | | +   \"20-x\"\n| | | | \n| | | * 18kut3ou8bho6dsontb0caqn2hes58k9\n|/ / /  Parent: ipht90juefbs13gpvjnjtdpkn83rjb77\n|       -   \"2\"\n|       +   \"2000-z\"\n|       \n* ipht90juefbs13gpvjnjtdpkn83rjb77\n| Parent: nf8shs15uvbl18et3ai05dogd0ddfvnb\n| -   \"1\"\n| +   \"2\"\n| \n* nf8shs15uvbl18et3ai05dogd0ddfvnb\n| Parent: None\n| \n"

	truncRes1  = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n| [  // 11 items\n|   \"one\",\n|   \"two\",\n|   \"three\",\n|   \"four\",\n|   \"five\",\n|   \"six\",\n|   \"seven\",\n| ...\n| \n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n| \"the first line\"\n"
	diffTrunc1 = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n| -   \"the first line\"\n| +   [  // 11 items\n| +     \"one\",\n| +     \"two\",\n| +     \"three\",\n| +     \"four\",\n| +     \"five\",\n| +     \"six\",\n| ...\n| \n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n| \n"

	truncRes2  = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n| [  // 11 items\n|   \"one\",\n|   \"two\",\n|   \"three\",\n|   \"four\",\n|   \"five\",\n|   \"six\",\n|   \"seven\",\n|   \"eight\",\n|   \"nine\",\n|   \"ten\",\n|   \"eleven\",\n| ]\n| \n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n| \"the first line\"\n"
	diffTrunc2 = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n| -   \"the first line\"\n| +   [  // 11 items\n| +     \"one\",\n| +     \"two\",\n| +     \"three\",\n| +     \"four\",\n| +     \"five\",\n| +     \"six\",\n| +     \"seven\",\n| +     \"eight\",\n| +     \"nine\",\n| +     \"ten\",\n| +     \"eleven\",\n| +   ]\n| \n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n| \n"

	truncRes3  = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n"
	diffTrunc3 = "* p1442asfqnhgv1ebg6rijhl3kb9n4vt3\n| Parent: 4tq9si4tk8n0pead7hovehcbuued45sa\n* 4tq9si4tk8n0pead7hovehcbuued45sa\n| Parent: None\n"

	metaRes1 = "p7jmuh67vhfccnqk1bilnlovnms1m67o\nParent: f8gjiv5974ojir9tnrl2k393o4s1tf0r\n-   \"1\"\n+   \"2\"\n\nf8gjiv5974ojir9tnrl2k393o4s1tf0r\nParent:          None\nLongNameForTest: \"Yoo\"\nTest2:           \"Hoo\"\n\n"
	metaRes2 = "p7jmuh67vhfccnqk1bilnlovnms1m67o (Parent: f8gjiv5974ojir9tnrl2k393o4s1tf0r)\nf8gjiv5974ojir9tnrl2k393o4s1tf0r (Parent: None)\n"

	pathValue = "oki4cv7vkh743rccese3r3omf6l6mao4\nParent: lca4vejkm0iqsk7ok5322pt61u4otn6q\n2\n\nlca4vejkm0iqsk7ok5322pt61u4otn6q\nParent: u42pi8ukgkvpoi6n7d46cklske41oguf\n1\n\nu42pi8ukgkvpoi6n7d46cklske41oguf\nParent: hgmlqmsnrb3sp9jqc6mas8kusa1trrs2\n0\n\nhgmlqmsnrb3sp9jqc6mas8kusa1trrs2\nParent: hffiuecdpoq622tamm3nvungeca99ohl\n<nil>\nhffiuecdpoq622tamm3nvungeca99ohl\nParent: None\n<nil>\n"

	pathDiff = "oki4cv7vkh743rccese3r3omf6l6mao4\nParent: lca4vejkm0iqsk7ok5322pt61u4otn6q\n-   1\n+   2\n\nlca4vejkm0iqsk7ok5322pt61u4otn6q\nParent: u42pi8ukgkvpoi6n7d46cklske41oguf\n-   0\n+   1\n\nu42pi8ukgkvpoi6n7d46cklske41oguf\nParent: hgmlqmsnrb3sp9jqc6mas8kusa1trrs2\nold (#hgmlqmsnrb3sp9jqc6mas8kusa1trrs2.value.bar) not found\n\nhgmlqmsnrb3sp9jqc6mas8kusa1trrs2\nParent: hffiuecdpoq622tamm3nvungeca99ohl\nnew (#hgmlqmsnrb3sp9jqc6mas8kusa1trrs2.value.bar) not found\nold (#hffiuecdpoq622tamm3nvungeca99ohl.value.bar) not found\n\nhffiuecdpoq622tamm3nvungeca99ohl\nParent: None\n\n"
)
