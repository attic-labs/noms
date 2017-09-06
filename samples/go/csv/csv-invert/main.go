// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/profile"
	flag "github.com/juju/gnuflag"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <dataset-to-invert> <output-dataset>\n", os.Args[0])
		flag.PrintDefaults()
	}

	profile.RegisterProfileFlags(flag.CommandLine)
	flag.Parse(true)

	if flag.NArg() != 2 {
		flag.Usage()
		return
	}

	cfg := config.NewResolver()
	inDB, inDS, err := cfg.GetDataset(flag.Arg(0))
	d.CheckError(err)
	defer inDB.Close()

	head, present := inDS.MaybeHead()
	if !present {
		d.CheckErrorNoUsage(fmt.Errorf("The dataset %s has no head", flag.Arg(0)))
	}
	v := head.Get(datas.ValueField)
	l, isList := v.(types.List)
	if !isList {
		d.CheckErrorNoUsage(fmt.Errorf("The head value of %s is not a list, but rather %s", flag.Arg(0), types.TypeOf(v).Describe()))
	}

	outDB, outDS, err := cfg.GetDataset(flag.Arg(1))
	defer outDB.Close()

	defer profile.MaybeStartProfile().Stop()
	streams := map[string]chan types.Value{}
	lists := map[string]<-chan types.List{}
	lowers := map[string]string{}

	sDesc := types.TypeOf(l).Desc.(types.CompoundDesc).ElemTypes[0].Desc.(types.StructDesc)
	sDesc.IterFields(func(name string, t *types.Type, optional bool) {
		lowerName := strings.ToLower(name)
		if _, present := streams[lowerName]; !present {
			streams[lowerName] = make(chan types.Value, 1024)
			lists[lowerName] = types.NewStreamingList(outDB, streams[lowerName])
		}
		lowers[name] = lowerName
	})

	columnVals := make(map[string]types.Value, len(streams))
	emptyString := types.String("")
	l.IterAll(func(v types.Value, index uint64) {
		for lowerName := range streams {
			columnVals[lowerName] = emptyString
		}
		v.(types.Struct).IterFields(func(name string, value types.Value) {
			columnVals[lowers[name]] = value
		})
		for lowerName, stream := range streams {
			stream <- columnVals[lowerName]
		}
	})

	invertedStructData := types.StructData{}
	for lowerName, stream := range streams {
		close(stream)
		invertedStructData[lowerName] = <-lists[lowerName]
	}
	str := types.NewStruct("Columnar", invertedStructData)

	parents := types.NewSet(outDB)
	if headRef, present := outDS.MaybeHeadRef(); present {
		parents = types.NewSet(outDB, headRef)
	}

	_, err = outDB.Commit(outDS, str, datas.CommitOptions{Parents: parents, Meta: head.Get(datas.MetaField).(types.Struct)})
	d.PanicIfError(err)
}
