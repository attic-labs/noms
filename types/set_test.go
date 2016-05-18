package types

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const testSetSize = 5000

type testSetGenFn func(v Number) Value

type testSet struct {
	values []Value
	tr     *Type
}

func (ts testSet) Len() int {
	return len(ts.values)
}

func (ts testSet) Less(i, j int) bool {
	return ts.values[i].Less(ts.values[j])
}

func (ts testSet) Swap(i, j int) {
	ts.values[i], ts.values[j] = ts.values[j], ts.values[i]
}

func (ts testSet) Remove(from, to int) testSet {
	values := make([]Value, 0, len(ts.values)-(to-from))
	values = append(values, ts.values[:from]...)
	values = append(values, ts.values[to:]...)
	return testSet{values, ts.tr}
}

func (ts testSet) toSet() Set {
	return NewSet(ts.values...)
}

func newTestSet(length int) testSet {
	var values []Value
	for i := 0; i < length; i++ {
		values = append(values, NewNumber(i))
	}

	return testSet{values, MakeSetType(NumberType)}
}

func newTestSetWithGen(length int, gen testSetGenFn, tr *Type) testSet {
	s := rand.NewSource(4242)
	used := map[int64]bool{}

	var values []Value
	for len(values) < length {
		v := s.Int63() & 0xffffff
		if _, ok := used[v]; !ok {
			values = append(values, gen(NewNumber(v)))
			used[v] = true
		}
	}

	return testSet{values, MakeSetType(tr)}
}

type setTestSuite struct {
	collectionTestSuite
	elems testSet
}

func newSetTestSuite(size uint, expectRefStr string, expectChunkCount int, expectPrependChunkDiff int, expectAppendChunkDiff int) *setTestSuite {
	length := 1 << size
	elems := newTestSet(length)
	tr := MakeSetType(NumberType)
	set := NewSet(elems.values...)
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
				out := []Value{}
				l2.IterAll(func(v Value) {
					out = append(out, v)
				})
				return valueSlicesEqual(elems.values, out)
			},
			prependOne: func() Collection {
				dup := make([]Value, length+1)
				dup[0] = NewNumber(-1)
				copy(dup[1:], elems.values)
				return NewSet(dup...)
			},
			appendOne: func() Collection {
				dup := make([]Value, length+1)
				copy(dup, elems.values)
				dup[len(dup)-1] = NewNumber(length + 1)
				return NewSet(dup...)
			},
		},
		elems: elems,
	}
}

func TestSetSuite1K(t *testing.T) {
	suite.Run(t, newSetTestSuite(10, "sha1-8836444230d08c68f55d936268350b6d148c4f88", 16, 2, 2))
}

func TestSetSuite4K(t *testing.T) {
	suite.Run(t, newSetTestSuite(12, "sha1-9831a1058d5ddddb269900704566e5e3697e7ac9", 3, 2, 2))
}

func getTestNativeOrderSet(scale int) testSet {
	return newTestSetWithGen(int(setPattern)*scale, func(v Number) Value {
		return v
	}, NumberType)
}

func getTestRefValueOrderSet(scale int) testSet {
	setType := MakeSetType(NumberType)
	return newTestSetWithGen(int(setPattern)*scale, func(v Number) Value {
		return NewSet(v)
	}, setType)
}

func getTestRefToNativeOrderSet(scale int, vw ValueWriter) testSet {
	refType := MakeRefType(NumberType)
	return newTestSetWithGen(int(setPattern)*scale, func(v Number) Value {
		return vw.WriteValue(v)
	}, refType)
}

func getTestRefToValueOrderSet(scale int, vw ValueWriter) testSet {
	setType := MakeSetType(NumberType)
	refType := MakeRefType(setType)
	return newTestSetWithGen(int(setPattern)*scale, func(v Number) Value {
		return vw.WriteValue(NewSet(v))
	}, refType)
}

func TestNewSet(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.True(MakeSetType(MakeUnionType()).Equals(s.Type()))
	assert.Equal(uint64(0), s.Len())

	s = NewSet(NewNumber(0))
	assert.True(MakeSetType(NumberType).Equals(s.Type()))

	s = NewSet()
	assert.IsType(MakeSetType(NumberType), s.Type())

	s2 := s.Remove(NewNumber(1))
	assert.IsType(s.Type(), s2.Type())
}

func TestSetLen(t *testing.T) {
	assert := assert.New(t)
	s0 := NewSet()
	assert.Equal(uint64(0), s0.Len())
	s1 := NewSet(Bool(true), NewNumber(1), NewString("hi"))
	assert.Equal(uint64(3), s1.Len())
	s2 := s1.Insert(Bool(false))
	assert.Equal(uint64(4), s2.Len())
	s3 := s2.Remove(Bool(true))
	assert.Equal(uint64(3), s3.Len())
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
	s1 := NewSet(Bool(true), NewNumber(42), NewNumber(42))
	assert.Equal(uint64(2), s1.Len())
}

func TestSetUniqueKeysString(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(NewString("hello"), NewString("world"), NewString("hello"))
	assert.Equal(uint64(2), s1.Len())
	assert.True(s1.Has(NewString("hello")))
	assert.True(s1.Has(NewString("world")))
	assert.False(s1.Has(NewString("foo")))
}

func TestSetUniqueKeysNewNumber(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(NewNumber(4), NewNumber(1), NewNumber(0), NewNumber(0), NewNumber(1), NewNumber(3))
	assert.Equal(uint64(4), s1.Len())
	assert.True(s1.Has(NewNumber(4)))
	assert.True(s1.Has(NewNumber(1)))
	assert.True(s1.Has(NewNumber(0)))
	assert.True(s1.Has(NewNumber(3)))
	assert.False(s1.Has(NewNumber(2)))
}

func TestSetHas(t *testing.T) {
	assert := assert.New(t)
	s1 := NewSet(Bool(true), NewNumber(1), NewString("hi"))
	assert.True(s1.Has(Bool(true)))
	assert.False(s1.Has(Bool(false)))
	assert.True(s1.Has(NewNumber(1)))
	assert.False(s1.Has(NewNumber(0)))
	assert.True(s1.Has(NewString("hi")))
	assert.False(s1.Has(NewString("ho")))

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
	assert := assert.New(t)

	vs := NewTestValueStore()
	doTest := func(ts testSet) {
		set := ts.toSet()
		set2 := vs.ReadValue(vs.WriteValue(set).TargetRef()).(Set)
		for _, v := range ts.values {
			assert.True(set.Has(v))
			assert.True(set2.Has(v))
		}
	}

	doTest(getTestNativeOrderSet(16))
	doTest(getTestRefValueOrderSet(2))
	doTest(getTestRefToNativeOrderSet(2, vs))
	doTest(getTestRefToValueOrderSet(2, vs))
}

func TestSetInsert(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	v1 := Bool(false)
	v2 := Bool(true)
	v3 := NewNumber(0)

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
	assert := assert.New(t)

	doTest := func(incr, offset int, ts testSet) {
		expected := ts.toSet()
		run := func(from, to int) {
			actual := ts.Remove(from, to).toSet().Insert(ts.values[from:to]...)
			assert.Equal(expected.Len(), actual.Len())
			assert.True(expected.Equals(actual))
		}
		for i := 0; i < len(ts.values)-offset; i += incr {
			run(i, i+offset)
		}
		run(len(ts.values)-offset, len(ts.values))
	}

	doTest(18, 3, getTestNativeOrderSet(9))
	doTest(64, 1, getTestNativeOrderSet(32))
	doTest(32, 1, getTestRefValueOrderSet(4))
	doTest(32, 1, getTestRefToNativeOrderSet(4, NewTestValueStore()))
	doTest(32, 1, getTestRefToValueOrderSet(4, NewTestValueStore()))
}

func TestSetInsertExistingValue(t *testing.T) {
	assert := assert.New(t)

	ts := getTestNativeOrderSet(2)
	original := ts.toSet()
	actual := original.Insert(ts.values[0])

	assert.Equal(original.Len(), actual.Len())
	assert.True(original.Equals(actual))
}

func TestSetRemove(t *testing.T) {
	assert := assert.New(t)
	v1 := Bool(false)
	v2 := Bool(true)
	v3 := NewNumber(0)
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

	assert := assert.New(t)

	doTest := func(incr, offset int, ts testSet) {
		whole := ts.toSet()
		run := func(from, to int) {
			expected := ts.Remove(from, to).toSet()
			actual := whole.Remove(ts.values[from:to]...)
			assert.Equal(expected.Len(), actual.Len())
			assert.True(expected.Equals(actual))
		}
		for i := 0; i < len(ts.values)-offset; i += incr {
			run(i, i+offset)
		}
		run(len(ts.values)-offset, len(ts.values))
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
	actual := original.Remove(NewNumber(-1)) // rand.Int63 returns non-negative values.

	assert.Equal(original.Len(), actual.Len())
	assert.True(original.Equals(actual))
}

func TestSetFirst(t *testing.T) {
	assert := assert.New(t)
	s := NewSet()
	assert.Nil(s.First())
	s = s.Insert(NewNumber(1))
	assert.NotNil(s.First())
	s = s.Insert(NewNumber(2))
	assert.NotNil(s.First())
	s2 := s.Remove(NewNumber(1))
	assert.NotNil(s2.First())
	s2 = s2.Remove(NewNumber(2))
	assert.Nil(s2.First())
}

func TestSetOfStruct(t *testing.T) {
	assert := assert.New(t)

	typ := MakeStructType("S1", TypeMap{
		"o": NumberType,
	})

	elems := []Value{}
	for i := 0; i < 200; i++ {
		elems = append(elems, newStructFromData(structData{"o": NewNumber(i)}, typ))
	}

	s := NewSet(elems...)
	for i := 0; i < 200; i++ {
		assert.True(s.Has(elems[i]))
	}
}

func TestSetIter(t *testing.T) {
	assert := assert.New(t)
	s := NewSet(NewNumber(0), NewNumber(1), NewNumber(2), NewNumber(3), NewNumber(4))
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

	doTest := func(ts testSet) {
		set := ts.toSet()
		sort.Sort(ts)
		idx := uint64(0)
		endAt := uint64(setPattern)

		set.Iter(func(v Value) (done bool) {
			assert.True(ts.values[idx].Equals(v))
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
	s := NewSet(NewNumber(0), NewNumber(1), NewNumber(2), NewNumber(3), NewNumber(4))
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

	doTest := func(ts testSet) {
		set := ts.toSet()
		sort.Sort(ts)
		idx := uint64(0)

		set.IterAll(func(v Value) {
			assert.True(ts.values[idx].Equals(v))
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
		assert.Equal(expectOrdering[i].Ref().String(), value.Ref().String())
		i++
	})
}

func TestSetOrdering(t *testing.T) {
	assert := assert.New(t)

	testSetOrder(assert,
		StringType,
		[]Value{
			NewString("a"),
			NewString("z"),
			NewString("b"),
			NewString("y"),
			NewString("c"),
			NewString("x"),
		},
		[]Value{
			NewString("a"),
			NewString("b"),
			NewString("c"),
			NewString("x"),
			NewString("y"),
			NewString("z"),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			NewNumber(0),
			NewNumber(1000),
			NewNumber(1),
			NewNumber(100),
			NewNumber(2),
			NewNumber(10),
		},
		[]Value{
			NewNumber(0),
			NewNumber(1),
			NewNumber(2),
			NewNumber(10),
			NewNumber(100),
			NewNumber(1000),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			NewNumber(0),
			NewNumber(-30),
			NewNumber(25),
			NewNumber(1002),
			NewNumber(-5050),
			NewNumber(23),
		},
		[]Value{
			NewNumber(-5050),
			NewNumber(-30),
			NewNumber(0),
			NewNumber(23),
			NewNumber(25),
			NewNumber(1002),
		},
	)

	testSetOrder(assert,
		NumberType,
		[]Value{
			NewNumber(0.0001),
			NewNumber(0.000001),
			NewNumber(1),
			NewNumber(25.01e3),
			NewNumber(-32.231123e5),
			NewNumber(23),
		},
		[]Value{
			NewNumber(-32.231123e5),
			NewNumber(0.000001),
			NewNumber(0.0001),
			NewNumber(1),
			NewNumber(23),
			NewNumber(25.01e3),
		},
	)

	testSetOrder(assert,
		ValueType,
		[]Value{
			NewString("a"),
			NewString("z"),
			NewString("b"),
			NewString("y"),
			NewString("c"),
			NewString("x"),
		},
		// Ordered by value
		[]Value{
			NewString("a"),
			NewString("b"),
			NewString("c"),
			NewString("x"),
			NewString("y"),
			NewString("z"),
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

	s = NewSet(NewNumber(0))
	assert.True(s.Type().Equals(MakeSetType(NumberType)))

	s2 := s.Remove(NewNumber(1))
	assert.True(s2.Type().Equals(MakeSetType(NumberType)))

	s2 = s.Insert(NewNumber(0), NewNumber(1))
	assert.True(s.Type().Equals(s2.Type()))

	s3 := s.Insert(Bool(true))
	assert.True(s3.Type().Equals(MakeSetType(MakeUnionType(BoolType, NumberType))))
	s4 := s.Insert(NewNumber(3), Bool(true))
	assert.True(s4.Type().Equals(MakeSetType(MakeUnionType(BoolType, NumberType))))
}

func TestSetChunks(t *testing.T) {
	assert := assert.New(t)

	l1 := NewSet(NewNumber(0))
	c1 := l1.Chunks()
	assert.Len(c1, 0)

	l2 := NewSet(NewRef(NewNumber(0)))
	c2 := l2.Chunks()
	assert.Len(c2, 1)
}

func TestSetChunks2(t *testing.T) {
	assert := assert.New(t)

	vs := NewTestValueStore()
	doTest := func(ts testSet) {
		set := ts.toSet()
		set2chunks := vs.ReadValue(vs.WriteValue(set).TargetRef()).Chunks()
		for i, r := range set.Chunks() {
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
	assert.Equal("sha1-8186877fb71711b8e6a516ed5c8ad1ccac8c6c00", s.Ref().String())
	assert.Equal(deriveCollectionHeight(s), getRefHeightOfCollection(s))
}

func TestSetRefOfStructFirstNNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	nums := generateNumbersAsRefOfStructs(testSetSize)
	s := NewSet(nums...)
	assert.Equal("sha1-14eeb2d1835011bf3e018121ba3274bc08e634e5", s.Ref().String())
	// height + 1 because the leaves are Ref values (with height 1).
	assert.Equal(deriveCollectionHeight(s)+1, getRefHeightOfCollection(s))
}

func TestSetModifyAfterRead(t *testing.T) {
	assert := assert.New(t)
	vs := NewTestValueStore()
	set := getTestNativeOrderSet(2).toSet()
	// Drop chunk values.
	set = vs.ReadValue(vs.WriteValue(set).TargetRef()).(Set)
	// Modify/query. Once upon a time this would crash.
	fst := set.First()
	set = set.Remove(fst)
	assert.False(set.Has(fst))
	assert.True(set.Has(set.First()))
	set = set.Insert(fst)
	assert.True(set.Has(fst))
}
