package http

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/attic-labs/noms/go/spec/lite"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("http://localhost/foo/bar/baz")
	assert.NoError(err)
	assert.Equal("http", sp.Protocol)
	assert.Equal("http://localhost/foo/bar/baz", sp.Href())

	sp, err = spec.ForDatabase("https://localhost/foo/bar/baz")
	assert.NoError(err)
	assert.Equal("https", sp.Protocol)
	assert.Equal("https://localhost/foo/bar/baz", sp.Href())

	sp, err = spec.ForDatabase("/var/data")
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol nbs in /var/data")
}
