package verbose_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/attic-labs/noms/go/util/verbose/verboseflags"
)

func TestVerbose(t *testing.T) {
	app := kingpin.New("app", "")
	for _, tt := range []struct {
		args    string
		verbose bool
		quiet   bool
	}{
		{args: "-v", verbose: true},
		{args: "--verbose", verbose: true},
		{args: "-q", quiet: true},
		{args: "--quiet", quiet: true},
		{args: "-vq", verbose: true, quiet: true},
		{args: "--verbose --quiet", verbose: true, quiet: true},
	} {
		t.Run(tt.args, func(t *testing.T) {
			assert := assert.New(t)

			verboseflags.Register(app)
			assert.False(verbose.Verbose())
			assert.False(verbose.Quiet())
			res, err := app.Parse(strings.Split(tt.args, " "))
			assert.Empty(res)
			assert.NoError(err)
			assert.Equal(tt.verbose, verbose.Verbose())
			assert.Equal(tt.quiet, verbose.Quiet())
		})
	}
}
