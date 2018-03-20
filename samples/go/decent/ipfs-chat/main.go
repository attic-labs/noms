// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/ipfs"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/samples/go/decent/dbg"
	"github.com/attic-labs/noms/samples/go/decent/lib"
	"github.com/jroimartin/gocui"
	"gx/ipfs/QmXporsyf5xMvffd2eiTDoq85dNpYUynGJhfabzDjwP8uR/go-ipfs/core"
)

func main() {
	// allow short (-h) help
	kingpin.CommandLine.HelpFlag.Short('h')

	clientCmd := kingpin.Command("client", "runs the ipfs-chat client UI")
	clientTopic := clientCmd.Flag("topic", "IPFS pubsub topic to publish and subscribe to").Default("ipfs-chat").String()
	username := clientCmd.Flag("username", "username to sign in as").String()
	portIdx := clientCmd.Flag("port-idx", "a single digit to add to all port values: api, gateway and swarm (must be 0-8 inclusive)").Default("0").Int()
	clientDS := clientCmd.Arg("dataset", "the dataset spec to store chat data in").Required().String()

	importCmd := kingpin.Command("import", "imports data into a chat")
	importDir := importCmd.Flag("dir", "directory that contains data to import").Default("./data").ExistingDir()
	importDS := importCmd.Arg("dataset", "the dataq set spec to import chat data to").Required().String()

	daemonCmd := kingpin.Command("daemon", "runs a daemon that simulates filecoin, eagerly storing all chunks for a chat")
	daemonTopic := daemonCmd.Flag("topic", "IPFS pubsub topic to publish and subscribe to").Default("ipfs-chat").String()
	daemonInterval := daemonCmd.Flag("interval", "amount of time to wait before publishing state to network").Default("5s").Duration()
	daemonPortIdx := daemonCmd.Flag("port-idx", "a single digit to add to all port values: api, gateway and swarm (must be 0-8 inclusive)").Default("0").Int()
	daemonDS := daemonCmd.Arg("dataset", "the dataset spec indicating ipfs repo to use").Required().String()

	kingpin.CommandLine.Help = "A demonstration of using Noms to build a scalable multiuser collaborative application."

	expandRLimit()
	switch kingpin.Parse() {
	case "client":
		cInfo := lib.ClientInfo{
			Topic:    *clientTopic,
			Username: *username,
			Idx:      *portIdx,
			IsDaemon: false,
			Delegate: lib.IPFSEventDelegate{},
		}
		runClient(*clientDS, cInfo)
	case "import":
		lib.RunImport(*importDir, *importDS)
	case "daemon":
		cInfo := lib.ClientInfo{
			Topic:    *daemonTopic,
			Username: "daemon",
			Interval: *daemonInterval,
			Idx:      *daemonPortIdx,
			IsDaemon: true,
			Delegate: lib.IPFSEventDelegate{},
		}
		runDaemon(*daemonDS, cInfo)
	}
}

func runClient(ipfsSpec string, cInfo lib.ClientInfo) {
	dbg.SetLogger(lib.NewLogger(cInfo.Username))
	d.CheckError(ipfs.RegisterProtocols(ipfs.SetPortIdx(cInfo.Idx)))
	sp, err := spec.ForDataset(ipfsSpec)
	d.CheckErrorNoUsage(err)

	if !isIPFS(sp.Protocol) {
		fmt.Println("ipfs-chat requires an 'ipfs' dataset")
		os.Exit(1)
	}

	// Create/Open a new IPFS-backed database
	node, db := initIpfsDb(sp)

	dbg.Debug("my ID is %s", node.Identity.Pretty())

	// Get the head of specified dataset.
	ds := db.GetDataset(sp.Path.Dataset)
	ds, err = lib.InitDatabase(ds)
	d.PanicIfError(err)

	events := make(chan lib.ChatEvent, 1024)
	t := lib.CreateTermUI(events)
	defer t.Close()

	d.PanicIfError(t.Layout())
	t.ResetAuthors(ds)
	t.UpdateMessages(ds, nil, nil)

	go lib.ProcessChatEvents(node, ds, events, t, cInfo)
	go lib.ReceiveMessages(node, events, cInfo)

	if err := t.Gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		dbg.Debug("mainloop has exited, err:", err)
		log.Panicln(err)
	}
}

func runDaemon(ipfsSpec string, cInfo lib.ClientInfo) {
	dbg.SetLogger(log.New(os.Stdout, "", 0))
	d.CheckError(ipfs.RegisterProtocols(ipfs.SetPortIdx(cInfo.Idx)))
	sp, err := spec.ForDataset(ipfsSpec)
	d.CheckErrorNoUsage(err)

	if !isIPFS(sp.Protocol) {
		fmt.Println("ipfs-chat requires an 'ipfs' dataset")
		os.Exit(1)
	}

	// Create/Open a new IPFS-backed database
	node, db := initIpfsDb(sp)

	// Get the head of specified dataset.
	ds := db.GetDataset(sp.Path.Dataset)
	ds, err = lib.InitDatabase(ds)
	d.PanicIfError(err)

	events := make(chan lib.ChatEvent, 1024)
	handleSIGQUIT(events)

	go lib.ReceiveMessages(node, events, cInfo)
	lib.ProcessChatEvents(node, ds, events, nil, cInfo)
}

func handleSIGQUIT(events chan<- lib.ChatEvent) {
	sigChan := make(chan os.Signal)
	go func() {
		for range sigChan {
			stacktrace := make([]byte, 1024*1024)
			length := runtime.Stack(stacktrace, true)
			dbg.Debug(string(stacktrace[:length]))
			events <- lib.ChatEvent{EventType: lib.QuitEvent}
		}
	}()
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGQUIT)
}

// IPFS can use a lot of file decriptors. There are several bugs in the IPFS
// repo about this and plans to improve. For the time being, we bump the limits
// for this process.
func expandRLimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	d.Chk.NoError(err, "Unable to query file rlimit: %s", err)
	if rLimit.Cur < rLimit.Max {
		rLimit.Max = 64000
		rLimit.Cur = 64000
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		d.Chk.NoError(err, "Unable to increase number of open files limit: %s", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	d.Chk.NoError(err)

	err = syscall.Getrlimit(8, &rLimit)
	d.Chk.NoError(err, "Unable to query thread rlimit: %s", err)
	if rLimit.Cur < rLimit.Max {
		rLimit.Max = 64000
		rLimit.Cur = 64000
		err = syscall.Setrlimit(8, &rLimit)
		d.Chk.NoError(err, "Unable to increase number of threads limit: %s", err)
	}
	err = syscall.Getrlimit(8, &rLimit)
	d.Chk.NoError(err)
}

func initIpfsDb(sp spec.Spec) (*core.IpfsNode, datas.Database) {
	db := sp.GetDatabase()
	node := db.(ipfs.HasIPFSNode).IPFSNode()
	return node, db
}

func isIPFS(protocol string) bool {
	return protocol == "ipfs" || protocol == "ipfs-local"
}
