package main

import (
	"flag"
	"os"

	. "github.com/attic-labs/noms/dbg"
	"github.com/attic-labs/noms/nomgen"
	"github.com/attic-labs/noms/types"
)

func main() {
	outFile := flag.String("o", "", "output file")
	flag.Parse()
	if *outFile == "" {
		flag.Usage()
		return
	}

	user := types.NewMap(
		types.NewString("$type"), types.NewString("noms.StructDef"),
		types.NewString("$name"), types.NewString("User"),
		types.NewString("id"), types.NewString("string"),
		types.NewString("name"), types.NewString("string"),
		types.NewString("oAuthToken"), types.NewString("string"),
		types.NewString("oAuthSecret"), types.NewString("string"))

	f, err := os.OpenFile(*outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	Chk.NoError(err)
	ng := nomgen.New(f)
	ng.WriteGo(user, "main")
}
