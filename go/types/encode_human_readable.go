// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package types

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/writers"
	humanize "github.com/dustin/go-humanize"
)

// Human Readable Serialization
type hrsWriter struct {
	ind         int
	w           io.Writer
	lineLength  int
	floatFormat byte
	err         error
}

func (w *hrsWriter) maybeWriteIndentation() {
	if w.lineLength == 0 {
		for i := 0; i < w.ind && w.err == nil; i++ {
			_, w.err = io.WriteString(w.w, "  ")
		}
		w.lineLength = 2 * w.ind
	}
}

func (w *hrsWriter) write(s string) {
	if w.err != nil {
		return
	}
	w.maybeWriteIndentation()
	var n int
	n, w.err = io.WriteString(w.w, s)
	w.lineLength += n
}

func (w *hrsWriter) indent() {
	w.ind++
}

func (w *hrsWriter) outdent() {
	w.ind--
}

func (w *hrsWriter) newLine() {
	w.write("\n")
	w.lineLength = 0
}

// hexWriter is used to write blob byte data as "00 01 ... 0f\n10 11 .."
// hexWriter is an io.Writer that writes to an underlying hrsWriter.
type hexWriter struct {
	hrs         *hrsWriter
	count       uint
	sizeWritten bool
	size        uint64
}

func (w *hexWriter) Write(p []byte) (n int, err error) {
	for _, v := range p {
		if !w.sizeWritten && len(p) > 16 {
			w.hrs.write("  // ")
			w.hrs.write(humanize.Bytes(w.size))
			w.sizeWritten = true
			w.hrs.indent()
			w.hrs.newLine()
		}

		if w.count == 16 {
			w.hrs.newLine()
			w.count = 0
		} else if w.count != 0 {
			w.hrs.write(" ")
		}
		if v < 0x10 {
			w.hrs.write("0")
		}
		w.hrs.write(strconv.FormatUint(uint64(v), 16))
		if w.hrs.err != nil {
			err = w.hrs.err
			return
		}
		n++
		w.count++
	}

	if w.sizeWritten {
		w.hrs.outdent()
		w.hrs.newLine()
	}

	return
}

func (w *hrsWriter) Write(v Value) {
	switch v.Kind() {
	case BoolKind:
		w.write(strconv.FormatBool(bool(v.(Bool))))
	case NumberKind:
		w.write(strconv.FormatFloat(float64(v.(Number)), w.floatFormat, -1, 64))

	case StringKind:
		w.write(strconv.Quote(string(v.(String))))

	case BlobKind:
		w.write("blob {")
		blob := v.(Blob)
		encoder := &hexWriter{hrs: w, size: blob.Len()}
		_, w.err = io.Copy(encoder, blob.Reader())
		w.write("}")

	case ListKind:
		w.write("[")
		w.writeSize(v)
		w.indent()
		v.(List).Iter(func(v Value, i uint64) bool {
			if i == 0 {
				w.newLine()
			}
			w.Write(v)
			w.write(",")
			w.newLine()
			return w.err != nil
		})
		w.outdent()
		w.write("]")

	case MapKind:
		w.write("{")
		w.writeSize(v)
		w.indent()
		if !v.(Map).Empty() {
			w.newLine()
		}
		v.(Map).Iter(func(key, val Value) bool {
			w.Write(key)
			w.write(": ")
			w.Write(val)
			w.write(",")
			w.newLine()
			return w.err != nil
		})
		w.outdent()
		w.write("}")

	case RefKind:
		w.write("#")
		w.write(v.(Ref).TargetHash().String())

	case SetKind:
		w.write("{")
		w.writeSize(v)
		w.indent()
		if !v.(Set).Empty() {
			w.newLine()
		}
		v.(Set).Iter(func(v Value) bool {
			w.Write(v)
			w.write(",")
			w.newLine()
			return w.err != nil
		})
		w.outdent()
		w.write("}")

	case TypeKind:
		w.writeType(v.(*Type), map[*Type]struct{}{})

	case StructKind:
		w.writeStruct(v.(Struct))

	default:
		panic("unreachable")
	}
}

type hrsStructWriter struct {
	*hrsWriter
}

func (w hrsStructWriter) name(n string) {
	w.write("struct ")
	if n != "" {
		w.write(n)
		w.write(" ")
	}
	w.write("{")
	w.indent()
}

func (w hrsStructWriter) count(c uint64) {
	if c > 0 {
		w.newLine()
	}
}

func (w hrsStructWriter) fieldName(n string) {
	w.write(n)
	w.write(": ")
}

func (w hrsStructWriter) fieldValue(v Value) {
	w.Write(v)
	w.write(",")
	w.newLine()
}

func (w hrsStructWriter) end() {
	w.outdent()
	w.write("}")
}

func (w *hrsWriter) writeStruct(v Struct) {
	v.iterParts(hrsStructWriter{w})
}

func (w *hrsWriter) writeSize(v Value) {
	switch v.Kind() {
	case ListKind, MapKind, SetKind:
		l := v.(Collection).Len()
		if l < 4 {
			return
		}
		w.write(fmt.Sprintf("  // %s items", humanize.Comma(int64(l))))
	default:
		panic("unreachable")
	}
}

func (w *hrsWriter) writeType(t *Type, seenStructs map[*Type]struct{}) {
	switch t.TargetKind() {
	case BlobKind, BoolKind, NumberKind, StringKind, TypeKind, ValueKind:
		w.write(t.TargetKind().String())
	case ListKind, RefKind, SetKind, MapKind:
		w.write(t.TargetKind().String())
		w.write("<")
		for i, et := range t.Desc.(CompoundDesc).ElemTypes {
			if et.TargetKind() == UnionKind && len(et.Desc.(CompoundDesc).ElemTypes) == 0 {
				// If one of the element types is an empty union all the other element types must
				// also be empty union types.
				break
			}
			if i != 0 {
				w.write(", ")
			}
			w.writeType(et, seenStructs)
			if w.err != nil {
				break
			}
		}
		w.write(">")
	case UnionKind:
		for i, et := range t.Desc.(CompoundDesc).ElemTypes {
			if i != 0 {
				w.write(" | ")
			}
			w.writeType(et, seenStructs)
			if w.err != nil {
				break
			}
		}
	case StructKind:
		w.writeStructType(t, seenStructs)
	case CycleKind:
		name := string(t.Desc.(CycleDesc))
		d.PanicIfTrue(name == "")

		// This can happen for types that have unresolved cyclic refs
		w.write(fmt.Sprintf("UnresolvedCycle<%s>", name))
		if w.err != nil {
			return
		}
	default:
		panic("unreachable")
	}
}

func (w *hrsWriter) writeStructType(t *Type, seenStructs map[*Type]struct{}) {
	name := t.Desc.(StructDesc).Name
	if _, ok := seenStructs[t]; ok {
		w.write(fmt.Sprintf("Cycle<%s>", name))
		return
	}
	seenStructs[t] = struct{}{}

	desc := t.Desc.(StructDesc)
	w.write("Struct ")
	if desc.Name != "" {
		w.write(desc.Name + " ")
	}
	w.write("{")
	w.indent()
	if desc.Len() > 0 {
		w.newLine()
	}
	desc.IterFields(func(name string, t *Type, optional bool) {
		w.write(name)
		if optional {
			w.write("?")
		}
		w.write(": ")
		w.writeType(t, seenStructs)
		w.write(",")
		w.newLine()
	})
	w.outdent()
	w.write("}")
}

func encodedValueFormatMaxLines(v Value, floatFormat byte, maxLines uint32) string {
	var buf bytes.Buffer
	mlw := &writers.MaxLineWriter{Dest: &buf, MaxLines: maxLines}
	w := &hrsWriter{w: mlw, floatFormat: floatFormat}
	w.Write(v)
	if w.err != nil {
		d.Chk.IsType(writers.MaxLinesError{}, w.err, "Unexpected error: %s", w.err)
	}
	return buf.String()
}

func encodedValueFormat(v Value, floatFormat byte) string {
	var buf bytes.Buffer
	w := &hrsWriter{w: &buf, floatFormat: floatFormat}
	w.Write(v)
	d.Chk.NoError(w.err)
	return buf.String()
}

func EncodedIndexValue(v Value) string {
	return encodedValueFormat(v, 'f')
}

// EncodedValue returns a string containing the serialization of a value.
func EncodedValue(v Value) string {
	return encodedValueFormat(v, 'g')
}

// EncodedValueMaxLines returns a string containing the serialization of a value.
// The string is truncated at |maxLines|.
func EncodedValueMaxLines(v Value, maxLines uint32) string {
	return encodedValueFormatMaxLines(v, 'g', maxLines)
}

// WriteEncodedValue writes the serialization of a value
func WriteEncodedValue(w io.Writer, v Value) error {
	hrs := &hrsWriter{w: w, floatFormat: 'g'}
	hrs.Write(v)
	return hrs.err
}

// WriteEncodedValue writes the serialization of a value. Writing will be
// stopped and an error returned after |maxLines|.
func WriteEncodedValueMaxLines(w io.Writer, v Value, maxLines uint32) error {
	mlw := &writers.MaxLineWriter{Dest: w, MaxLines: maxLines}
	hrs := &hrsWriter{w: mlw, floatFormat: 'g'}
	hrs.Write(v)
	return hrs.err
}
