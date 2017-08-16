// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"os"
	"os/signal"
	"syscall"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/util/profile"
)

func nomsServe(noms *kingpin.Application) (*kingpin.CmdClause, CommandHandler) {
	serve := noms.Command("serve", `Serves a Noms database over HTTP

See Spelling Objects at https://github.com/attic-labs/noms/blob/master/doc/spelling.md for details on the database argument.
`)
	port := serve.Flag("port", "port to listen on for HTTP requests").Default("8000").Int()
	database := AddDatabaseArg(serve)

	return serve, func() int {
		cfg := config.NewResolver()
		cs, err := cfg.GetChunkStore(*database)
		d.CheckError(err)
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
}
