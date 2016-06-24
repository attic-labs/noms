package sizecache

import (
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/testify/assert"
)

var (
	valueCounter = 0
)

func hashFromString(s string) hash.Hash {
	return hash.FromData([]byte(s))
}

func TestSizeCache(t *testing.T) {
	assert := assert.New(t)
	defSize := uint64(200)

	c := New(1024).(*realSizeCache)
	for i, v := range []string{"data-1", "data-2", "data-3", "data-4", "data-5", "data-6", "data-7", "data-8", "data-9"} {
		c.Add(hashFromString(v), defSize, v)
		maxElements := uint64(i + 1)
		if maxElements >= uint64(5) {
			maxElements = uint64(5)
		}
		assert.Equal(maxElements*defSize, c.totalSize)
	}

	_, ok := c.Get(hashFromString("data-1"))
	assert.False(ok)
	assert.Equal(hashFromString("data-5").String(), c.lru.Front().Value.(string))

	v, ok := c.Get(hashFromString("data-5"))
	assert.True(ok)
	assert.Equal("data-5", v.(string))
	assert.Equal(hashFromString("data-5").String(), c.lru.Back().Value.(string))
	assert.Equal(hashFromString("data-6").String(), c.lru.Front().Value.(string))

	c.Add(hashFromString("data-7"), defSize, "data-7")
	assert.Equal(hashFromString("data-7").String(), c.lru.Back().Value.(string))
	assert.Equal(uint64(1000), c.totalSize)

	c.Add(hashFromString("no-data"), 0, nil)
	v, ok = c.Get(hashFromString("no-data"))
	assert.True(ok)
	assert.Nil(v)
	assert.Equal(hashFromString("no-data").String(), c.lru.Back().Value.(string))
	assert.Equal(uint64(1000), c.totalSize)
	assert.Equal(6, c.lru.Len())
	assert.Equal(6, len(c.cache))

	for _, v := range []string{"data-5", "data-6", "data-7", "data-8", "data-9"} {
		c.Get(hashFromString(v))
		assert.Equal(hashFromString(v).String(), c.lru.Back().Value.(string))
	}
	assert.Equal(hashFromString("no-data").String(), c.lru.Front().Value.(string))

	c.Add(hashFromString("data-10"), 200, "data-10")
	assert.Equal(uint64(1000), c.totalSize)
	assert.Equal(5, c.lru.Len())
	assert.Equal(5, len(c.cache))

	_, ok = c.Get(hashFromString("no-data"))
	assert.False(ok)
	_, ok = c.Get(hashFromString("data-5"))
	assert.False(ok)
}

func TestNoopSizeCache(t *testing.T) {
	assert := assert.New(t)

	c := New(0)
	c.Add(hashFromString("data1"), 200, "data1")
	_, ok := c.Get(hashFromString("data1"))
	assert.False(ok)
}
