package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/search"
)

var (
	port = flag.Int("port", 8000, "")
)

func main() {
	flags := search.NewFlags()
	flag.Parse()
	s, ok := flags.CreateSearcher()
	if !ok {
		flag.Usage()
		return
	}

	server := search.NewSearchServer(s, *port)

	// Shutdown server gracefully so that profile may be written
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		server.Stop()
		s.Close()
	}()

	d.Try(func() {
		if util.MaybeStartCPUProfile() {
			defer util.StopCPUProfile()
		}
		server.Run()
	})
}
