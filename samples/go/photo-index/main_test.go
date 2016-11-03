// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestBasics(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	clienttest.ClientTestSuite
}

func (s *testSuite) TestWin() {
	sp := fmt.Sprintf("ldb:%s::test", s.LdbDir)
	db, ds, _ := spec.GetDataset(sp)

	type Face struct {
		Name       string
		X, Y, W, H int
	}

	type Date struct {
		NsSinceEpoch int
	}

	type Photo struct {
		Id    string
		Title string
		Tags  types.Set
		Faces types.Set
		Sizes map[struct {
			Width  int
			Height int
		}]string
		DateTaken     Date
		DatePublished Date
		DateUpdated   Date
	}

	getTags := func(n int) types.Set {
		s := types.NewSet()
		for i := 0; i < n; i++ {
			s = s.Insert(types.String(fmt.Sprintf("tag%d", i)))
		}
		return s
	}

	getFaces := func(n int) types.Set {
		set := types.NewSet()
		for i := 0; i < n; i++ {
			v, err := marshal.Marshal(Face{
				fmt.Sprintf("harry%d", i),
				i, i, n, n,
			})
			s.NoError(err)
			set = set.Insert(v)
		}
		return set
	}

	begin := time.Unix(0, 0)
	getDate := func(n int) int {
		return int(begin.AddDate(n, n, n).UnixNano())
	}

	getPhoto := func(n int) Photo {
		return Photo{
			Id:    fmt.Sprintf("photo%d", n),
			Title: fmt.Sprintf("photo %d", n),
			Tags:  getTags(n),
			Sizes: map[struct{ Width, Height int }]string{
				{100, 100}: "100.jpg"},
			DateTaken:     Date{getDate(n)},
			DatePublished: Date{getDate(n) + 1},
			DateUpdated:   Date{getDate(n) + 2},
			Faces:         getFaces(n),
		}
	}

	photos := []Photo{}
	for i := 0; i < 5; i++ {
		photos = append(photos, getPhoto(i))
	}

	v, err := marshal.Marshal(photos)
	s.NoError(err)
	ds, err = db.CommitValue(ds, v)
	s.NoError(err)
	db.Close()

	_, _ = s.MustRun(main, []string{"--out-ds", "idx", "--db", s.LdbDir, "test"})

	db, ds, _ = spec.GetDataset(fmt.Sprintf("%s::idx", s.LdbDir))
	var idx struct {
		ByDate       map[int]types.Set
		ByTag        map[string]map[int]types.Set
		ByFace       map[string]map[int]types.Set
		ByYear       map[int]map[int]map[int]map[int]types.Set
		TagsByCount  map[int]types.Set
		FacesByCount map[int]types.Set
	}
	marshal.Unmarshal(ds.HeadValue(), &idx)

	s.Equal(5, len(idx.ByDate))
	for i := 0; i < 5; i++ {
		g := idx.ByDate[-getDate(i)]
		s.Equal(uint64(1), g.Len())
		p := idx.ByDate[-getDate(i)].First().(types.Struct)
		s.Equal(fmt.Sprintf("photo %d", i), string(p.Get("title").(types.String)))
	}

	s.Equal(4, len(idx.ByTag))
	for i := 0; i < 4; i++ {
		k := fmt.Sprintf("tag%d", i)
		v := idx.ByTag[k]
		s.Equal(4-i, len(v))
	}

	s.Equal(4, len(idx.ByFace))
	for i := 0; i < 4; i++ {
		k := fmt.Sprintf("harry%d", i)
		v := idx.ByFace[k]
		s.Equal(4-i, len(v))
	}

	s.Equal(5, len(idx.ByYear))
	for i := 0; i < 5; i++ {
		d := time.Unix(0, int64(getDate(i)))
		s.Equal(1, len(idx.ByYear[d.Year()]))
		s.Equal(1, len(idx.ByYear[d.Year()][int(d.Month())]))
		days := idx.ByYear[d.Year()][int(d.Month())][d.Day()]
		s.Equal(1, len(days))
		p := days[-getDate(i)].First().(types.Struct)
		s.Equal(fmt.Sprintf("photo %d", i), string(p.Get("title").(types.String)))
	}

	s.Equal(4, len(idx.TagsByCount))
	for i := 0; i < 4; i++ {
		tags := idx.TagsByCount[-4+i]
		s.Equal(1, int(tags.Len()))
		k := fmt.Sprintf("tag%d", i)
		s.True(tags.Has(types.String(k)))
	}

	s.Equal(4, len(idx.FacesByCount))
	for i := 0; i < 4; i++ {
		tags := idx.FacesByCount[-4+i]
		s.Equal(1, int(tags.Len()))
		k := fmt.Sprintf("harry%d", i)
		s.True(tags.Has(types.String(k)))
	}
}
