package spec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	_, err := ForDatabase("http://localhost/foo/bar/baz")
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol http in ")

	spec, err := ForDatabase("/var/data")
	fmt.Println("Spec:", spec)
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol nbs in /var/data")

	_, err = ForDatabase("aws:table/bucket/db")
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol aws in")

	spec, err = ForDataset("mem::test")
	assert.NoError(err)
	defer spec.Close()
	assert.Equal("mem", spec.Protocol)
	assert.Equal("", spec.DatabaseName)
	assert.Equal("test", spec.Path.Dataset)
	assert.True(spec.Path.Path.IsEmpty())
}
