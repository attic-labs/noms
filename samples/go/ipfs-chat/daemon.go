// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/ipfs"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/ipfs-chat/dbg"
	"github.com/attic-labs/noms/samples/go/ipfs-chat/lib"
	"github.com/ipfs/go-ipfs/core"
)

func runDaemon(ipfsSpec string, topic string, nodeIdx int, interval time.Duration) {
	dbg.SetLogger(log.New(os.Stdout, "", 0))

	stackDumpOnSIGQUIT()
	sp, err := spec.ForDataset(ipfsSpec)
	d.CheckErrorNoUsage(err)

	if !isIPFS(sp.Protocol) {
		fmt.Println("ipfs-chat requires an 'ipfs' dataset")
		os.Exit(1)
	}

	// Create/Open a new network chunkstore
	node, cs := initChunkStore(sp, nodeIdx)
	db := datas.NewDatabase(cs)

	// Get the head of specified dataset.
	ds := db.GetDataset(sp.Path.Dataset)
	ds, err = lib.InitDatabase(ds)
	d.PanicIfError(err)

	dbg.Debug("Storing locally to: %s", sp.String())

	go replicate(node, topic, ds, func(ds1 datas.Dataset) {
		ds = ds1
	})

	for {
		Publish(node, topic, ds.HeadRef().TargetHash())
		time.Sleep(interval)
	}
}

// replicate continually listens for commit hashes published by ipfs-chat nodes,
// ensures that all nodes are replicated locally, and merges new data into it's
// dataset when necessary.
func replicate(node *core.IpfsNode, topic string, ds datas.Dataset, didChange func(ds datas.Dataset)) {
	mchan := make(chan hash.Hash, 1024)

	go func() {
		recieveMessages(node, topic, mchan)
	}()

	for h := range mchan {
		processHash := func(h hash.Hash) {
			defer dbg.BoxF("processingHash: %s, cid: %s", h, ipfs.NomsHashToCID(h))()

			db := ds.Database()
			pinBlocks(node, h, db, 0)

			headRef := ds.HeadRef()
			if h == headRef.TargetHash() {
				dbg.Debug("received hash same as current head, nothing to do")
				return
			}

			dbg.Debug("reading value: %s", h)
			hCommit := db.ReadValue(h)
			sourceRef := types.NewRef(hCommit)

			dbg.Debug("Finding common ancestor for merge, sourceRef: %s, headRef: %s", sourceRef.TargetHash(), headRef.TargetHash())
			a, ok := datas.FindCommonAncestor(sourceRef, headRef, db)
			if !ok {
				dbg.Debug("no common ancestor, cannot merge update!")
				return
			}
			dbg.Debug("Checking if source commit is ancestor")
			if a.Equals(sourceRef) {
				dbg.Debug("source commit was ancestor, nothing to do")
				return
			}
			if a.Equals(headRef) {
				dbg.Debug("fast-forward to source commit")
				ds, err := db.SetHead(ds, sourceRef)
				d.Chk.NoError(err)
				didChange(ds)
				return
			}

			dbg.Debug("We have mergeable commit")
			left := ds.HeadValue()
			right := hCommit.(types.Struct).Get("value")
			parent := a.TargetValue(db).(types.Struct).Get("value")

			dbg.Debug("Starting three-way commit")
			merged, err := merge.ThreeWay(left, right, parent, db, nil, nil)
			if err != nil {
				dbg.Debug("could not merge received data: " + err.Error())
				return
			}

			dbg.Debug("setting new datasetHead on localDB")
			newCommit := datas.NewCommit(merged, types.NewSet(db, ds.HeadRef(), sourceRef), types.EmptyStruct)
			commitRef := db.WriteValue(newCommit)
			dbg.Debug("wrote new commit: %s", commitRef.TargetHash())
			ds, err = db.SetHead(ds, commitRef)
			if err != nil {
				dbg.Debug("call to db.SetHead on failed, err: %s", err)
			}
			pinBlocks(node, ds.HeadRef().TargetHash(), db, 0)
			newH := ds.HeadRef().TargetHash()
			dbg.Debug("merged commit, dataset: %s, head: %s, cid: %s", ds.ID(), newH, ipfs.NomsHashToCID(newH))
			didChange(ds)
		}
		processHash(h)
	}
}

func pinBlocks(node *core.IpfsNode, h hash.Hash, netDB datas.Database, level int) {
	defer func() {
		if level == 0 {
			dbg.Debug("EXITING PULL-COMMITS!!!")
		}
	}()

	cid := ipfs.NomsHashToCID(h)
	_, pinned, err := node.Pinning.IsPinned(cid)
	d.Chk.NoError(err)
	if pinned {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v := netDB.ReadValue(h)
	d.Chk.NotNil(v)

	v.WalkRefs(func(r types.Ref) {
		pinBlocks(node, r.TargetHash(), netDB, level+1)
	})

	n, err := node.DAG.Get(ctx, cid)
	d.Chk.NoError(err)
	err = node.Pinning.Pin(ctx, n, false)
	d.Chk.NoError(err)
}

func stackDumpOnSIGQUIT() {
	sigChan := make(chan os.Signal)
	go func() {
		stacktrace := make([]byte, 1024*1024)
		for range sigChan {
			length := runtime.Stack(stacktrace, true)
			fmt.Println(string(stacktrace[:length]))
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT)
}
