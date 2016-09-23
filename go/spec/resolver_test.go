// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/attic-labs/testify/assert"
)

const (
	localSpec   = ldbSpec
	remoteSpec  = httpSpec
	testDs      = "testds"
	testObject  = "#pckdvpvr9br1fie6c3pjudrlthe7na18"
)

type testData struct {
	input    string
	expected string
}

var (
	rtestRoot = os.TempDir()

	rtestConfig = &Config{
		"",
		DefaultConfig{ localSpec },
		map[string]DbConfig{ remoteAlias: { remoteSpec } },
	}

	dbTestsNoAliases = []testData {
		{localSpec, localSpec},
		{remoteSpec, remoteSpec},
	}

	dbTestsWithAliases = []testData {
		{"", localSpec},
		{remoteAlias, remoteSpec},
	}

	pathTestsNoAliases = []testData {
		{remoteSpec + "::" + testDs, remoteSpec + "::" + testDs},
		{remoteSpec + "::" + testObject, remoteSpec + "::" + testObject},

	}

	pathTestsWithAliases = []testData {
		{testDs, localSpec + "::" + testDs},
		{remoteAlias + "::" + testDs, remoteSpec + "::" + testDs},
		{testObject, localSpec + "::" + testObject},
		{remoteAlias + "::" + testObject, remoteSpec + "::" + testObject},
	}
)


func withConfig(t *testing.T) *Resolver {
	assert := assert.New(t)
	dir := filepath.Join(rtestRoot, "with-config")
	_, err := rtestConfig.WriteTo(dir)
	assert.NoError(err, dir)
	assert.NoError(os.Chdir(dir))
	r := NewResolver() // resolver must be created after changing directory
	return r

}

func withoutConfig(t *testing.T) *Resolver {
	assert := assert.New(t)
	dir := filepath.Join(rtestRoot, "without-config")
	assert.NoError(os.MkdirAll(dir, os.ModePerm), dir)
	assert.NoError(os.Chdir(dir))
	r := NewResolver() // resolver must be created after changing directory
	return r
}

func assertPathSpecsEquiv(assert *assert.Assertions, expected string, actual string) {
	e, err := parsePathSpec(expected)
	assert.NoError(err)
	a, err := parsePathSpec(actual)
	assert.NoError(err)
	assertDbSpecsEquiv(assert, e.DbSpec.String(), a.DbSpec.String())
	assert.Equal(e.Path.String(), a.Path.String())
}

func TestResolveDatabaseWithConfig(t *testing.T) {
	spec := withConfig(t)
	assert := assert.New(t)
	for _, d := range append(dbTestsNoAliases, dbTestsWithAliases...) {
		db := spec.ResolveDatabase(d.input)
		assertDbSpecsEquiv(assert, d.expected, db)
	}
}

func TestResolvePathWithConfig(t *testing.T) {
	spec := withConfig(t)
	assert := assert.New(t)
	for _, d := range append(pathTestsNoAliases, pathTestsWithAliases...) {
		path := spec.ResolvePath(d.input)
		assertPathSpecsEquiv(assert, d.expected, path)
	}
}

func TestResolveDatabaseWithoutConfig(t *testing.T) {
	spec := withoutConfig(t)
	assert := assert.New(t)
	for _, d := range dbTestsNoAliases {
		db := spec.ResolveDatabase(d.input)
		assert.Equal(d.expected, db, d.input)
	}
}

func TestResolvePathWithoutConfig(t *testing.T) {
	spec := withoutConfig(t)
	assert := assert.New(t)
	for _, d := range pathTestsNoAliases {
		path := spec.ResolvePath(d.input)
		assertPathSpecsEquiv(assert, d.expected, path)
	}

}
