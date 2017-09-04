// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/ipfs"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/ipfs-chat/dbg"
	"github.com/ipfs/go-ipfs/core"
)

func runDaemon(topic string, interval time.Duration, ipfsSpec string, nodeIdx int) {
	dbg.SetLogger(log.New(os.Stdout, "", 0))

	sp, err := spec.ForDataset(ipfsSpec)
	d.CheckErrorNoUsage(err)
	ds := sp.GetDataset()

	node, cs := reconfigureIPFSChunkStore(sp, topic, nodeIdx)
	sourceDB := datas.NewDatabase(cs)
	sourceDS := sourceDB.GetDataset(ds.ID())

	cs = ipfs.NewChunkStorePrimitive(sp.DatabaseName, true, node)
	destDB := datas.NewDatabase(cs)
	destDS := destDB.GetDataset(ds.ID())
	destDS, err = InitDatabase(destDS)
	d.PanicIfError(err)

	dbg.Debug("Storing locally to:", sp.String())

	go replicate(node, topic, sourceDS, destDS, func(ds1 datas.Dataset) {
		destDS = ds1
	})

	for {
		Publish(node, topic, destDS.HeadRef().TargetHash())
		time.Sleep(interval)
	}
}

// replicate continually listens for commit hashes published by ipfs-chat nodes,
// ensures that all nodes are replicated locally, and merges new data into it's
// dataset when necessary.
func replicate(node *core.IpfsNode, topic string, sourceDS, destDS datas.Dataset, didChange func(ds datas.Dataset)) {
	sub, err := node.Floodsub.Subscribe(topic)
	d.Chk.NoError(err)

	var lastHash hash.Hash
	for {
		dbg.Debug("looking for msgs")
		msg, err := sub.Next(context.Background())
		d.PanicIfError(err)
		msgHash := hash.Parse(string(msg.Data))
		dbg.Debug("got msg, msgHash: %s, lastHash: %s", msgHash.String(), lastHash.String())
		if lastHash == msgHash {
			continue
		}
		lastHash = msgHash

		dbg.Debug("got update: %s from %s", msgHash, base64.StdEncoding.EncodeToString(msg.From))
		destDB := destDS.Database()
		destDB.Rebase()
		destDS = destDB.GetDataset(destDS.ID())
		d.PanicIfFalse(destDS.HasHead())

		dbg.Debug("syncing commits")
		pullCommits(msgHash, sourceDS.Database(), destDB)

		if msgHash == destDS.HeadRef().TargetHash() {
			dbg.Debug("received hash same as current head, nothing to do")
			continue
		}
		sourceCommit := destDB.ReadValue(msgHash)
		sourceRef := types.NewRef(sourceCommit)
		a, ok := datas.FindCommonAncestor(sourceRef, destDS.HeadRef(), destDB)
		if !ok {
			dbg.Debug("no common ancestor, cannot merge update!")
			continue
		}
		if a.Equals(sourceRef) {
			dbg.Debug("source commit was ancestor, nothing to do")
			continue
		}
		if a.Equals(destDS.HeadRef()) {
			dbg.Debug("fast-forward to source commit")
			destDS, err = destDB.SetHead(destDS, sourceRef)
			didChange(destDS)
			continue
		}

		left := destDS.HeadValue()
		right := sourceCommit.(types.Struct).Get("value")
		parent := a.TargetValue(destDB).(types.Struct).Get("value")

		merged, err := merge.ThreeWay(left, right, parent, destDB, nil, nil)
		if err != nil {
			dbg.Debug("could not merge received data: " + err.Error())
			continue
		}

		destDS, err = destDB.SetHead(destDS, destDB.WriteValue(datas.NewCommit(merged, types.NewSet(destDB, destDS.HeadRef(), sourceRef), types.EmptyStruct)))
		if err != nil {
			dbg.Debug("call failed to SetHead on destDB, err: %s", err)
		}
		didChange(destDS)
	}
}

func pullCommits(h hash.Hash, sourceDB, destDB datas.Database) {
	dbg.Debug("pullCommits, h: %s", h)

	// read
	v := destDB.ReadValue(h)
	if v != nil {
		dbg.Debug("pullCommits, found h: %s, commit:", h, types.EncodedValueMaxLines(v, 10))
		return
	}
	v = sourceDB.ReadValue(h)
	types.EncodedValue(v)
	dbg.Debug("pullCommits, read h: %s, commit:", h, types.EncodedValueMaxLines(v, 10))
	commit := v.(types.Struct)
	parents := commit.Get("parents").(types.Set)
	parents.IterAll(func(v types.Value) {
		ph := v.(types.Ref).TargetHash()
		pullCommits(ph, sourceDB, destDB)
	})
}
