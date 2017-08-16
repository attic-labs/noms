// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

func nomsBlob(noms *kingpin.Application) (*kingpin.CmdClause, kCommandHandler) {
	blob := noms.Command("blob", "interact with blobs in a dataset")
	return blob, func() int {
		fmt.Println("do blob")
		return 0
	}
}
