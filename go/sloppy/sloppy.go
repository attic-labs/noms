// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import "github.com/attic-labs/noms/go/d"

// Sloppy is similar to, and in all ways worse than Google's snappy compression
// algorithm (https://github.com/google/snappy). However, it has one important
// property: it prefers copies to closer byte sequences.
//
// For example, snappy would logically encode ABCDxABCDyABCDz as:
//
//  ["ABCDx", copy(5, 4), "y", copy(10, 4), "z"]
//
// while sloppy will encode it as
//
//  ["ABCDx", copy(5, 4), "y", copy(5, 4), "z"]
//
// This property is useful as sloppy's purpose is to roughly estimate the
// effect of snappy compressing a given byte sequence, while allowing a
// change in any byte of the input stream to only effect bytes in the output
// stream within an N byte window. This allows a byte stream to be fed to
// sloppy, then into a rolling hash and result in chunk sizes AFTER compression
// with snappy be to close to a choosen target average.
//
// Sloppy is slower and less effective than snappy and its output cannot be
// decoded. If you are using sloppy for anything other than the purpose above,
// you are almost certainly doing it wrong.

const (
	maxOffset    = 1<<12 - 1
	maxTableSize = 1 << 14
	maxLength    = 1<<12 - 1
	tableMask    = maxTableSize - 1
	shift        = uint32(20)
)

type Sloppy struct {
	enc                      encoder
	idx, maxOffset           int
	matching                 bool
	matchOffset, matchLength int
	table                    [maxTableSize]uint16
}

// New returns a new sloppy encoder which will encode to |f|. If |f| ever
// returns false, then encoding ends immediately. |max| is the maximum offset
// with which a copy will refer back in the input stream.
func New(f func(b byte) bool, max uint16) *Sloppy {
	d.PanicIfTrue(max > maxOffset)

	return &Sloppy{
		binaryEncoder{f},
		0, int(max),
		false,
		0, 0,
		[maxTableSize]uint16{},
	}
}

// Update continues the encoding of a given input stream. The caller is expected
// to call update after having (ONLY) appended bytes to |src|. When |Update|
// returns, sloppy will have emitted 0 or more literals or copies by calling
// the |sf.f|. Note that sloppy will ALWAYS buffer the final three bytes of
// input.
func (sl *Sloppy) Update(src []byte) {
	for ; sl.idx < len(src)-3; sl.idx++ {
		// TODO: Implement skip heuristic

		// If there are at least four bytes left, compute hash
		if sl.idx < len(src)-3 {
			nextHash := fbhash(load32(src, sl.idx), shift)
			if !sl.matching && sl.idx > 0 {
				// Look for new match
				matchPos := int(sl.table[nextHash&tableMask])
				if sl.idx-matchPos <= sl.maxOffset && src[sl.idx] == src[matchPos] {
					sl.matching = true
					sl.matchOffset = matchPos
					sl.matchLength = 0
				}
			}

			// Store new hashed offset
			sl.table[nextHash&tableMask] = uint16(sl.idx)
		}

		if sl.matching {
			if sl.matchLength <= maxLength && src[sl.idx] == src[sl.matchOffset+sl.matchLength] {
				// Extend match
				sl.matchLength++
				continue
			}

			// Terminate match
			if sl.copy(sl.idx) {
				return
			}
		}

		if sl.enc.emitLiteral(src[sl.idx]) {
			return
		}
	}
}

func (sl *Sloppy) Reset() {
	sl.matching = false
	sl.idx = 0
	sl.matchOffset = 0
	sl.matchLength = 0
	sl.table = [maxTableSize]uint16{}
}

func (sl *Sloppy) copy(i int) bool {
	off, len := uint16(i-(sl.matchOffset+sl.matchLength)), uint16(sl.matchLength)
	sl.matching = false
	sl.matchOffset = 0
	sl.matchLength = 0
	return sl.enc.emitCopy(off, len)
}

type encoder interface {
	emitLiteral(b byte) bool
	emitCopy(offset, length uint16) bool
}

type binaryEncoder struct {
	f func(b byte) bool
}

func (be binaryEncoder) emitLiteral(b byte) bool {
	return be.f(b)
}

func (be binaryEncoder) emitCopy(offset, length uint16) bool {
	// all copies are encoded as 3 bytes.
	// 12 bits for offset and 12 bits for length

	// 8 MSBits of offset
	if be.f(byte(offset >> 4)) {
		return true
	}

	// 4 LSBits offset | 4 MSBits length
	if be.f(byte(offset<<4) | byte(length>>4)) {
		return true
	}

	// 8 LSBits of length
	if be.f(byte(length)) {
		return true
	}

	return false
}

func fbhash(u, shift uint32) uint32 {
	return (u * 0x1e35a7bd) >> shift
}

func load32(b []byte, i int) uint32 {
	b = b[i : i+4 : len(b)] // Help the compiler eliminate bounds checks on the next line.
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
