// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/ipfs"
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/samples/go/ipfs-chat/dbg"
	"github.com/ipfs/go-ipfs/core"
)

// MergeMessages continually listens for commit hashes published by ipfs-chat. It
// merges new messages into it's existing dataset when necessary and if an actual
// merge was necessary, it re-publishes the new commit.
func mergeMessages(node *core.IpfsNode, topic string, ds datas.Dataset, didChange func(ds datas.Dataset)) {
	mchan := make(chan hash.Hash, 1024)

	go func() {
		recieveMessages(node, topic, mchan)
	}()

	for h := range mchan {
		processHash := func(h hash.Hash) {
			defer dbg.BoxF("processingHash: %s, cid: %s", h, ipfs.NomsHashToCID(h))()
			defer limitRateF()()

			db := ds.Database()
			db.Rebase()
			ds = db.GetDataset(ds.ID())
			d.PanicIfFalse(ds.HasHead())

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

			newH := ds.HeadRef().TargetHash()
			dbg.Debug("merged commit, dataset: %s, head: %s, cid: %s", ds.ID(), newH, ipfs.NomsHashToCID(newH))
			Publish(node, topic, commitRef.TargetHash())
			didChange(ds)
		}
		processHash(h)
	}
}

func recieveMessages(node *core.IpfsNode, topic string, mchan chan hash.Hash) {
	sub, err := node.Floodsub.Subscribe(topic)
	d.Chk.NoError(err)

	var lastHash hash.Hash
	dbg.Debug("start listening for msgs on channel: %s", topic)
	for {
		msg, err := sub.Next(context.Background())
		d.PanicIfError(err)
		hstring := strings.TrimSpace(string(msg.Data))
		sender := base64.StdEncoding.EncodeToString(msg.From)
		h, ok := hash.MaybeParse(hstring)
		if !ok {
			dbg.Debug("mergeMsgs: received unknown msg: %s from: %s", hstring, sender)
			continue
		}
		if lastHash == h {
			continue
		}
		lastHash = h
		dbg.Debug("got update: %s from %s", h, sender)
		mchan <- h
	}
	dbg.Debug("should never get here")
	panic("should never get here")
}

func Publish(node *core.IpfsNode, topic string, h hash.Hash) {
	dbg.Debug("publishing to topic: %s, hash: %s", topic, h)
	node.Floodsub.Publish(topic, []byte(h.String()+"\r\n"))
}
