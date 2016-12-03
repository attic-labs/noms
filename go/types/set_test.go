// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

const testSetSize = 5000

type testSet ValueSlice

func (ts testSet) Remove(from, to int) testSet {
	values := make(testSet, 0, len(ts)-(to-from))
	values = append(values, ts[:from]...)
	values = append(values, ts[to:]...)
	return values
}

func (ts testSet) Has(key Value) bool {
	for _, entry := range ts {
		if entry.Equals(key) {
			return true
		}
	}
	return false
}

func (ts testSet) Diff(last testSet) (added []Value, removed []Value) {
	// Note: this could be use ts.toSet/last.toSet and then tsSet.Diff(lastSet) but the
	// purpose of this method is to be redundant.
	if len(ts) == 0 && len(last) == 0 {
		return // nothing changed
	}
	if len(ts) == 0 {
		// everything removed
		for _, entry := range last {
			removed = append(removed, entry)
		}
		return
	}
	if len(last) == 0 {
		// everything added
		for _, entry := range ts {
			added = append(added, entry)
		}
		return
	}
	for _, entry := range ts {
		if !last.Has(entry) {
			added = append(added, entry)
		}
	}
	for _, entry := range last {
		if !ts.Has(entry) {
			removed = append(removed, entry)
		}
	}
	return
}

func (ts testSet) toSet() Set {
	return NewSet(ts...)
}

func newSortedTestSet(length int, gen genValueFn) (values testSet) {
	for i := 0; i < length; i++ {
		values = append(values, gen(i))
	}
	return
}

func newTestSetFromSet(s Set) testSet {
	values := make([]Value, 0, s.Len())
	s.IterAll(func(v Value) {
		values = append(values, v)
	})
	return values
}

func newRandomTestSet(length int, gen genValueFn) testSet {
	s := rand.NewSource(4242)
	used := map[int]bool{}

	var values []Value
	for len(values) < length {
		v := int(s.Int63()) & 0xffffff
		if _, ok := used[v]; !ok {
			values = append(values, gen(v))
			used[v] = true
		}
	}

	return values
}

func validateSet(t *testing.T, s Set, values ValueSlice) {
	assert.True(t, s.Equals(NewSet(values...)))
	out := ValueSlice{}
	s.IterAll(func(v Value) {
		out = append(out, v)
	})
	assert.True(t, out.Equals(values))
}

type setTestSuite struct {
	collectionTestSuite
	elems testSet
}

func newSetTestSuite(size uint, expectRefStr string, expectChunkCount int, expectPrependChunkDiff int, expectAppendChunkDiff int, gen genValueFn) *setTestSuite {
	length := 1 << size
	elemType := gen(0).Type()
	elems := newSortedTestSet(length, gen)
	tr := MakeSetType(elemType)
	set := NewSet(elems...)
	return &setTestSuite{
		collectionTestSuite: collectionTestSuite{
			col:                    set,
			expectType:             tr,
			expectLen:              uint64(length),
			expectRef:              expectRefStr,
			expectChunkCount:       expectChunkCount,
			expectPrependChunkDiff: expectPrependChunkDiff,
			expectAppendChunkDiff:  expectAppendChunkDiff,
			validate: func(v2 Collection) bool {
				l2 := v2.(Set)
				out := ValueSlice{}
				l2.IterAll(func(v Value) {
					out = append(out, v)
				})
				return ValueSlice(elems).Equals(out)
			},
			prependOne: func() Collection {
				dup := make([]Value, length+1)
				dup[0] = Number(-1)
				copy(dup[1:], elems)
				return NewSet(dup...)
			},
			appendOne: func() Collection {
				dup := make([]Value, length+1)
				copy(dup, elems)
				dup[len(dup)-1] = Number(length + 1)
				return NewSet(dup...)
			},
		},
		elems: elems,
	}
}

func (suite *setTestSuite) createStreamingSet(vs *ValueStore) {
	randomized := make(ValueSlice, len(suite.elems))
	for i, j := range rand.Perm(len(randomized)) {
		randomized[j] = suite.elems[i]
	}

	vChan := make(chan Value)
	setChan := NewStreamingSet(vs, vChan)
	for _, entry := range randomized {
		vChan <- entry
	}
	close(vChan)
	suite.True(suite.validate(<-setChan))
}

func (suite *setTestSuite) TestStreamingSet() {
	vs := NewTestValueStore()
	defer vs.Close()
	suite.createStreamingSet(vs)
}

func (suite *setTestSuite) TestStreamingSet2() {
	vs := NewTestValueStore()
	defer vs.Close()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		suite.createStreamingSet(vs)
		wg.Done()
	}()
	go func() {
		suite.createStreamingSet(vs)
		wg.Done()
	}()
	wg.Wait()
}

func TestSetSuite1K(t *testing.T) {
	suite.Run(t, newSetTestSuite(10, "n99i86gc4s23ol7ctmjuc1p4jk4msr4i", 0, 0, 0, newNumber))
}

func TestSetSuite4K(t *testing.T) {
	suite.Run(t, newSetTestSuite(12, "8i0nfi4vf4ilp6g5bb4uhlqpbie0rbrh", 2, 2, 2, newNumber))
}

func TestSetSuite1KStructs(t *testing.T) {
	suite.Run(t, newSetTestSuite(10, "901lc62988o1epbj29811f5f0u06ep8g", 0, 0, 0, newNumberStruct))
}

func TestSetSuite4KStructs(t *testing.T) {
	suite.Run(t, newSetTestSuite(12, "410of8pb7ib5rms4b611490brueqcc7c", 4, 2, 2, newNumberStruct))
}

func getTestNativeOrderSet(scale int) testSet {
	return newRandomTestSet(64*scale, newNumber)
}

func getTestRefValueOrderSet(scale int) testSet {
	return newRandomTestSet(64*scale, newNumber)
}

func getTestRefToNativeOrderSet(scale int, vw ValueWriter) testSet {
	return newRandomTestSet(64*scale, func(v int) Value {
		return vw.WriteValue(Number(v))
	})
}

func getTestRefToValueOrderSet(scale int, vw ValueWriter) testSet {
	return newRandomTestSet(64*scale, func(v int) Value {
		return vw.WriteValue(NewSet(Number(v)))
	})
}

func accumulateSetDiffChanges(s1, s2 Set) (added []Value, removed []Value) {
	changes := make(chan ValueChanged)
	go func() {
		s1.Diff(s2, changes, nil)
		close(changes)
	}()
	for change := range changes {
		if change.ChangeType == DiffChangeAdded {
			added = append(added, change.V)
		} else if change.ChangeType == DiffChangeRemoved {
			removed = append(removed, change.V)
		}
	}
	return
}

func diffSetTest(assert *assert.Assertions, s1 Set, s2 Set, numAddsExpected int, numRemovesExpected int) (added []Value, removed []Value) {
	added, removed = accumulateSetDiffChanges(s1, s2)
	assert.Equal(numAddsExpected, len(added), "num added is not as expected")
	assert.Equal(numRemovesExpected, len(removed), "num removed is not as expected")

	ts1 := newTestSetFromSet(s1)
	ts2 := newTestSetFromSet(s2)
	tsAdded, tsRemoved := ts1.Diff(ts2)
	assert.Equal(numAddsExpected, len(tsAdded), "num added is not as expected")
	assert.Equal(numRemovesExpected, len(tsRemoved), "num removed is not as expected")

	assert.Equal(added, tsAdded, "set added != tsSet added")
	assert.Equal(removed, tsRemoved, "set removed != tsSet removed")
	return
}

func TestNewSet(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.True(MakeSetType(MakeUnionType()).Equals(s.Type()))
	assert.Equal(uint64(0), s.Len())

	s = NewSet(Number(0))
	assert.True(MakeSetType(NumberType).Equals(s.Type()))

	s = NewSet()
	assert.IsType(MakeSetType(NumberType), s.Type())

	s2 := s.Remove(Number(1))
	assert.IsType(s.Type(), s2.Type())
}

func TestSetLen(t *testing.T) {
	assert := assert.New(t)
	s0 := NewSet()
	assert.Equal(uint64(0), s0.Len())
	s1 := NewSet(Bool(true), Number(1), String("hi"))
	assert.Equal(uint64(3), s1.Len())
	diffSetTest(assert, s0, s1, 0, 3)
	diffSetTest(assert, s1, s0, 3, 0)

	s2 := s1.Insert(Bool(false))
	assert.Equal(uint64(4), s2.Len())
	diffSetTest(assert, s0, s2, 0, 4)
	diffSetTest(assert, s2, s0, 4, 0)
	diffSetTest(assert, s1, s2, 0, 1)
	diffSetTest(assert, s2, s1, 1, 0)

	s3 := s2.Remove(Bool(true))
	assert.Equal(uint64(3), s3.Len())
	diffSetTest(assert, s2, s3, 1, 0)
	diffSetTest(assert, s3, s2, 0, 1)
}

func TestSetEmpty(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.True(s.Empty())
	assert.Equal(uint64(0), s.Len())
}

func TestSetEmptyInsert(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.True(s.Empty())
	s = s.Insert(Bool(false))
	assert.False(s.Empty())
	assert.Equal(uint64(1), s.Len())
}

func TestSetEmptyInsertRemove(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.True(s.Empty())
	s = s.Insert(Bool(false))
	assert.False(s.Empty())
	assert.Equal(uint64(1), s.Len())
	s = s.Remove(Bool(false))
	assert.True(s.Empty())
	assert.Equal(uint64(0), s.Len())
}

// BUG 98
func TestSetDuplicateInsert(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(Bool(true), Number(42), Number(42))
	assert.Equal(uint64(2), s1.Len())
}

func TestSetUniqueKeysString(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(String("hello"), String("world"), String("hello"))
	assert.Equal(uint64(2), s1.Len())
	assert.True(s1.Has(String("hello")))
	assert.True(s1.Has(String("world")))
	assert.False(s1.Has(String("foo")))
}

func TestSetUniqueKeysNumber(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(Number(4), Number(1), Number(0), Number(0), Number(1), Number(3))
	assert.Equal(uint64(4), s1.Len())
	assert.True(s1.Has(Number(4)))
	assert.True(s1.Has(Number(1)))
	assert.True(s1.Has(Number(0)))
	assert.True(s1.Has(Number(3)))
	assert.False(s1.Has(Number(2)))
}

func TestSetHas(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(Bool(true), Number(1), String("hi"))
	assert.True(s1.Has(Bool(true)))
	assert.False(s1.Has(Bool(false)))
	assert.True(s1.Has(Number(1)))
	assert.False(s1.Has(Number(0)))
	assert.True(s1.Has(String("hi")))
	assert.False(s1.Has(String("ho")))

	s2 := s1.Insert(Bool(false))
	assert.True(s2.Has(Bool(false)))
	assert.True(s2.Has(Bool(true)))

	assert.True(s1.Has(Bool(true)))
	assert.False(s1.Has(Bool(false)))
}

func TestSetHas2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	smallTestChunks()
	defer normalProductionChunks()

	assert := assert.New(t)

	vs := NewTestValueStore()
	doTest := func(ts testSet) {
		set := ts.toSet()
		set2 := vs.ReadValue(vs.WriteValue(set).TargetHash()).(Set)
		for _, v := range ts {
			assert.True(set.Has(v))
			assert.True(set2.Has(v))
		}
		diffSetTest(assert, set, set2, 0, 0)
	}

	doTest(getTestNativeOrderSet(16))
	doTest(getTestRefValueOrderSet(2))
	doTest(getTestRefToNativeOrderSet(2, vs))
	doTest(getTestRefToValueOrderSet(2, vs))
}

func validateSetInsertion(t *testing.T, values ValueSlice) {
	s := NewSet()
	for i, v := range values {
		s = s.Insert(v)
		validateSet(t, s, values[0:i+1])
	}
}

func TestSetValidateInsertAscending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	smallTestChunks()
	defer normalProductionChunks()

	validateSetInsertion(t, generateNumbersAsValues(300))
}

func TestSetInsert(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	v1 := Bool(false)
	v2 := Bool(true)
	v3 := Number(0)

	assert.False(s.Has(v1))
	s = s.Insert(v1)
	assert.True(s.Has(v1))
	s = s.Insert(v2)
	assert.True(s.Has(v1))
	assert.True(s.Has(v2))
	s2 := s.Insert(v3)
	assert.True(s.Has(v1))
	assert.True(s.Has(v2))
	assert.False(s.Has(v3))
	assert.True(s2.Has(v1))
	assert.True(s2.Has(v2))
	assert.True(s2.Has(v3))
}

func TestSetInsert2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	smallTestChunks()
	defer normalProductionChunks()

	assert := assert.New(t)

	doTest := func(incr, offset int, ts testSet) {
		expected := ts.toSet()
		run := func(from, to int) {
			actual := ts.Remove(from, to).toSet().Insert(ts[from:to]...)
			assert.Equal(expected.Len(), actual.Len())
			assert.True(expected.Equals(actual))
			diffSetTest(assert, expected, actual, 0, 0)
		}
		for i := 0; i < len(ts)-offset; i += incr {
			run(i, i+offset)
		}
		run(len(ts)-offset, len(ts))
	}

	doTest(18, 3, getTestNativeOrderSet(9))
	doTest(64, 1, getTestNativeOrderSet(32))
	doTest(32, 1, getTestRefValueOrderSet(4))
	doTest(32, 1, getTestRefToNativeOrderSet(4, NewTestValueStore()))
	doTest(32, 1, getTestRefToValueOrderSet(4, NewTestValueStore()))
}

func TestSetInsertExistingValue(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	ts := getTestNativeOrderSet(2)
	original := ts.toSet()
	actual := original.Insert(ts[0])

	assert.Equal(original.Len(), actual.Len())
	assert.True(original.Equals(actual))
}

func TestSetRemove(t *testing.T) {
	assert := assert.New(t)
	v1 := Bool(false)
	v2 := Bool(true)
	v3 := Number(0)
	s := NewSet(v1, v2, v3)
	assert.True(s.Has(v1))
	assert.True(s.Has(v2))
	assert.True(s.Has(v3))
	s = s.Remove(v1)
	assert.False(s.Has(v1))
	assert.True(s.Has(v2))
	assert.True(s.Has(v3))
	s2 := s.Remove(v2)
	assert.False(s.Has(v1))
	assert.True(s.Has(v2))
	assert.True(s.Has(v3))
	assert.False(s2.Has(v1))
	assert.False(s2.Has(v2))
	assert.True(s2.Has(v3))
}

func TestSetRemove2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	smallTestChunks()
	defer normalProductionChunks()

	assert := assert.New(t)

	doTest := func(incr, offset int, ts testSet) {
		whole := ts.toSet()
		run := func(from, to int) {
			expected := ts.Remove(from, to).toSet()
			actual := whole.Remove(ts[from:to]...)
			assert.Equal(expected.Len(), actual.Len())
			assert.True(expected.Equals(actual))
			diffSetTest(assert, expected, actual, 0, 0)
		}
		for i := 0; i < len(ts)-offset; i += incr {
			run(i, i+offset)
		}
		run(len(ts)-offset, len(ts))
	}

	doTest(18, 3, getTestNativeOrderSet(9))
	doTest(64, 1, getTestNativeOrderSet(32))
	doTest(32, 1, getTestRefValueOrderSet(4))
	doTest(32, 1, getTestRefToNativeOrderSet(4, NewTestValueStore()))
	doTest(32, 1, getTestRefToValueOrderSet(4, NewTestValueStore()))
}

func TestSetRemoveNonexistentValue(t *testing.T) {
	assert := assert.New(t)

	ts := getTestNativeOrderSet(2)
	original := ts.toSet()
	actual := original.Remove(Number(-1)) // rand.Int63 returns non-negative values.

	assert.Equal(original.Len(), actual.Len())
	assert.True(original.Equals(actual))
}

func TestSetFirst(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.Nil(s.First())
	s = s.Insert(Number(1))
	assert.NotNil(s.First())
	s = s.Insert(Number(2))
	assert.NotNil(s.First())
	s2 := s.Remove(Number(1))
	assert.NotNil(s2.First())
	s2 = s2.Remove(Number(2))
	assert.Nil(s2.First())
}

func TestSetOfStruct(t *testing.T) {
	assert := assert.New(t)

	typ := MakeStructType("S1", []string{"o"}, []*Type{NumberType})

	elems := []Value{}
	for i := 0; i < 200; i++ {
		elems = append(elems, NewStructWithType(typ, ValueSlice{Number(i)}))
	}

	s := NewSet(elems...)
	for i := 0; i < 200; i++ {
		assert.True(s.Has(elems[i]))
	}
}

func TestSetIter(t *testing.T) {
	assert := assert.New(t)
	s := NewSet(Number(0), Number(1), Number(2), Number(3), Number(4))
	acc := NewSet()
	s.Iter(func(v Value) bool {
		_, ok := v.(Number)
		assert.True(ok)
		acc = acc.Insert(v)
		return false
	})
	assert.True(s.Equals(acc))

	acc = NewSet()
	s.Iter(func(v Value) bool {
		return true
	})
	assert.True(acc.Empty())
}

func TestSetIter2(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	doTest := func(ts testSet) {
		set := ts.toSet()
		sort.Sort(ValueSlice(ts))
		idx := uint64(0)
		endAt := uint64(64)

		set.Iter(func(v Value) (done bool) {
			assert.True(ts[idx].Equals(v))
			if idx == endAt {
				done = true
			}
			idx++
			return
		})

		assert.Equal(endAt, idx-1)
	}

	doTest(getTestNativeOrderSet(16))
	doTest(getTestRefValueOrderSet(2))
	doTest(getTestRefToNativeOrderSet(2, NewTestValueStore()))
	doTest(getTestRefToValueOrderSet(2, NewTestValueStore()))
}

func TestSetIterAll(t *testing.T) {
	assert := assert.New(t)
	s := NewSet(Number(0), Number(1), Number(2), Number(3), Number(4))
	acc := NewSet()
	s.IterAll(func(v Value) {
		_, ok := v.(Number)
		assert.True(ok)
		acc = acc.Insert(v)
	})
	assert.True(s.Equals(acc))
}

func TestSetIterAll2(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	doTest := func(ts testSet) {
		set := ts.toSet()
		sort.Sort(ValueSlice(ts))
		idx := uint64(0)

		set.IterAll(func(v Value) {
			assert.True(ts[idx].Equals(v))
			idx++
		})
	}

	doTest(getTestNativeOrderSet(16))
	doTest(getTestRefValueOrderSet(2))
	doTest(getTestRefToNativeOrderSet(2, NewTestValueStore()))
	doTest(getTestRefToValueOrderSet(2, NewTestValueStore()))
}

func testSetOrder(assert *assert.Assertions, valueType *Type, value []Value, expectOrdering []Value) {
	m := NewSet(value...)
	i := 0
	m.IterAll(func(value Value) {
		assert.Equal(expectOrdering[i].Hash().String(), value.Hash().String())
		i++
	})
}

func TestSetOrdering(t *testing.T) {
	assert := assert.New(t)

	testSetOrder(assert,
		StringType,
		[]Value{
			String("a"),
			String("z"),
			String("b"),
			String("y"),
			String("c"),
			String("x"),
		},
		[]Value{
			String("a"),
			String("b"),
			String("c"),
			String("x"),
			String("y"),
			String("z"),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			Number(0),
			Number(1000),
			Number(1),
			Number(100),
			Number(2),
			Number(10),
		},
		[]Value{
			Number(0),
			Number(1),
			Number(2),
			Number(10),
			Number(100),
			Number(1000),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			Number(0),
			Number(-30),
			Number(25),
			Number(1002),
			Number(-5050),
			Number(23),
		},
		[]Value{
			Number(-5050),
			Number(-30),
			Number(0),
			Number(23),
			Number(25),
			Number(1002),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			Number(0.0001),
			Number(0.000001),
			Number(1),
			Number(25.01e3),
			Number(-32.231123e5),
			Number(23),
		},
		[]Value{
			Number(-32.231123e5),
			Number(0.000001),
			Number(0.0001),
			Number(1),
			Number(23),
			Number(25.01e3),
		},
	)

	testSetOrder(assert,
		ValueType,
		[]Value{
			String("a"),
			String("z"),
			String("b"),
			String("y"),
			String("c"),
			String("x"),
		},
		// Ordered by value
		[]Value{
			String("a"),
			String("b"),
			String("c"),
			String("x"),
			String("y"),
			String("z"),
		},
	)

	testSetOrder(assert,
		BoolType,
		[]Value{
			Bool(true),
			Bool(false),
		},
		// Ordered by value
		[]Value{
			Bool(false),
			Bool(true),
		},
	)
}

func TestSetType(t *testing.T) {
	assert := assert.New(t)

	s := NewSet()
	assert.True(s.Type().Equals(MakeSetType(MakeUnionType())))

	s = NewSet(Number(0))
	assert.True(s.Type().Equals(MakeSetType(NumberType)))

	s2 := s.Remove(Number(1))
	assert.True(s2.Type().Equals(MakeSetType(NumberType)))

	s2 = s.Insert(Number(0), Number(1))
	assert.True(s.Type().Equals(s2.Type()))

	s3 := s.Insert(Bool(true))
	assert.True(s3.Type().Equals(MakeSetType(MakeUnionType(BoolType, NumberType))))
	s4 := s.Insert(Number(3), Bool(true))
	assert.True(s4.Type().Equals(MakeSetType(MakeUnionType(BoolType, NumberType))))
}

func TestSetChunks(t *testing.T) {
	assert := assert.New(t)

	l1 := NewSet(Number(0))
	c1 := getChunks(l1)
	assert.Len(c1, 0)

	l2 := NewSet(NewRef(Number(0)))
	c2 := getChunks(l2)
	assert.Len(c2, 1)
}

func TestSetChunks2(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	vs := NewTestValueStore()
	doTest := func(ts testSet) {
		set := ts.toSet()
		set2chunks := getChunks(vs.ReadValue(vs.WriteValue(set).TargetHash()))
		for i, r := range getChunks(set) {
			assert.True(r.Type().Equals(set2chunks[i].Type()), "%s != %s", r.Type().Describe(), set2chunks[i].Type().Describe())
		}
	}

	doTest(getTestNativeOrderSet(16))
	doTest(getTestRefValueOrderSet(2))
	doTest(getTestRefToNativeOrderSet(2, vs))
	doTest(getTestRefToValueOrderSet(2, vs))
}

func TestSetFirstNNumbers(t *testing.T) {
	assert := assert.New(t)

	nums := generateNumbersAsValues(testSetSize)
	s := NewSet(nums...)
	assert.Equal("hius38tca4nfd5lveqe3h905ass99uq2", s.Hash().String())
	assert.Equal(deriveCollectionHeight(s), getRefHeightOfCollection(s))
}

func TestSetRefOfStructFirstNNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	nums := generateNumbersAsRefOfStructs(testSetSize)
	s := NewSet(nums...)
	assert.Equal("n9thkv66vt7c35khatmdr84rhk8ip9tv", s.Hash().String())
	// height + 1 because the leaves are Ref values (with height 1).
	assert.Equal(deriveCollectionHeight(s)+1, getRefHeightOfCollection(s))
}

func TestSetModifyAfterRead(t *testing.T) {
	smallTestChunks()
	defer normalProductionChunks()

	assert := assert.New(t)
	vs := NewTestValueStore()
	set := getTestNativeOrderSet(2).toSet()
	// Drop chunk values.
	set = vs.ReadValue(vs.WriteValue(set).TargetHash()).(Set)
	// Modify/query. Once upon a time this would crash.
	fst := set.First()
	set = set.Remove(fst)
	assert.False(set.Has(fst))
	assert.True(set.Has(set.First()))
	set = set.Insert(fst)
	assert.True(set.Has(fst))
}

func TestSetTypeAfterMutations(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	test := func(n int, c interface{}) {
		values := generateNumbersAsValues(n)

		s := NewSet(values...)
		assert.Equal(s.Len(), uint64(n))
		assert.IsType(c, s.sequence())
		assert.True(s.Type().Equals(MakeSetType(NumberType)))

		s = s.Insert(String("a"))
		assert.Equal(s.Len(), uint64(n+1))
		assert.IsType(c, s.sequence())
		assert.True(s.Type().Equals(MakeSetType(MakeUnionType(NumberType, StringType))))

		s = s.Remove(String("a"))
		assert.Equal(s.Len(), uint64(n))
		assert.IsType(c, s.sequence())
		assert.True(s.Type().Equals(MakeSetType(NumberType)))
	}

	test(10, setLeafSequence{})
	test(2000, metaSequence{})
}

func TestChunkedSetWithValuesOfEveryType(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	vs := []Value{
		// Values
		Bool(true),
		Number(0),
		String("hello"),
		NewBlob(bytes.NewBufferString("buf")),
		NewSet(Bool(true)),
		NewList(Bool(true)),
		NewMap(Bool(true), Number(0)),
		NewStruct("", StructData{"field": Bool(true)}),
		// Refs of values
		NewRef(Bool(true)),
		NewRef(Number(0)),
		NewRef(String("hello")),
		NewRef(NewBlob(bytes.NewBufferString("buf"))),
		NewRef(NewSet(Bool(true))),
		NewRef(NewList(Bool(true))),
		NewRef(NewMap(Bool(true), Number(0))),
		NewRef(NewStruct("", StructData{"field": Bool(true)})),
	}

	s := NewSet(vs...)
	for i := 1; !isMetaSequence(s.sequence()); i++ {
		v := Number(i)
		vs = append(vs, v)
		s = s.Insert(v)
	}

	assert.Equal(len(vs), int(s.Len()))
	assert.True(bool(s.First().(Bool)))

	for _, v := range vs {
		assert.True(s.Has(v))
	}

	for len(vs) > 0 {
		v := vs[0]
		vs = vs[1:]
		s = s.Remove(v)
		assert.False(s.Has(v))
		assert.Equal(len(vs), int(s.Len()))
	}
}

func TestSetRemoveLastWhenNotLoaded(t *testing.T) {
	assert := assert.New(t)

	smallTestChunks()
	defer normalProductionChunks()

	vs := NewTestValueStore()
	reload := func(s Set) Set {
		return vs.ReadValue(vs.WriteValue(s).TargetHash()).(Set)
	}

	ts := getTestNativeOrderSet(8)
	ns := ts.toSet()

	for len(ts) > 0 {
		last := ts[len(ts)-1]
		ts = ts[:len(ts)-1]
		ns = reload(ns.Remove(last))
		assert.True(ts.toSet().Equals(ns))
	}
}

func TestSetAt(t *testing.T) {
	assert := assert.New(t)

	values := []Value{Bool(false), Number(42), String('a'), String('b'), String('c')}
	s := NewSet(values...)

	for i, v := range values {
		assert.Equal(v, s.At(uint64(i)))
	}

	assert.Panics(func() {
		s.At(42)
	})
}
