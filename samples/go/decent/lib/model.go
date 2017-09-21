// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package lib

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/attic-labs/noms/samples/go/decent/dbg"
)

type Root struct {
	// Map<Key, Message>
	// Keys are strings like: <Ordinal>,<Author>
	// This scheme allows:
	// - map is naturally sorted in the right order
	// - conflicts will generally be avoided
	// - messages are editable
	Messages types.Map
	Index    types.Map
	Users    []string `noms:",set"`
}

type Message struct {
	Ordinal    uint64
	Author     string
	Body       string
	ClientTime datetime.DateTime
}

func (m Message) ID() string {
	return fmt.Sprintf("%020x/%s", m.ClientTime.UnixNano(), m.Author)
}

func AddMessage(body string, author string, clientTime time.Time, ds datas.Dataset) (datas.Dataset, error) {
	defer dbg.BoxF("AddMessage, body: %s", body)()
	root, err := getRoot(ds)
	if err != nil {
		return datas.Dataset{}, err
	}

	db := ds.Database()

	nm := Message{
		Author:     author,
		Body:       body,
		ClientTime: datetime.DateTime{clientTime},
		Ordinal:    root.Messages.Len(),
	}
	root.Messages = root.Messages.Edit().Set(types.String(nm.ID()), marshal.MustMarshal(db, nm)).Map()
	IndexNewMessage(db, &root, nm)
	newRoot := marshal.MustMarshal(db, root)
	ds, err = db.CommitValue(ds, newRoot)
	return ds, err
}

func InitDatabase(ds datas.Dataset) (datas.Dataset, error) {
	if ds.HasHead() {
		return ds, nil
	}
	db := ds.Database()
	root := Root{
		Index:    types.NewMap(db),
		Messages: types.NewMap(db),
	}
	return db.CommitValue(ds, marshal.MustMarshal(db, root))
}

func GetAuthors(ds datas.Dataset) []string {
	r, err := getRoot(ds)
	d.PanicIfError(err)
	return r.Users
}

func IndexNewMessage(vrw types.ValueReadWriter, root *Root, m Message) {
	defer dbg.BoxF("IndexNewMessage")()

	ti := NewTermIndex(vrw, root.Index)
	id := types.String(m.ID())
	root.Index = ti.Edit().InsertAll(GetTerms(m), id).Value().TermDocs
	root.Users = append(root.Users, m.Author)
}

func SearchIndex(ds datas.Dataset, search []string) types.Map {
	root, err := getRoot(ds)
	d.PanicIfError(err)
	idx := root.Index
	ti := NewTermIndex(ds.Database(), idx)
	ids := ti.Search(search)
	dbg.Debug("search for: %s, returned: %d", strings.Join(search, " "), ids.Len())
	return ids
}

var (
	punctPat = regexp.MustCompile("[[:punct:]]+")
	wsPat    = regexp.MustCompile("\\s+")
)

func TermsFromString(s string) []string {
	s1 := punctPat.ReplaceAllString(strings.TrimSpace(s), " ")
	terms := wsPat.Split(s1, -1)
	clean := []string{}
	for _, t := range terms {
		if t == "" {
			continue
		}
		clean = append(clean, strings.ToLower(t))
	}
	return clean
}

func GetTerms(m Message) []string {
	terms := TermsFromString(m.Body)
	terms = append(terms, TermsFromString(m.Author)...)
	return terms
}

func ListMessages(ds datas.Dataset, searchIds *types.Map, doneChan chan struct{}) (msgMap types.Map, mc chan types.String, err error) {
	//dbg.Debug("##### listMessages: entered")

	root, err := getRoot(ds)
	db := ds.Database()
	if err != nil {
		return types.NewMap(db), nil, err
	}
	msgMap = root.Messages

	mc = make(chan types.String)
	done := false
	go func() {
		<-doneChan
		done = true
		<-mc
		//dbg.Debug("##### listMessages: exiting 'done' goroutine")
	}()

	go func() {
		keyMap := msgMap
		if searchIds != nil {
			keyMap = *searchIds
		}
		i := uint64(0)
		for ; i < keyMap.Len() && !done; i++ {
			key, _ := keyMap.At(keyMap.Len() - i - 1)
			mc <- key.(types.String)
		}
		//dbg.Debug("##### listMessages: exiting 'for loop' goroutine, examined: %d", i)
		close(mc)
	}()
	return
}

func getRoot(ds datas.Dataset) (Root, error) {
	defer dbg.BoxF("getRoot")()

	db := ds.Database()
	root := Root{
		Messages: types.NewMap(db),
		Index:    types.NewMap(db),
	}
	// TODO: It would be nice if Dataset.MaybeHeadValue() or HeadValue()
	// would return just <value>, and it would be nil if not there, so you
	// could chain calls.
	if !ds.HasHead() {
		return root, nil
	}
	err := marshal.Unmarshal(ds.HeadValue(), &root)
	if err != nil {
		return Root{}, err
	}
	return root, nil
}
