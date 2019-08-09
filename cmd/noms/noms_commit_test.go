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
	"github.com/stretchr/testify/suite"
)

type nomsCommitTestSuite struct {
	clienttest.ClientTestSuite
}

func TestNomsCommit(t *testing.T) {
	suite.Run(t, &nomsCommitTestSuite{})
}

func (s *nomsCommitTestSuite) setupDataset(name string, doCommit bool) (sp spec.Spec, ref types.Ref) {
	var err error
	sp, err = spec.ForDataset(spec.CreateValueSpecString("nbs", s.DBDir, name))
	s.NoError(err)

	v := types.String("testcommit")
	ref = sp.GetDatabase().WriteValue(v)

	if doCommit {
		_, err = sp.GetDatabase().CommitValue(sp.GetDataset(), ref)
		s.NoError(err)
	}
	return
}

func (s *nomsCommitTestSuite) TestNomsCommitToDatasetWithoutHead() {
	sp, ref := s.setupDataset("commitTest", false)
	defer sp.Close()

	_, ok := sp.GetDataset().MaybeHead()
	s.False(ok, "should not have a commit")

	stdoutString, stderrString := s.MustRun(main, []string{"commit", "#" + ref.TargetHash().String(), sp.String()})
	s.Empty(stderrString)
	s.Contains(stdoutString, "New head #")

	sp, _ = spec.ForDataset(sp.String())
	defer sp.Close()

	commit, ok := sp.GetDataset().MaybeHead()
	s.True(ok, "should have a commit now")
	value := commit.Get(datas.ValueField)
	s.True(value.Hash() == ref.TargetHash(), "commit.value hash == writevalue hash")

	meta := commit.Get(datas.MetaField).(types.Struct)
	s.NotEmpty(meta.Get("date"))
}

func structFieldEqual(old, now types.Struct, field string) bool {
	oldValue, oldOk := old.MaybeGet(field)
	nowValue, nowOk := now.MaybeGet(field)
	return oldOk && nowOk && nowValue.Equals(oldValue)
}

func (s *nomsCommitTestSuite) runDuplicateTest(allowDuplicate bool) {
	dsName := "commitTestDuplicate"
	sp, ref := s.setupDataset(dsName, true)
	defer sp.Close()

	_, ok := sp.GetDataset().MaybeHeadValue()
	s.True(ok, "should have a commit")

	cliOptions := []string{"commit"}
	if allowDuplicate {
		cliOptions = append(cliOptions, "--allow-dupe")
	}
	cliOptions = append(cliOptions, dsName+".value", sp.String())

	stdoutString, stderrString, err := s.Run(main, cliOptions)
	s.Nil(err)
	s.Empty(stderrString)
	if allowDuplicate {
		s.NotContains(stdoutString, "Commit aborted")
		s.Contains(stdoutString, "New head #")
	} else {
		s.Contains(stdoutString, "Commit aborted")
	}

	sp, _ = spec.ForDataset(sp.String())
	defer sp.Close()

	value, ok := sp.GetDataset().MaybeHeadValue()
	s.True(ok, "should still have a commit")
	s.True(value.Hash() == ref.Hash(), "commit.value hash == previous commit hash")
}

func (s *nomsCommitTestSuite) TestNomsCommitDuplicate() {
	s.runDuplicateTest(false)
	s.runDuplicateTest(true)
}

func (s *nomsCommitTestSuite) TestNomsCommitMetadata() {
	dsName := "commitTestMetadata"
	sp, _ := s.setupDataset(dsName, true)
	defer sp.Close()

	metaOld := sp.GetDataset().Head().Get(datas.MetaField).(types.Struct)

	stdoutString, stderrString, err := s.Run(main, []string{"commit", "--allow-dupe", "--message=foo", dsName + ".value", sp.String()})
	s.Nil(err)
	s.Empty(stderrString)
	s.Contains(stdoutString, "New head #")

	sp, _ = spec.ForDataset(sp.String())
	defer sp.Close()

	metaNew := sp.GetDataset().Head().Get(datas.MetaField).(types.Struct)

	s.False(metaOld.Equals(metaNew), "meta didn't change")
	s.False(structFieldEqual(metaOld, metaNew, "date"), "date didn't change")
	s.False(structFieldEqual(metaOld, metaNew, "message"), "message didn't change")
	s.True(metaNew.Get("message").Equals(types.String("foo")), "message wasn't set")

	metaOld = metaNew

	stdoutString, stderrString = s.MustRun(main, []string{"commit", "--allow-dupe", "--message=bar", "--date=" + spec.CommitMetaDateFormat[:20], dsName + ".value", sp.String()})
	s.Empty(stderrString)
	s.Contains(stdoutString, "New head #")

	sp, _ = spec.ForDataset(sp.String())
	defer sp.Close()

	metaNew = sp.GetDataset().Head().Get(datas.MetaField).(types.Struct)

	s.False(metaOld.Equals(metaNew), "meta didn't change")
	s.False(structFieldEqual(metaOld, metaNew, "date"), "date didn't change")
	s.False(structFieldEqual(metaOld, metaNew, "message"), "message didn't change")
	s.True(metaNew.Get("message").Equals(types.String("bar")), "message wasn't set")
}

func (s *nomsCommitTestSuite) TestNomsCommitHashNotFound() {
	sp, _ := s.setupDataset("commitTestBadHash", true)
	defer sp.Close()

	s.Panics(func() {
		s.MustRun(main, []string{"commit", "#9ei6fbrs0ujo51vifd3f2eebufo4lgdu", sp.String()})
	})
}

func (s *nomsCommitTestSuite) TestNomsCommitMetadataBadDateFormat() {
	sp, ref := s.setupDataset("commitTestMetadata", true)
	defer sp.Close()

	s.Panics(func() {
		s.MustRun(main, []string{"commit", "--allow-dupe", "--date=a", "#" + ref.TargetHash().String(), sp.String()})
	})
}
