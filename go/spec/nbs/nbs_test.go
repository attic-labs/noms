package nbs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/attic-labs/noms/go/spec/lite"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("/var/data")
	assert.NoError(err)
	assert.Equal("nbs", sp.Protocol)
	assert.Equal("/var/data", sp.DatabaseName)

	sp, err = spec.ForDatabase("nbs:/var/data")
	assert.NoError(err)
	assert.Equal("nbs", sp.Protocol)
	assert.Equal("/var/data", sp.DatabaseName)

	sp, err = spec.ForDatabase("http://localhost/foo/bar/baz")
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol http")
}
