// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package profile

import (
	"flag"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/attic-labs/noms/go/d"
)

var (
	cpuProfile   = flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile   = flag.String("memprofile", "", "write memory profile to this file")
	blockProfile = flag.String("blockprofile", "", "write block profile to this file")
)

// MaybeStartProfile checks the -blockProfile, -cpuProfile, and -memProfile flag and, for each that is set, attempts to start gathering profiling data into the appropriate files. It returns an object with one method, Stop(), that must be called in order to flush profile data to disk before the process terminates.
func MaybeStartProfile() interface {
	Stop()
} {
	p := &prof{}
	if *blockProfile != "" {
		f, err := os.Create(*blockProfile)
		d.Exp.NoError(err)
		runtime.SetBlockProfileRate(1)
		p.bp = f
	}
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		d.Exp.NoError(err)
		pprof.StartCPUProfile(f)
		p.cpu = f
	}
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		d.Exp.NoError(err)
		p.mem = f
	}
	return p
}

type prof struct {
	bp  io.WriteCloser
	cpu io.Closer
	mem io.WriteCloser
}

func (p *prof) Stop() {
	if p.bp != nil {
		pprof.Lookup("block").WriteTo(p.bp, 0)
		p.bp.Close()
		runtime.SetBlockProfileRate(0)
	}
	if p.cpu != nil {
		p.cpu.Close()
	}
	if p.mem != nil {
		pprof.WriteHeapProfile(p.mem)
		p.mem.Close()
	}
}
