// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	"github.com/attic-labs/noms/go/util/status"
	humanize "github.com/dustin/go-humanize"
)

func nomsSync(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("sync", "Efficiently moves values between databases.")
	source := cmd.Arg("source-value", "see Spelling Values at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()
	dest := cmd.Arg("dest-dataset", "see Spelling Datasets at https://github.com/attic-labs/noms/blob/master/doc/spelling.md").Required().String()

	return cmd, func(_ string) int {
		cfg := config.NewResolver()
		sourceStore, sourceObj, err := cfg.GetPath(*source)
		d.CheckError(err)
		defer sourceStore.Close()

		if sourceObj == nil {
			d.CheckErrorNoUsage(fmt.Errorf("Object not found: %s", *source))
		}

		sinkDB, sinkDataset, err := cfg.GetDataset(*dest)
		d.CheckError(err)
		defer sinkDB.Close()

		start := time.Now()
		progressCh := make(chan datas.PullProgress)
		lastProgressCh := make(chan datas.PullProgress)

		go func() {
			var last datas.PullProgress

			for info := range progressCh {
				last = info
				if info.KnownCount == 1 {
					// It's better to print "up to date" than "0% (0/1); 100% (1/1)".
					continue
				}

				if status.WillPrint() {
					pct := 100.0 * float64(info.DoneCount) / float64(info.KnownCount)
					status.Printf("Syncing - %.2f%% (%s/s)", pct, bytesPerSec(info.ApproxWrittenBytes, start))
				}
			}
			lastProgressCh <- last
		}()

		sourceRef := types.NewRef(sourceObj)
		sinkRef, sinkExists := sinkDataset.MaybeHeadRef()
		nonFF := false
		err = d.Try(func() {
			defer profile.MaybeStartProfile().Stop()
			datas.Pull(sourceStore, sinkDB, sourceRef, progressCh)

			var err error
			sinkDataset, err = sinkDB.FastForward(sinkDataset, sourceRef)
			if err == datas.ErrMergeNeeded {
				sinkDataset, err = sinkDB.SetHead(sinkDataset, sourceRef)
				nonFF = true
			}
			d.PanicIfError(err)
		})

		if err != nil {
			log.Fatal(err)
		}

		close(progressCh)
		if last := <-lastProgressCh; last.DoneCount > 0 {
			status.Printf("Done - Synced %s in %s (%s/s)",
				humanize.Bytes(last.ApproxWrittenBytes), since(start), bytesPerSec(last.ApproxWrittenBytes, start))
			status.Done()
		} else if !sinkExists {
			fmt.Printf("All chunks already exist at destination! Created new dataset %s.\n", *dest)
		} else if nonFF && !sourceRef.Equals(sinkRef) {
			fmt.Printf("Abandoning %s; new head is %s\n", sinkRef.TargetHash(), sourceRef.TargetHash())
		} else {
			fmt.Printf("Dataset %s is already up to date.\n", *dest)
		}

		return 0
	}
}

func bytesPerSec(bytes uint64, start time.Time) string {
	bps := float64(bytes) / float64(time.Since(start).Seconds())
	return humanize.Bytes(uint64(bps))
}

func since(start time.Time) string {
	round := time.Second / 100
	now := time.Now().Round(round)
	return now.Sub(start.Round(round)).String()
}
