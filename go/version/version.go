// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package version contains utilities for working with the Noms format version.
package version

import (
	"os"
)

const (
	USE_VNEXT_ENV = "NOMS_VERSION_NEXT"
)

var (
	// TODO: generate this from some central thing with go generate, so that JS and Go can be easily kept in sync
	nomsVersionStable = "7"
	nomsVersionNext   = "8"
	NomsGitSHA        = "<developer build>"
)

func Current() string {
	if IsNext() {
		return nomsVersionNext
	} else {
		return nomsVersionStable
	}
}

func IsNext() bool {
	return os.Getenv(USE_VNEXT_ENV) == "1"
}

func IsStable() bool {
	return !IsNext()
}

func UseNext(v bool) {
	os.Setenv(USE_VNEXT_ENV, "1")
}
