// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package status prints status messages to a console, overwriting previous values.
package status

import (
	"fmt"
)

const (
	clearLine = "\x1b[2K\r"
)

func Clear() {
	Printf("")
}

func Printf(format string, args ...interface{}) {
	fmt.Printf(clearLine+format, args...)
}

func Done() {
	fmt.Println("")
}
