package verboseflags

import (
	"github.com/attic-labs/kingpin"

	"github.com/attic-labs/noms/go/util/verbose"
)

// Register registers -v|--verbose flags for general usage
func Register(app *kingpin.Application) {
	// Must reset globals because under test this can get called multiple times.
	verbose.SetVerbose(false)
	verbose.SetQuiet(false)
	loud := false
	quiet := false
	app.Flag("verbose", "show more").Short('v').Action(func(ctx *kingpin.ParseContext) error {
		verbose.SetVerbose(loud)
		return nil
	}).BoolVar(&loud)
	app.Flag("quiet", "show less").Short('q').Action(func(ctx *kingpin.ParseContext) error {
		verbose.SetQuiet(quiet)
		return nil
	}).BoolVar(&quiet)
}
