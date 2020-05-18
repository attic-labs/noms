package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/attic-labs/noms/go/spec/lite"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("aws://table/bucket")
	assert.Error(err)
	assert.Contains(err.Error(), "aws spec must match pattern aws:")

	sp, err = spec.ForDatabase("aws:table/bucket/db")
	assert.NoError(err)
	assert.Equal("aws", sp.Protocol)
	assert.Equal("aws:table/bucket/db", sp.Href())

	sp, err = spec.ForDatabase("https://localhost/foo/bar/baz")
	assert.Error(err)
	assert.Contains(err.Error(), "Invalid database protocol https")
}
