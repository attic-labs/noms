// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/util/exit"
	"github.com/attic-labs/noms/go/util/verbose"
	flag "github.com/juju/gnuflag"
	"gopkg.in/alecthomas/kingpin.v2"
)

var commands = []*util.Command{
	nomsCommit,
	nomsConfig,
	nomsDiff,
	nomsDs,
	nomsLog,
	nomsMerge,
	nomsRoot,
	nomsServe,
	nomsShow,
	nomsSync,
	nomsVersion,
}

type kCommandHandler func(input string) (exitCode int)
type kCommand func(*kingpin.Application) (*kingpin.CmdClause, kCommandHandler)

var kCommands = []kCommand{
	nomsBlob,
}

var actions = []string{
	"interacting with",
	"poking at",
	"goofing with",
	"dancing with",
	"playing with",
	"contemplation of",
	"showing off",
	"jiggerypokery of",
	"singing to",
	"nomming on",
}

func usageString() string {
	i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(actions))
	return fmt.Sprintf(`Noms is a tool for %s Noms data.`, actions[i])
}

func main() {
	// allow short (-h) help
	kingpin.CommandLine.HelpFlag.Short('h')
	noms := kingpin.New("noms", usageString())

	// set up docs for non-kingpin commands
	addNomsDocs(noms)

	kHandlers := map[string]kCommandHandler{}

	// install kingpin handlers
	for _, cmdFunction := range kCommands {
		command, handler := cmdFunction(noms)
		kHandlers[command.FullCommand()] = handler
	}

	input := kingpin.MustParse(noms.Parse(os.Args[1:]))
	if handler := kHandlers[strings.Split(input, " ")[0]]; handler != nil {
		handler(input)
	}

	// fall back to previous (non-kingpin) noms commands

	flag.Parse(false)

	args := flag.Args()

	// Don't prefix log messages with timestamp when running interactively
	log.SetFlags(0)

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			flags := cmd.Flags()
			flags.Usage = cmd.Usage

			flags.Parse(true, args[1:])
			args = flags.Args()
			if cmd.Nargs != 0 && len(args) < cmd.Nargs {
				cmd.Usage()
			}
			exitCode := cmd.Run(args)
			if exitCode != 0 {
				exit.Exit(exitCode)
			}
			return
		}
	}
}

var (
	verboseFlag *bool
	quietFlag   *bool
)

// addVerboseFlags adds --verbose and --quiet flags to the passed command
func addVerboseFlags(cmd *kingpin.CmdClause) (verboseFlag *bool, quietFlag *bool) {
	verboseFlag = cmd.Flag("verbose", "show more").Short('v').Bool()
	quietFlag = cmd.Flag("quiet", "show less").Short('q').Bool()
	return
}

// applyVerbosity - run when commands are invoked to apply the verbosity arguments configured in addVerboseFlags
func applyVerbosity() {
	if verboseFlag != nil {
		verbose.SetVerbose(*verboseFlag)
	}
	if quietFlag != nil {
		verbose.SetQuiet(*quietFlag)
	}
}

// addDatabaseArg adds a "database" arg to the passed command
func addDatabaseArg(cmd *kingpin.CmdClause) (arg *string) {
	return cmd.Arg("database", "a noms database path").Required().String() // TODO: custom parser for noms db URL?
}

// addNomsDocs - adds documentation (docs only, not commands) for existing (pre-kingpin) commands.
func addNomsDocs(noms *kingpin.Application) {
	// commmit
	commit := noms.Command("commit", `Commits a specified value as head of the dataset
If absolute-path is not provided, then it is read from stdin. See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the dataset and absolute-path arguments.
`)
	commit.Flag("allow-dupe", "creates a new commit, even if it would be identical (modulo metadata and parents) to the existing HEAD.").Default("0").Int()
	commit.Flag("date", "alias for -meta 'date=<date>'. '<date>' must be iso8601-formatted. If '<date>' is empty, it defaults to the current date.").String()
	commit.Flag("message", "alias for -meta 'message=<message>'").String()
	commit.Flag("meta", "'<key>=<value>' - creates a metadata field called 'key' set to 'value'. Value should be human-readable encoded.").String()
	commit.Flag("meta-p", "'<key>=<path>' - creates a metadata field called 'key' set to the value at <path>").String()
	addVerboseFlags(commit)
	commit.Arg("absolute-path", "the path to read data from").String()
	// TODO: this should be required, but kingpin does not allow required args after non-required ones. Perhaps a custom type would fix that?
	commit.Arg("database", "a noms database path").String()

	// config
	noms.Command("config", "Prints the active configuration if a .nomsconfig file is present")

	// diff
	diff := noms.Command("diff", `Shows the difference between two objects
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object arguments.
`)
	diff.Flag("summarize", "Writes a summary of the changes instead").Short('s').Bool()
	addVerboseFlags(diff)
	diff.Arg("object1", "").Required().String()
	diff.Arg("object2", "").Required().String()

	// ds
	ds := noms.Command("ds", `Noms dataset management
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
`)
	ds.Flag("delete", "dataset to delete").Short('d').String()
	addVerboseFlags(ds)
	ds.Arg("database", "a noms database path").String()

	// log
	log := noms.Command("log", `Displays the history of a path
See Spelling Values at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the <path-spec> parameter.
`)
	log.Flag("color", "value of 1 forces color on, 0 forces color off").Default("-1").Int()
	log.Flag("max-lines", "max number of lines to show per commit (-1 for all lines)").Default("9").Int()
	log.Flag("max-commits", "max number of commits to display (0 for all commits)").Short('n').Default("0").Int()
	log.Flag("oneline", "show a summary of each commit on a single line").Bool()
	log.Flag("graph", "show ascii-based commit hierarchy on left side of output").Bool()
	log.Flag("show-value", "show commit value rather than diff information").Bool()
	addVerboseFlags(log)
	log.Arg("path-spec", "").Required().String()

	// merge
	merge := noms.Command("merge", `Merges and commits the head values of two named datasets
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
You must provide a working database and the names of two Datasets you want to merge. The values at the heads of these Datasets will be merged, put into a new Commit object, and set as the Head of the third provided Dataset name.
`)
	merge.Flag("policy", "conflict resolution policy for merging. Defaults to 'n', which means no resolution strategy will be applied. Supported values are 'l' (left), 'r' (right) and 'p' (prompt). 'prompt' will bring up a simple command-line prompt allowing you to resolve conflicts by choosing between 'l' or 'r' on a case-by-case basis.").Default("n").Enum("n", "r", "l", "p")
	addVerboseFlags(merge)
	addDatabaseArg(merge)
	merge.Arg("left-dataset-name", "a dataset").Required().String()
	merge.Arg("right-dataset-name", "a dataset").Required().String()
	merge.Arg("output-dataset-name", "a dataset").Required().String()

	// root
	root := noms.Command("root", `Get or set the current root hash of the entire database
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
`)
	root.Flag("update", "Replaces the entire database with the one with the given hash").String()
	addVerboseFlags(root)
	addDatabaseArg(root)

	// serve
	serve := noms.Command("serve", `Serves a Noms database over HTTP
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
`)
	serve.Flag("port", "port to listen on for HTTP requests").Default("8000").Int()
	addDatabaseArg(serve)

	// show
	show := noms.Command("show", `Shows a serialization of a Noms object
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object argument.
`)
	show.Flag("raw", "If true, dumps the raw binary version of the data").Bool()
	show.Flag("stats", "If true, reports statistics related to the value").Bool()
	addVerboseFlags(show)
	show.Arg("object", "a noms object").Required().String()

	sync := noms.Command("sync", `Moves datasets between or within databases
See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the object and dataset arguments.
`)
	sync.Flag("parallelism", "").Short('p').Default("512").Int()
	addVerboseFlags(sync)
	sync.Arg("source-object", "a noms source object").Required().String()
	sync.Arg("dest-dataset", "a noms dataset").Required().String()

	// version
	noms.Command("version", "Print the noms version")
}
