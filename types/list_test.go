package types

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const testListSize = 5000

type testList []Value

func (tl testList) Set(idx int, v Value) (res testList) {
	res = append(res, tl[:idx]...)
	res = append(res, v)
	res = append(res, tl[idx+1:]...)
	return
}

func (tl testList) Insert(idx int, vs ...Value) (res testList) {
	res = append(res, tl[:idx]...)
	res = append(res, vs...)
	res = append(res, tl[idx:]...)
	return
}

func (tl testList) Remove(start, end int) (res testList) {
	res = append(res, tl[:start]...)
	res = append(res, tl[end:]...)
	return
}

func (tl testList) RemoveAt(idx int) testList {
	return tl.Remove(idx, idx+1)
}

func (tl testList) toList() List {
	return NewList(tl...)
}

func newTestList(length int) testList {
	tl := testList{}
	for i := 0; i < length; i++ {
		tl = append(tl, NewNumber(i))
	}
	return tl
}

func newTestListFromList(list List) testList {
	tl := testList{}
	list.IterAll(func(v Value, idx uint64) {
		tl = append(tl, v)
	})
	return tl
}

type listTestSuite struct {
	collectionTestSuite
	elems testList
}

func newListTestSuite(size uint, expectRefStr string, expectChunkCount int, expectPrependChunkDiff int, expectAppendChunkDiff int) *listTestSuite {
	length := 1 << size
	elems := newTestList(length)
	tr := MakeListType(NumberType)
	list := NewList(elems...)
	return &listTestSuite{
		collectionTestSuite: collectionTestSuite{
			col:                    list,
			expectType:             tr,
			expectLen:              uint64(length),
			expectRef:              expectRefStr,
			expectChunkCount:       expectChunkCount,
			expectPrependChunkDiff: expectPrependChunkDiff,
			expectAppendChunkDiff:  expectAppendChunkDiff,
			validate: func(v2 Collection) bool {
				l2 := v2.(List)
				out := []Value{}
				l2.IterAll(func(v Value, index uint64) {
					out = append(out, v)
				})
				return valueSlicesEqual(elems, out)
			},
			prependOne: func() Collection {
				dup := make([]Value, length+1)
				dup[0] = NewNumber(0)
				copy(dup[1:], elems)
				return NewList(dup...)
			},
			appendOne: func() Collection {
				dup := make([]Value, length+1)
				copy(dup, elems)
				dup[len(dup)-1] = NewNumber(0)
				return NewList(dup...)
			},
		},
		elems: elems,
	}
}

func (suite *listTestSuite) TestGet() {
	list := suite.col.(List)
	for i := 0; i < len(suite.elems); i++ {
		suite.True(suite.elems[i].Equals(list.Get(uint64(i))))
	}
	suite.Equal(suite.expectLen, list.Len())
}

func (suite *listTestSuite) TestIter() {
	list := suite.col.(List)
	expectIdx := uint64(0)
	endAt := suite.expectLen / 2
	list.Iter(func(v Value, idx uint64) bool {
		suite.Equal(expectIdx, idx)
		expectIdx++
		suite.Equal(suite.elems[idx], v)
		return expectIdx == endAt
	})

	suite.Equal(endAt, expectIdx)
}

func (suite *listTestSuite) TestMap() {
	list := suite.col.(List)
	l := list.Map(func(v Value, i uint64) interface{} {
		v1 := v.(Number).ToUint64()
		return NewNumber(v1 + i)
	})

	suite.Equal(uint64(len(l)), suite.expectLen)
	for i := 0; i < len(l); i++ {
		suite.True(l[i].(Number).Equals(NewNumber(list.Get(uint64(i)).(Number).ToUint64() + uint64(i))))
	}
}

func TestListSuite1K(t *testing.T) {
	suite.Run(t, newListTestSuite(10, "sha1-26169e4d8d3175994c992ca21be07c30ad2007e3", 17, 19, 2))
}

func TestListSuite4K(t *testing.T) {
	suite.Run(t, newListTestSuite(12, "sha1-35f79a6d2ddbe34ad469b1bf2a9a1b460e0e997c", 2, 3, 2))
}

func TestListInsert(t *testing.T) {
	assert := assert.New(t)

	tl := newTestList(512)
	list := tl.toList()

	for i := 0; i < len(tl); i += 16 {
		tl = tl.Insert(i, NewNumber(i))
		list = list.Insert(uint64(i), NewNumber(i))
	}

	assert.True(tl.toList().Equals(list))
}

func TestListRemove(t *testing.T) {
	assert := assert.New(t)

	tl := newTestList(512)
	list := tl.toList()

	for i := len(tl) - 16; i >= 0; i -= 16 {
		tl = tl.Remove(i, i+4)
		list = list.Remove(uint64(i), uint64(i+4))
	}

	assert.True(tl.toList().Equals(list))
}

func TestListRemoveAt(t *testing.T) {
	assert := assert.New(t)

	l0 := NewList()
	l0 = l0.Append(Bool(false), Bool(true))
	l1 := l0.RemoveAt(1)
	assert.True(NewList(Bool(false)).Equals(l1))
	l1 = l1.RemoveAt(0)
	assert.True(NewList().Equals(l1))

	assert.Panics(func() {
		l1.RemoveAt(0)
	})
}

func getTestListLen() uint64 {
	return uint64(listPattern) * 50
}

func getTestList() testList {
	length := int(getTestListLen())
	s := rand.NewSource(42)
	values := make([]Value, length)
	for i := 0; i < length; i++ {
		values[i] = NewNumber(s.Int63() & 0xff)
	}

	return values
}

func getTestListUnique() testList {
	length := int(getTestListLen())
	s := rand.NewSource(42)
	uniques := map[int64]bool{}
	for len(uniques) < length {
		uniques[s.Int63()] = true
	}
	values := make([]Value, 0, length)
	for k := range uniques {
		values = append(values, NewNumber(k))
	}
	return values
}

func testListFromNomsList(list List) testList {
	simple := make(testList, list.Len())
	list.IterAll(func(v Value, offset uint64) {
		simple[offset] = v
	})
	return simple
}

func TestStreamingListCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	vs := NewTestValueStore()
	simpleList := getTestList()

	cl := NewList(simpleList...)
	valueChan := make(chan Value)
	listChan := NewStreamingList(vs, valueChan)
	for _, v := range simpleList {
		valueChan <- v
	}
	close(valueChan)
	sl := <-listChan
	assert.True(cl.Equals(sl))
	cl.Iter(func(v Value, idx uint64) (done bool) {
		done = !assert.True(v.Equals(sl.Get(idx)))
		return
	})
}

func TestListAppend(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	newList := func(items testList) List {
		return NewList(items...)
	}

	listToSimple := func(cl List) (simple testList) {
		cl.IterAll(func(v Value, offset uint64) {
			simple = append(simple, v)
		})
		return
	}

	cl := newList(getTestList())
	cl2 := cl.Append(NewNumber(42))
	cl3 := cl2.Append(NewNumber(43))
	cl4 := cl3.Append(getTestList()...)
	cl5 := cl4.Append(NewNumber(44), NewNumber(45))
	cl6 := cl5.Append(getTestList()...)

	expected := getTestList()
	assert.Equal(expected, listToSimple(cl))
	assert.Equal(getTestListLen(), cl.Len())
	assert.True(newList(expected).Equals(cl))

	expected = append(expected, NewNumber(42))
	assert.Equal(expected, listToSimple(cl2))
	assert.Equal(getTestListLen()+1, cl2.Len())
	assert.True(newList(expected).Equals(cl2))

	expected = append(expected, NewNumber(43))
	assert.Equal(expected, listToSimple(cl3))
	assert.Equal(getTestListLen()+2, cl3.Len())
	assert.True(newList(expected).Equals(cl3))

	expected = append(expected, getTestList()...)
	assert.Equal(expected, listToSimple(cl4))
	assert.Equal(2*getTestListLen()+2, cl4.Len())
	assert.True(newList(expected).Equals(cl4))

	expected = append(expected, NewNumber(44), NewNumber(45))
	assert.Equal(expected, listToSimple(cl5))
	assert.Equal(2*getTestListLen()+4, cl5.Len())
	assert.True(newList(expected).Equals(cl5))

	expected = append(expected, getTestList()...)
	assert.Equal(expected, listToSimple(cl6))
	assert.Equal(3*getTestListLen()+4, cl6.Len())
	assert.True(newList(expected).Equals(cl6))
}

func TestListInsertNothing(t *testing.T) {
	assert := assert.New(t)

	cl := getTestList().toList()

	assert.True(cl.Equals(cl.Insert(0)))
	for i := uint64(1); i < getTestListLen(); i *= 2 {
		assert.True(cl.Equals(cl.Insert(i)))
	}
	assert.True(cl.Equals(cl.Insert(cl.Len() - 1)))
	assert.True(cl.Equals(cl.Insert(cl.Len())))
}

func TestListInsertStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	cl := getTestList().toList()
	cl2 := cl.Insert(0, NewNumber(42))
	cl3 := cl2.Insert(0, NewNumber(43))
	cl4 := cl3.Insert(0, getTestList()...)
	cl5 := cl4.Insert(0, NewNumber(44), NewNumber(45))
	cl6 := cl5.Insert(0, getTestList()...)

	expected := getTestList()
	assert.Equal(expected, testListFromNomsList(cl))
	assert.Equal(getTestListLen(), cl.Len())
	assert.True(expected.toList().Equals(cl))

	expected = expected.Insert(0, NewNumber(42))
	assert.Equal(expected, testListFromNomsList(cl2))
	assert.Equal(getTestListLen()+1, cl2.Len())
	assert.True(expected.toList().Equals(cl2))

	expected = expected.Insert(0, NewNumber(43))
	assert.Equal(expected, testListFromNomsList(cl3))
	assert.Equal(getTestListLen()+2, cl3.Len())
	assert.True(expected.toList().Equals(cl3))

	expected = expected.Insert(0, getTestList()...)
	assert.Equal(expected, testListFromNomsList(cl4))
	assert.Equal(2*getTestListLen()+2, cl4.Len())
	assert.True(expected.toList().Equals(cl4))

	expected = expected.Insert(0, NewNumber(44), NewNumber(45))
	assert.Equal(expected, testListFromNomsList(cl5))
	assert.Equal(2*getTestListLen()+4, cl5.Len())
	assert.True(expected.toList().Equals(cl5))

	expected = expected.Insert(0, getTestList()...)
	assert.Equal(expected, testListFromNomsList(cl6))
	assert.Equal(3*getTestListLen()+4, cl6.Len())
	assert.True(expected.toList().Equals(cl6))
}

func TestListInsertMiddle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	cl := getTestList().toList()
	cl2 := cl.Insert(100, NewNumber(42))
	cl3 := cl2.Insert(200, NewNumber(43))
	cl4 := cl3.Insert(300, getTestList()...)
	cl5 := cl4.Insert(400, NewNumber(44), NewNumber(45))
	cl6 := cl5.Insert(500, getTestList()...)
	cl7 := cl6.Insert(600, NewNumber(100))

	expected := getTestList()
	assert.Equal(expected, testListFromNomsList(cl))
	assert.Equal(getTestListLen(), cl.Len())
	assert.True(expected.toList().Equals(cl))

	expected = expected.Insert(100, NewNumber(42))
	assert.Equal(expected, testListFromNomsList(cl2))
	assert.Equal(getTestListLen()+1, cl2.Len())
	assert.True(expected.toList().Equals(cl2))

	expected = expected.Insert(200, NewNumber(43))
	assert.Equal(expected, testListFromNomsList(cl3))
	assert.Equal(getTestListLen()+2, cl3.Len())
	assert.True(expected.toList().Equals(cl3))

	expected = expected.Insert(300, getTestList()...)
	assert.Equal(expected, testListFromNomsList(cl4))
	assert.Equal(2*getTestListLen()+2, cl4.Len())
	assert.True(expected.toList().Equals(cl4))

	expected = expected.Insert(400, NewNumber(44), NewNumber(45))
	assert.Equal(expected, testListFromNomsList(cl5))
	assert.Equal(2*getTestListLen()+4, cl5.Len())
	assert.True(expected.toList().Equals(cl5))

	expected = expected.Insert(500, getTestList()...)
	assert.Equal(expected, testListFromNomsList(cl6))
	assert.Equal(3*getTestListLen()+4, cl6.Len())
	assert.True(expected.toList().Equals(cl6))

	expected = expected.Insert(600, NewNumber(100))
	assert.Equal(expected, testListFromNomsList(cl7))
	assert.Equal(3*getTestListLen()+5, cl7.Len())
	assert.True(expected.toList().Equals(cl7))
}

func TestListInsertRanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	testList := getTestList()
	whole := testList.toList()

	// Compare list equality. Increment by 256 (16^2) because each iteration requires building a new list, which is slow.
	for incr, i := 256, 0; i < len(testList)-incr; i += incr {
		for window := 1; window <= incr; window *= 16 {
			testListPart := testList.Remove(i, i+window)
			actual := testListPart.toList().Insert(uint64(i), testList[i:i+window]...)
			assert.Equal(whole.Len(), actual.Len())
			assert.True(whole.Equals(actual))
		}
	}

	// Compare list length, which doesn't require building a new list every iteration, so the increment can be smaller.
	for incr, i := 10, 0; i < len(testList); i += incr {
		assert.Equal(len(testList)+incr, int(whole.Insert(uint64(i), testList[0:incr]...).Len()))
	}
}

func TestListRemoveNothing(t *testing.T) {
	assert := assert.New(t)

	cl := getTestList().toList()

	assert.True(cl.Equals(cl.Remove(0, 0)))
	for i := uint64(1); i < getTestListLen(); i *= 2 {
		assert.True(cl.Equals(cl.Remove(i, i)))
	}
	assert.True(cl.Equals(cl.Remove(cl.Len()-1, cl.Len()-1)))
	assert.True(cl.Equals(cl.Remove(cl.Len(), cl.Len())))
}

func TestListRemoveEverything(t *testing.T) {
	assert := assert.New(t)

	cl := getTestList().toList().Remove(0, getTestListLen())

	assert.True(NewList().Equals(cl))
	assert.Equal(0, int(cl.Len()))
}

func TestListRemoveAtMiddle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	cl := getTestList().toList()
	cl2 := cl.RemoveAt(100)
	cl3 := cl2.RemoveAt(200)

	expected := getTestList()
	assert.Equal(expected, testListFromNomsList(cl))
	assert.Equal(getTestListLen(), cl.Len())
	assert.True(expected.toList().Equals(cl))

	expected = expected.RemoveAt(100)
	assert.Equal(expected, testListFromNomsList(cl2))
	assert.Equal(getTestListLen()-1, cl2.Len())
	assert.True(expected.toList().Equals(cl2))

	expected = expected.RemoveAt(200)
	assert.Equal(expected, testListFromNomsList(cl3))
	assert.Equal(getTestListLen()-2, cl3.Len())
	assert.True(expected.toList().Equals(cl3))
}

func TestListRemoveRanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	testList := getTestList()
	whole := testList.toList()

	// Compare list equality. Increment by 256 (16^2) because each iteration requires building a new list, which is slow.
	for incr, i := 256, 0; i < len(testList)-incr; i += incr {
		for window := 1; window <= incr; window *= 16 {
			testListPart := testList.Remove(i, i+window)
			expected := testListPart.toList()
			actual := whole.Remove(uint64(i), uint64(i+window))
			assert.Equal(expected.Len(), actual.Len())
			assert.True(expected.Equals(actual))
		}
	}

	// Compare list length, which doesn't require building a new list every iteration, so the increment can be smaller.
	for incr, i := 10, 0; i < len(testList)-incr; i += incr {
		assert.Equal(len(testList)-incr, int(whole.Remove(uint64(i), uint64(i+incr)).Len()))
	}
}

func TestListSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)

	testList := getTestList()
	cl := testList.toList()

	testIdx := func(idx int, testEquality bool) {
		newVal := NewNumber(-1) // Test values are never < 0
		cl2 := cl.Set(uint64(idx), newVal)
		assert.False(cl.Equals(cl2))
		if testEquality {
			assert.True(testList.Set(idx, newVal).toList().Equals(cl2))
		}
	}

	// Compare list equality. Increment by 100 because each iteration requires building a new list, which is slow, but always test the last index.
	for incr, i := 100, 0; i < len(testList); i += incr {
		testIdx(i, true)
	}
	testIdx(len(testList)-1, true)

	// Compare list unequality, which doesn't require building a new list every iteration, so the increment can be smaller.
	for incr, i := 10, 0; i < len(testList); i += incr {
		testIdx(i, false)
	}
}

func TestListFirstNNumbers(t *testing.T) {
	assert := assert.New(t)

	firstNNumbers := func(n int) []Value {
		nums := []Value{}
		for i := 0; i < n; i++ {
			nums = append(nums, NewNumber(i))
		}

		return nums
	}

	nums := firstNNumbers(testListSize)
	s := NewList(nums...)
	assert.Equal("sha1-aa1605484d993e89dbc0431acb9f2478282f9d94", s.Ref().String())
}

func TestListRefOfStructFirstNNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	assert := assert.New(t)
	vs := NewTestValueStore()

	firstNNumbers := func(n int) []Value {
		nums := []Value{}
		for i := 0; i < n; i++ {
			r := vs.WriteValue(NewStruct("num", structData{"n": NewNumber(i)}))
			nums = append(nums, r)
		}

		return nums
	}

	nums := firstNNumbers(testListSize)
	s := NewList(nums...)
	assert.Equal("sha1-2e79d54322aa793d0e8d48380a28927a257a141a", s.Ref().String())
}

func TestListModifyAfterRead(t *testing.T) {
	assert := assert.New(t)
	vs := NewTestValueStore()

	list := getTestList().toList()
	// Drop chunk values.
	list = vs.ReadValue(vs.WriteValue(list).TargetRef()).(List)
	// Modify/query. Once upon a time this would crash.
	llen := list.Len()
	z := list.Get(0)
	list = list.RemoveAt(0)
	assert.Equal(llen-1, list.Len())
	list = list.Append(z)
	assert.Equal(llen, list.Len())
}
