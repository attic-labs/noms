// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package outputpager

import (
	"flag"
	"io"
	"os"
	"os/exec"

	"github.com/attic-labs/noms/go/d"
	goisatty "github.com/mattn/go-isatty"
)

var (
	noPager         bool
	flagsRegistered = false
)

type Pager struct {
	Writer        io.Writer
	cmd           *exec.Cmd
	stdin, stdout *os.File
}

func NewOrNil() *Pager {
	if noPager || !IsStdoutTty() {
		return nil
	}

	lessPath, err := exec.LookPath("less")
	d.Chk.NoError(err)

	// -F ... Quit if entire file fits on first screen.
	// -S ... Chop (truncate) long lines rather than wrapping.
	// -R ... Output "raw" control characters.
	// -X ... Don't use termcap init/deinit strings.
	cmd := exec.Command(lessPath, "-FSRX")
	stdin, stdout, err := os.Pipe()
	d.Chk.NoError(err)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = stdin
	return &Pager{stdout, cmd, stdin, stdout}
}

func (p *Pager) RunAndExit() {
	err := p.cmd.Run()
	d.Chk.NoError(err)
	os.Exit(0)
}

func (p *Pager) Stop() {
	p.stdin.Close()
	p.stdout.Close()
}

func RegisterOutputpagerFlags(flags *flag.FlagSet) {
	if !flagsRegistered {
		flagsRegistered = true
		flags.BoolVar(&noPager, "no-pager", false, "suppress paging functionality")
	}
}

func IsStdoutTty() bool {
	return goisatty.IsTerminal(os.Stdout.Fd())
}
