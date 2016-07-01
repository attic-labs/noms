// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/samples/go/util"
)

var (
	serveFlagSet = flag.NewFlagSet("serve", flag.ExitOnError)
	port         = serveFlagSet.Int("port", 8000, "")
)

var nomsServe = &nomsCommand{
	Run:       runServe,
	UsageLine: "serve [options] <database>",
	Short:     "Serves a Noms database over HTTP",
	Long:      "See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.",
	Flag:      serveFlagSet,
	Nargs:     1,
}

func init() {
	spec.RegisterDatabaseFlags(serveFlagSet)
}

func runServe(args []string) int {
	cs, err := spec.GetChunkStore(args[0])
	util.CheckError(err)
	server := datas.NewRemoteDatabaseServer(cs, *port)

	// Shutdown server gracefully so that profile may be written
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		server.Stop()
	}()

	d.Try(func() {
		defer profile.MaybeStartProfile().Stop()
		server.Run()
	})
	return 0
}
