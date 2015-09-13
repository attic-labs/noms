package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
)

var (
	host = flag.String("host", ":8000", "TCP address to listen on - e.g., foo.noms.io or 12.16.8.4:48")
)

func main() {
	flags := chunks.NewFlags()
	flag.Parse()
	cs := flags.CreateStore()
	if cs == nil {
		flag.Usage()
		return
	}

	server := chunks.NewHttpStoreServer(cs, *host)

	// Shutdown server gracefully so that profile may be written
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		server.Stop()
		cs.Close()
	}()

	d.Try(func() {
		if util.MaybeStartCPUProfile() {
			defer util.StopCPUProfile()
		}
		server.Run()
	})
}
