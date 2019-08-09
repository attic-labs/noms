// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/util/profile"
)

func nomsServe(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("serve", "Serves a Noms database over HTTP.")
	port := cmd.Flag("port", "port to listen on").Default("8080").Int()
	db := cmd.Arg("db", "database to work with - see Spelling Databases at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()

	return cmd, func(_ string) int {
		cfg := config.NewResolver()
		cs, err := cfg.GetChunkStore(*db)
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
