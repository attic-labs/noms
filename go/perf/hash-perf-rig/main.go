// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"time"

	"github.com/attic-labs/kingpin"
	"github.com/codahale/blake2"
	humanize "github.com/dustin/go-humanize"
	"github.com/kch42/buzhash"
)

func main() {
	useSHA := kingpin.Flag("use-sha", "<default>=no hashing, 1=sha1, 256=sha256, 512=sha512, blake=blake2b").String()
	useBH := kingpin.Flag("use-bh", "whether we buzhash the bytes").Bool()
	bigFile := kingpin.Arg("bigfile", "input file to chunk").Required().String()

	kingpin.Parse()

	bh := buzhash.NewBuzHash(64 * 8)
	f, _ := os.Open(*bigFile)
	defer f.Close()
	t0 := time.Now()
	buf := make([]byte, 4*1024)
	l := uint64(0)

	var h hash.Hash
	if *useSHA == "1" {
		h = sha1.New()
	} else if *useSHA == "256" {
		h = sha256.New()
	} else if *useSHA == "512" {
		h = sha512.New()
	} else if *useSHA == "blake" {
		h = blake2.NewBlake2B()
	}

	for {
		n, err := f.Read(buf)
		l += uint64(n)
		if err == io.EOF {
			break
		}
		s := buf[:n]
		if h != nil {
			h.Write(s)
		}
		if *useBH {
			bh.Write(s)
		}
	}

	t1 := time.Now()
	d := t1.Sub(t0)
	fmt.Printf("Read  %s in %s (%s/s)\n", humanize.Bytes(l), d, humanize.Bytes(uint64(float64(l)/d.Seconds())))
	digest := []byte{}
	if h != nil {
		fmt.Printf("%x\n", h.Sum(digest))
	}
}
