// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package suite

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

type testSuite struct {
	PerfSuite
	tempFileName string
	atticLabs    string
}

// This is the only test that does anything interesting. The others are just to test naming.
func (s *testSuite) TestInterestingStuff() {
	assert := s.NewAssert()
	assert.NotNil(s.T)
	assert.NotNil(s.W)
	assert.NotEqual("", s.AtticLabs)
	assert.NotEqual("", s.DatabaseSpec)

	val := types.Bool(true)
	r := s.Database.WriteValue(val)
	assert.True(s.Database.ReadValue(r.TargetHash()).Equals(val))

	s.tempFileName = s.TempFile().Name()
	s.Pause(func() {
		s.waitForSmidge()
	})
}

func (s *testSuite) TestFoo() {
	s.waitForSmidge()
}

func (s *testSuite) TestBar() {
	s.waitForSmidge()
}

func (s *testSuite) Test01Abc() {
	s.waitForSmidge()
}

func (s *testSuite) Test02Def() {
	s.waitForSmidge()
}

func (s *testSuite) testNothing() {
	s.waitForSmidge()
}

func (s *testSuite) Testimate() {
	s.waitForSmidge()
}

func (s *testSuite) waitForSmidge() {
	// Tests should call this to make sure the measurement shows up as > 0, not that it shows up as a millisecond.
	<-time.After(time.Millisecond)
}

func TestSuite(t *testing.T) {
	assert := assert.New(t)

	// Write test results to our own temporary LDB database.
	ldbDir, err := ioutil.TempDir("", "suite.TestSuite")
	assert.NoError(err)

	perfFlagVal := *perfFlag
	perfRepeatFlagVal := *perfRepeatFlag
	*perfFlag = ldbDir
	*perfRepeatFlag = 3
	defer func() {
		*perfFlag = perfFlagVal
		*perfRepeatFlag = perfRepeatFlagVal
	}()

	s := &testSuite{}
	Run("ds", t, s)

	// The temp file should have been cleaned up.
	_, err = os.Stat(s.tempFileName)
	assert.NotNil(err)

	// The results should have been written to the "ds" dataset.
	ds, err := spec.GetDataset(ldbDir + "::ds")
	assert.NoError(err)
	head := ds.HeadValue().(types.Struct)

	// The general structure should be correct.
	env := head.Get("environment").(types.Struct)
	env.Get("diskUsages")
	env.Get("cpus")
	env.Get("mem")
	env.Get("host")
	env.Get("partitions")

	assert.NotEqual("", head.Get("nomsVersion"))
	head.Get("testdataVersion") // don't assert it's not empty, it might not be checked out

	// Sanity check test runs.
	runs := head.Get("runs").(types.List)
	assert.Equal(*perfRepeatFlag, int(runs.Len()))

	runs.IterAll(func(run types.Value, _ uint64) {
		expectedTests := []string{"Abc", "Bar", "Def", "Foo", "InterestingStuff"}
		i := 0

		run.(types.Map).IterAll(func(k, timesVal types.Value) {
			assert.True(i < len(expectedTests))
			assert.Equal(expectedTests[i], string(k.(types.String)))

			times := timesVal.(types.Struct)
			assert.True(times.Get("elapsed").(types.Number) > 0)
			assert.True(times.Get("total").(types.Number) > 0)

			if k == types.String("InterestingStuff") {
				assert.True(times.Get("paused").(types.Number) > 0)
			} else {
				assert.True(times.Get("paused").(types.Number) == 0)
			}

			i++
		})

		assert.Equal(i, len(expectedTests))
	})
}
