// Copyright 2014 The Go Authors.
// See https://code.google.com/p/go/source/browse/CONTRIBUTORS
// Licensed under the same terms as Go itself:
// https://code.google.com/p/go/source/browse/LICENSE

package bufpipe

import (
	"io"
	"reflect"
	"testing"
)

var bufferReadTests = []struct {
	buf      Buffer
	read, wn int
	werr     error
	wp       []byte
	wbuf     Buffer
}{
	{
		Buffer{[]byte{'a', 0}, 0, 1, false, nil},
		5, 1, nil, []byte{'a'},
		Buffer{[]byte{'a', 0}, 1, 1, false, nil},
	},
	{
		Buffer{[]byte{'a', 0}, 0, 1, true, io.EOF},
		5, 1, io.EOF, []byte{'a'},
		Buffer{[]byte{'a', 0}, 1, 1, true, io.EOF},
	},
	{
		Buffer{[]byte{0, 'a'}, 1, 2, false, nil},
		5, 1, nil, []byte{'a'},
		Buffer{[]byte{0, 'a'}, 2, 2, false, nil},
	},
	{
		Buffer{[]byte{0, 'a'}, 1, 2, true, io.EOF},
		5, 1, io.EOF, []byte{'a'},
		Buffer{[]byte{0, 'a'}, 2, 2, true, io.EOF},
	},
	{
		Buffer{[]byte{}, 0, 0, false, nil},
		5, 0, errReadEmpty, []byte{},
		Buffer{[]byte{}, 0, 0, false, nil},
	},
	{
		Buffer{[]byte{}, 0, 0, true, io.EOF},
		5, 0, io.EOF, []byte{},
		Buffer{[]byte{}, 0, 0, true, io.EOF},
	},
}

func TestBufferRead(t *testing.T) {
	for i, tt := range bufferReadTests {
		read := make([]byte, tt.read)
		n, err := tt.buf.Read(read)
		if n != tt.wn {
			t.Errorf("#%d: wn = %d want %d", i, n, tt.wn)
			continue
		}
		if err != tt.werr {
			t.Errorf("#%d: werr = %v want %v", i, err, tt.werr)
			continue
		}
		read = read[:n]
		if !reflect.DeepEqual(read, tt.wp) {
			t.Errorf("#%d: read = %+v want %+v", i, read, tt.wp)
		}
		if !reflect.DeepEqual(tt.buf, tt.wbuf) {
			t.Errorf("#%d: buf = %+v want %+v", i, tt.buf, tt.wbuf)
		}
	}
}

var bufferWriteTests = []struct {
	buf       Buffer
	write, wn int
	werr      error
	wbuf      Buffer
}{
	{
		buf: Buffer{
			buf: []byte{},
		},
		wbuf: Buffer{
			buf: []byte{},
		},
	},
	{
		buf: Buffer{
			buf: []byte{1, 'a'},
		},
		write: 1,
		wn:    1,
		wbuf: Buffer{
			buf: []byte{0, 'a'},
			w:   1,
		},
	},
	{
		buf: Buffer{
			buf: []byte{'a', 1},
			r:   1,
			w:   1,
		},
		write: 2,
		wn:    2,
		wbuf: Buffer{
			buf: []byte{0, 0},
			w:   2,
		},
	},
	{
		buf: Buffer{
			buf:    []byte{},
			r:      1,
			closed: true,
		},
		write: 5,
		werr:  errWriteClosed,
		wbuf: Buffer{
			buf:    []byte{},
			r:      1,
			closed: true,
		},
	},
	{
		buf: Buffer{
			buf: []byte{},
		},
		write: 5,
		werr:  errWriteFull,
		wbuf: Buffer{
			buf: []byte{},
		},
	},
}

func TestBufferWrite(t *testing.T) {
	for i, tt := range bufferWriteTests {
		n, err := tt.buf.Write(make([]byte, tt.write))
		if n != tt.wn {
			t.Errorf("#%d: wrote %d bytes; want %d", i, n, tt.wn)
			continue
		}
		if err != tt.werr {
			t.Errorf("#%d: error = %v; want %v", i, err, tt.werr)
			continue
		}
		if !reflect.DeepEqual(tt.buf, tt.wbuf) {
			t.Errorf("#%d: buf = %+v; want %+v", i, tt.buf, tt.wbuf)
		}
	}
}
