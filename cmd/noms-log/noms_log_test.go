// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/dataset"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/test_util"
	"github.com/attic-labs/noms/samples/go/util"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

type testExiter struct{}
type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("Exiting with code: %d", e.code)
}

func (testExiter) Exit(code int) {
	panic(exitError{code})
}

func TestNomsShow(t *testing.T) {
	util.UtilExiter = testExiter{}
	suite.Run(t, &nomsShowTestSuite{})
}

type nomsShowTestSuite struct {
	test_util.ClientTestSuite
}

func testCommitInResults(s *nomsShowTestSuite, str string, i int) {
	sp, err := spec.ParseDatasetSpec(str)
	s.NoError(err)
	ds, err := sp.Dataset()
	s.NoError(err)
	ds, err = ds.Commit(types.Number(i))
	s.NoError(err)
	commit := ds.Head()
	ds.Database().Close()
	s.Contains(s.Run(main, []string{str}), commit.Hash().String())
}

func (s *nomsShowTestSuite) TestNomsLog() {
	datasetName := "dsTest"
	str := test_util.CreateValueSpecString("ldb", s.LdbDir, datasetName)
	sp, err := spec.ParseDatasetSpec(str)
	s.NoError(err)

	ds, err := sp.Dataset()
	s.NoError(err)
	ds.Database().Close()
	s.Panics(func() { s.Run(main, []string{str}) })

	testCommitInResults(s, str, 1)
	testCommitInResults(s, str, 2)
}

func addCommit(ds dataset.Dataset, v string) (dataset.Dataset, error) {
	return ds.Commit(types.String(v))
}

func addCommitWithValue(ds dataset.Dataset, v types.Value) (dataset.Dataset, error) {
	return ds.Commit(v)
}

func addBranchedDataset(newDs, parentDs dataset.Dataset, v string) (dataset.Dataset, error) {
	return newDs.CommitWithParents(types.String(v), types.NewSet().Insert(parentDs.HeadRef()))
}

func mergeDatasets(ds1, ds2 dataset.Dataset, v string) (dataset.Dataset, error) {
	return ds1.CommitWithParents(types.String(v), types.NewSet(ds1.HeadRef(), ds2.HeadRef()))
}

func (s *nomsShowTestSuite) TestNArg() {
	str := test_util.CreateDatabaseSpecString("ldb", s.LdbDir)
	dsName := "nArgTest"
	dbSpec, err := spec.ParseDatabaseSpec(str)
	s.NoError(err)
	db, err := dbSpec.Database()
	s.NoError(err)

	ds := dataset.NewDataset(db, dsName)

	ds, err = addCommit(ds, "1")
	h1 := ds.Head().Hash()
	s.NoError(err)
	ds, err = addCommit(ds, "2")
	s.NoError(err)
	h2 := ds.Head().Hash()
	ds, err = addCommit(ds, "3")
	s.NoError(err)
	h3 := ds.Head().Hash()
	db.Close()

	dsSpec := test_util.CreateValueSpecString("ldb", s.LdbDir, dsName)
	s.NotContains(s.Run(main, []string{"-n=1", dsSpec}), h1.String())
	res := s.Run(main, []string{"-n=0", dsSpec})
	s.Contains(res, h3.String())
	s.Contains(res, h2.String())
	s.Contains(res, h1.String())

	vSpec := test_util.CreateValueSpecString("ldb", s.LdbDir, h3.String())
	s.NotContains(s.Run(main, []string{"-n=1", vSpec}), h1.String())
	res = s.Run(main, []string{"-n=0", vSpec})
	s.Contains(res, h3.String())
	s.Contains(res, h2.String())
	s.Contains(res, h1.String())
}

func (s *nomsShowTestSuite) TestNomsGraph1() {
	str := test_util.CreateDatabaseSpecString("ldb", s.LdbDir)
	dbSpec, err := spec.ParseDatabaseSpec(str)
	s.NoError(err)
	db, err := dbSpec.Database()
	s.NoError(err)

	b1 := dataset.NewDataset(db, "b1")

	b1, err = addCommit(b1, "1")
	s.NoError(err)
	b1, err = addCommit(b1, "2")
	s.NoError(err)
	b1, err = addCommit(b1, "3")
	s.NoError(err)

	b2 := dataset.NewDataset(db, "b2")
	b2, err = addBranchedDataset(b2, b1, "3.1")
	s.NoError(err)

	b1, err = addCommit(b1, "3.2")
	s.NoError(err)
	b1, err = addCommit(b1, "3.6")
	s.NoError(err)

	b3 := dataset.NewDataset(db, "b3")
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

	b1.Database().Close()
	s.Equal(graphRes1, s.Run(main, []string{"-graph", "-show-value=true", test_util.CreateValueSpecString("ldb", s.LdbDir, "b1")}))
	s.Equal(diffRes1, s.Run(main, []string{"-graph", "-show-value=false", test_util.CreateValueSpecString("ldb", s.LdbDir, "b1")}))
}

func (s *nomsShowTestSuite) TestNomsGraph2() {
	str := test_util.CreateDatabaseSpecString("ldb", s.LdbDir)
	dbSpec, err := spec.ParseDatabaseSpec(str)
	s.NoError(err)
	db, err := dbSpec.Database()
	s.NoError(err)

	ba := dataset.NewDataset(db, "ba")

	ba, err = addCommit(ba, "1")
	s.NoError(err)

	bb := dataset.NewDataset(db, "bb")
	bb, err = addCommit(bb, "10")
	s.NoError(err)

	bc := dataset.NewDataset(db, "bc")
	bc, err = addCommit(bc, "100")
	s.NoError(err)

	ba, err = mergeDatasets(ba, bb, "11")
	s.NoError(err)

	_, err = mergeDatasets(ba, bc, "101")
	s.NoError(err)

	db.Close()
	s.Equal(graphRes2, s.Run(main, []string{"-graph", "-show-value=true", test_util.CreateValueSpecString("ldb", s.LdbDir, "ba")}))
	s.Equal(diffRes2, s.Run(main, []string{"-graph", "-show-value=false", test_util.CreateValueSpecString("ldb", s.LdbDir, "ba")}))
}

func (s *nomsShowTestSuite) TestNomsGraph3() {
	str := test_util.CreateDatabaseSpecString("ldb", s.LdbDir)
	dbSpec, err := spec.ParseDatabaseSpec(str)
	s.NoError(err)
	db, err := dbSpec.Database()
	s.NoError(err)

	w := dataset.NewDataset(db, "w")

	w, err = addCommit(w, "1")
	s.NoError(err)

	w, err = addCommit(w, "2")
	s.NoError(err)

	x := dataset.NewDataset(db, "x")
	x, err = addBranchedDataset(x, w, "20-x")
	s.NoError(err)

	y := dataset.NewDataset(db, "y")
	y, err = addBranchedDataset(y, w, "200-y")
	s.NoError(err)

	z := dataset.NewDataset(db, "z")
	z, err = addBranchedDataset(z, w, "2000-z")
	s.NoError(err)

	w, err = mergeDatasets(w, x, "22-wx")
	s.NoError(err)

	w, err = mergeDatasets(w, y, "222-wy")
	s.NoError(err)

	_, err = mergeDatasets(w, z, "2222-wz")
	s.NoError(err)

	db.Close()
	s.Equal(graphRes3, s.Run(main, []string{"-graph", "-show-value=true", test_util.CreateValueSpecString("ldb", s.LdbDir, "w")}))
	s.Equal(diffRes3, s.Run(main, []string{"-graph", "-show-value=false", test_util.CreateValueSpecString("ldb", s.LdbDir, "w")}))
}

func (s *nomsShowTestSuite) TestTruncation() {
	toNomsList := func(l []string) types.List {
		nv := []types.Value{}
		for _, v := range l {
			nv = append(nv, types.String(v))
		}
		return types.NewList(nv...)
	}

	str := test_util.CreateDatabaseSpecString("ldb", s.LdbDir)
	dbSpec, err := spec.ParseDatabaseSpec(str)
	s.NoError(err)
	db, err := dbSpec.Database()
	s.NoError(err)

	t := dataset.NewDataset(db, "truncate")

	t, err = addCommit(t, "the first line")
	s.NoError(err)

	l := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven"}
	_, err = addCommitWithValue(t, toNomsList(l))
	s.NoError(err)
	db.Close()

	dsSpec := test_util.CreateValueSpecString("ldb", s.LdbDir, "truncate")
	s.Equal(truncRes1, s.Run(main, []string{"-graph", "-show-value=true", dsSpec}))
	s.Equal(diffTrunc1, s.Run(main, []string{"-graph", "-show-value=false", dsSpec}))

	s.Equal(truncRes2, s.Run(main, []string{"-graph", "-show-value=true", "-max-lines=-1", dsSpec}))
	s.Equal(diffTrunc2, s.Run(main, []string{"-graph", "-show-value=false", "-max-lines=-1", dsSpec}))

	s.Equal(truncRes3, s.Run(main, []string{"-graph", "-show-value=true", "-max-lines=0", dsSpec}))
	s.Equal(diffTrunc3, s.Run(main, []string{"-graph", "-show-value=false", "-max-lines=0", dsSpec}))
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
	graphRes1 = "* 1rddfxfuufwh2niw8gkvvoqsblx17wlplnienz7n7fxnke6oov\n| Parent: 2f1o6lmnw6bzkusq1oz8idqldh36sbmo7s3zp2o2hx9idxi4e4\n| \"7\"\n| \n* 2f1o6lmnw6bzkusq1oz8idqldh36sbmo7s3zp2o2hx9idxi4e4\n| Parent: 37jcb29nnm76hnmarjrvlk80j81fcl1kqxw9eieqjh533or2qc\n| \"6\"\n| \n* 37jcb29nnm76hnmarjrvlk80j81fcl1kqxw9eieqjh533or2qc\n| Parent: 08nngvk1e5nuqxm8hi3jrnjj4t72ygx1w9xultqzc0cvgl48au\n| \"5\"\n| \n*   08nngvk1e5nuqxm8hi3jrnjj4t72ygx1w9xultqzc0cvgl48au\n|\\  Merge: 09k1p4ldm02cwfrgoht35tq9lqkoo1nxw51y7tgpvli0rcjmuj 0vd1d9b70li8gmt2hlqluz616wce9p1hzbqyc1vqghhm89mouk\n| | \"4\"\n| | \n| * 0vd1d9b70li8gmt2hlqluz616wce9p1hzbqyc1vqghhm89mouk\n| | Parent: 496rmybqdnaqvhmqrkqxeoslt2z1zhodsln2blto1mfhaqev4c\n| | \"3.7\"\n| | \n| *   496rmybqdnaqvhmqrkqxeoslt2z1zhodsln2blto1mfhaqev4c\n| |\\  Merge: 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id 08z5h7xc4jpsp5ues9iu01ij3fntpbbaug8b8lbti0zbj60e1f\n| | | \"3.5\"\n| | | \n| | * 08z5h7xc4jpsp5ues9iu01ij3fntpbbaug8b8lbti0zbj60e1f\n| | | Parent: 4r7gqi3hd9mkf65qp3miph6lw5kfg9gu258ruqw2enwpg6hy0s\n| | | \"3.1.7\"\n| | | \n| | * 4r7gqi3hd9mkf65qp3miph6lw5kfg9gu258ruqw2enwpg6hy0s\n| | | Parent: 51vq10qotrvq39aec9ja1ex7t4bojvcc4791fzyonfi789o91z\n| | | \"3.1.5\"\n| | | \n* | | 09k1p4ldm02cwfrgoht35tq9lqkoo1nxw51y7tgpvli0rcjmuj\n| | | Parent: 6306gzu686dcvoyj05nvt7msxs282seugs5iq5iw35a84kxdd8\n| | | \"3.6\"\n| | | \n| | * 51vq10qotrvq39aec9ja1ex7t4bojvcc4791fzyonfi789o91z\n| | | Parent: 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id\n| | | \"3.1.3\"\n| | | \n* | | 6306gzu686dcvoyj05nvt7msxs282seugs5iq5iw35a84kxdd8\n| |/  Parent: 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n| |   \"3.2\"\n| |   \n| * 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id\n|/  Parent: 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n|   \"3.1\"\n|   \n* 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n| Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| \"3\"\n| \n* 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| Parent: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| \"2\"\n| \n* 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| Parent: None\n| \"1\"\n"
	diffRes1  = "* 1rddfxfuufwh2niw8gkvvoqsblx17wlplnienz7n7fxnke6oov\n| Parent: 2f1o6lmnw6bzkusq1oz8idqldh36sbmo7s3zp2o2hx9idxi4e4\n| -   \"6\"\n| +   \"7\"\n| \n* 2f1o6lmnw6bzkusq1oz8idqldh36sbmo7s3zp2o2hx9idxi4e4\n| Parent: 37jcb29nnm76hnmarjrvlk80j81fcl1kqxw9eieqjh533or2qc\n| -   \"5\"\n| +   \"6\"\n| \n* 37jcb29nnm76hnmarjrvlk80j81fcl1kqxw9eieqjh533or2qc\n| Parent: 08nngvk1e5nuqxm8hi3jrnjj4t72ygx1w9xultqzc0cvgl48au\n| -   \"4\"\n| +   \"5\"\n| \n*   08nngvk1e5nuqxm8hi3jrnjj4t72ygx1w9xultqzc0cvgl48au\n|\\  Merge: 09k1p4ldm02cwfrgoht35tq9lqkoo1nxw51y7tgpvli0rcjmuj 0vd1d9b70li8gmt2hlqluz616wce9p1hzbqyc1vqghhm89mouk\n| | -   \"3.6\"\n| | +   \"4\"\n| | \n| * 0vd1d9b70li8gmt2hlqluz616wce9p1hzbqyc1vqghhm89mouk\n| | Parent: 496rmybqdnaqvhmqrkqxeoslt2z1zhodsln2blto1mfhaqev4c\n| | -   \"3.5\"\n| | +   \"3.7\"\n| | \n| *   496rmybqdnaqvhmqrkqxeoslt2z1zhodsln2blto1mfhaqev4c\n| |\\  Merge: 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id 08z5h7xc4jpsp5ues9iu01ij3fntpbbaug8b8lbti0zbj60e1f\n| | | -   \"3.1\"\n| | | +   \"3.5\"\n| | | \n| | * 08z5h7xc4jpsp5ues9iu01ij3fntpbbaug8b8lbti0zbj60e1f\n| | | Parent: 4r7gqi3hd9mkf65qp3miph6lw5kfg9gu258ruqw2enwpg6hy0s\n| | | -   \"3.1.5\"\n| | | +   \"3.1.7\"\n| | | \n| | * 4r7gqi3hd9mkf65qp3miph6lw5kfg9gu258ruqw2enwpg6hy0s\n| | | Parent: 51vq10qotrvq39aec9ja1ex7t4bojvcc4791fzyonfi789o91z\n| | | -   \"3.1.3\"\n| | | +   \"3.1.5\"\n| | | \n* | | 09k1p4ldm02cwfrgoht35tq9lqkoo1nxw51y7tgpvli0rcjmuj\n| | | Parent: 6306gzu686dcvoyj05nvt7msxs282seugs5iq5iw35a84kxdd8\n| | | -   \"3.2\"\n| | | +   \"3.6\"\n| | | \n| | * 51vq10qotrvq39aec9ja1ex7t4bojvcc4791fzyonfi789o91z\n| | | Parent: 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id\n| | | -   \"3.1\"\n| | | +   \"3.1.3\"\n| | | \n* | | 6306gzu686dcvoyj05nvt7msxs282seugs5iq5iw35a84kxdd8\n| |/  Parent: 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n| |   -   \"3\"\n| |   +   \"3.2\"\n| |   \n| * 4ycx3q8rzupfb7bcr6tj21d2ljyye8cwb8od6kgbta3o6fz7id\n|/  Parent: 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n|   -   \"3\"\n|   +   \"3.1\"\n|   \n* 0svi1yv996tz9j2zbpb16jrl408l46qvzj8l81a1gumkwti5de\n| Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| -   \"2\"\n| +   \"3\"\n| \n* 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| Parent: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| -   \"1\"\n| +   \"2\"\n| \n* 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| Parent: None\n| \n"

	graphRes2 = "*   2fsrpbbtqonc6vddm1hh89osh49h1zam4lpmi6ozhmbv7cadaw\n|\\  Merge: 0b0gcibpbzdmnj5qb74wqmoow2sy0gnpviuhq8g4rhrsiggay3 2mwypyaqwaxuudmcmn0coehz7nslnnvr21kkixe04k1vb0m993\n| | \"101\"\n| | \n* |   0b0gcibpbzdmnj5qb74wqmoow2sy0gnpviuhq8g4rhrsiggay3\n|\\ \\  Merge: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc 57yam5w501mzvzohxpjm7x83lzkwpg4702g35trfo9nvz4aab8\n| | | \"11\"\n| | | \n* | 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| | Parent: None\n| | \"1\"\n| | \n* 57yam5w501mzvzohxpjm7x83lzkwpg4702g35trfo9nvz4aab8\n| Parent: None\n| \"10\"\n| \n* 2mwypyaqwaxuudmcmn0coehz7nslnnvr21kkixe04k1vb0m993\n| Parent: None\n| \"100\"\n"
	diffRes2  = "*   2fsrpbbtqonc6vddm1hh89osh49h1zam4lpmi6ozhmbv7cadaw\n|\\  Merge: 0b0gcibpbzdmnj5qb74wqmoow2sy0gnpviuhq8g4rhrsiggay3 2mwypyaqwaxuudmcmn0coehz7nslnnvr21kkixe04k1vb0m993\n| | -   \"11\"\n| | +   \"101\"\n| | \n* |   0b0gcibpbzdmnj5qb74wqmoow2sy0gnpviuhq8g4rhrsiggay3\n|\\ \\  Merge: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc 57yam5w501mzvzohxpjm7x83lzkwpg4702g35trfo9nvz4aab8\n| | | -   \"1\"\n| | | +   \"11\"\n| | | \n* | 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| | Parent: None\n| | \n* 57yam5w501mzvzohxpjm7x83lzkwpg4702g35trfo9nvz4aab8\n| Parent: None\n| \n* 2mwypyaqwaxuudmcmn0coehz7nslnnvr21kkixe04k1vb0m993\n| Parent: None\n| \n"

	graphRes3 = "*   5zml97ss3er79n1j7213d7yztdnvu23w7sp8d1zdr0pua6gft6\n|\\  Merge: 38s8tyqj2hpxdokfufv00pafyhzjokxpx3qehlwgqn1aougcuh 4ufn050dt885ckk5u4gtuvicfj20qhyobtvkia02ht9gnbs2po\n| | \"2222-wz\"\n| | \n* |   38s8tyqj2hpxdokfufv00pafyhzjokxpx3qehlwgqn1aougcuh\n|\\ \\  Merge: 4ym4ib87bs97fc4ou7bf13bvhyl9i9ekqu654yjnzkn7p3ih1r 4bgj3l6g78ukpvm26gbk0nw12begs4epx0wpys45ovfnb99eyf\n| | | \"222-wy\"\n| | | \n| * |   4bgj3l6g78ukpvm26gbk0nw12begs4epx0wpys45ovfnb99eyf\n| |\\ \\  Merge: 654ho77pq0uh86899x5bm0sst5l4r1hanseflototeyy0h6vzf 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | \"22-wx\"\n| | | | \n* | | | 4ym4ib87bs97fc4ou7bf13bvhyl9i9ekqu654yjnzkn7p3ih1r\n| | | | Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | \"200-y\"\n| | | | \n| * | | 654ho77pq0uh86899x5bm0sst5l4r1hanseflototeyy0h6vzf\n| | | | Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | \"20-x\"\n| | | | \n| | | * 4ufn050dt885ckk5u4gtuvicfj20qhyobtvkia02ht9gnbs2po\n|/ / /  Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n|       \"2000-z\"\n|       \n* 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| Parent: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| \"2\"\n| \n* 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| Parent: None\n| \"1\"\n"
	diffRes3  = "*   5zml97ss3er79n1j7213d7yztdnvu23w7sp8d1zdr0pua6gft6\n|\\  Merge: 38s8tyqj2hpxdokfufv00pafyhzjokxpx3qehlwgqn1aougcuh 4ufn050dt885ckk5u4gtuvicfj20qhyobtvkia02ht9gnbs2po\n| | -   \"222-wy\"\n| | +   \"2222-wz\"\n| | \n* |   38s8tyqj2hpxdokfufv00pafyhzjokxpx3qehlwgqn1aougcuh\n|\\ \\  Merge: 4ym4ib87bs97fc4ou7bf13bvhyl9i9ekqu654yjnzkn7p3ih1r 4bgj3l6g78ukpvm26gbk0nw12begs4epx0wpys45ovfnb99eyf\n| | | -   \"200-y\"\n| | | +   \"222-wy\"\n| | | \n| * |   4bgj3l6g78ukpvm26gbk0nw12begs4epx0wpys45ovfnb99eyf\n| |\\ \\  Merge: 654ho77pq0uh86899x5bm0sst5l4r1hanseflototeyy0h6vzf 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | -   \"20-x\"\n| | | | +   \"22-wx\"\n| | | | \n* | | | 4ym4ib87bs97fc4ou7bf13bvhyl9i9ekqu654yjnzkn7p3ih1r\n| | | | Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | -   \"2\"\n| | | | +   \"200-y\"\n| | | | \n| * | | 654ho77pq0uh86899x5bm0sst5l4r1hanseflototeyy0h6vzf\n| | | | Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| | | | -   \"2\"\n| | | | +   \"20-x\"\n| | | | \n| | | * 4ufn050dt885ckk5u4gtuvicfj20qhyobtvkia02ht9gnbs2po\n|/ / /  Parent: 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n|       -   \"2\"\n|       +   \"2000-z\"\n|       \n* 5fs0se7wscpzhop97ww3hgzpyskqomblegevrgeb3613y04ndf\n| Parent: 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| -   \"1\"\n| +   \"2\"\n| \n* 0j4ba2t00wvtlc8f1i430u686vq8gak33qpizs9ycxcx3w07pc\n| Parent: None\n| \n"

	truncRes1  = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| List<String>([\n|   \"one\",\n|   \"two\",\n|   \"three\",\n|   \"four\",\n|   \"five\",\n|   \"six\",\n|   \"seven\",\n| ...\n| \n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n| \"the first line\"\n"
	diffTrunc1 = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| -   \"the first line\"\n| +   [\n| +     \"one\",\n| +     \"two\",\n| +     \"three\",\n| +     \"four\",\n| +     \"five\",\n| +     \"six\",\n| ...\n| \n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n| \n"

	truncRes2  = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| List<String>([\n|   \"one\",\n|   \"two\",\n|   \"three\",\n|   \"four\",\n|   \"five\",\n|   \"six\",\n|   \"seven\",\n|   \"eight\",\n|   \"nine\",\n|   \"ten\",\n|   \"eleven\",\n| ])\n| \n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n| \"the first line\"\n"
	diffTrunc2 = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| -   \"the first line\"\n| +   [\n| +     \"one\",\n| +     \"two\",\n| +     \"three\",\n| +     \"four\",\n| +     \"five\",\n| +     \"six\",\n| +     \"seven\",\n| +     \"eight\",\n| +     \"nine\",\n| +     \"ten\",\n| +     \"eleven\",\n| +   ]\n| \n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n| \n"

	truncRes3  = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n"
	diffTrunc3 = "* 4x4qpr9nv7skzdfyv8dpctlsfromlwz7erao2x13dawwnsoot3\n| Parent: 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n* 2ud00v0hv01xfpeoyjwpvkh42w2pgyvlnluissj9ord6uwc87c\n| Parent: None\n"
)
