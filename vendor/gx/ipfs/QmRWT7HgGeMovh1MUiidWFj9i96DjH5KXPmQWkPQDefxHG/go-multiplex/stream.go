package multiplex

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	mpool "gx/ipfs/QmRQhVisS8dmPbjBUthVkenn81pBxrx1GxE281csJhm2vL/go-msgio/mpool"
)

type Stream struct {
	id     uint64
	name   string
	header uint64
	dataIn chan []byte
	mp     *Multiplex

	extra []byte

	// exbuf is for holding the reference to the beginning of the extra slice
	// for later memory pool freeing
	exbuf []byte

	wDeadline time.Time
	rDeadline time.Time

	clLock       sync.Mutex
	closedLocal  bool
	closedRemote bool
}

func (s *Stream) Name() string {
	return s.name
}

func (s *Stream) receive(b []byte) {
	s.clLock.Lock()
	remoteClosed := s.closedRemote
	s.clLock.Unlock()
	if remoteClosed {
		log.Errorf("Received data from remote after stream was closed by them. (len = %d)", len(b))
		return
	}
	s.dataIn <- b
}

func (s *Stream) waitForData(ctx context.Context) error {
	if !s.rDeadline.IsZero() {
		dctx, cancel := context.WithDeadline(ctx, s.rDeadline)
		defer cancel()
		ctx = dctx
	}
	select {
	case read, ok := <-s.dataIn:
		if !ok {
			return io.EOF
		}
		s.extra = read
		s.exbuf = read
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Stream) Read(b []byte) (int, error) {
	if s.extra == nil {
		err := s.waitForData(context.Background())
		if err != nil {
			return 0, err
		}
	}
	n := copy(b, s.extra)
	if n < len(s.extra) {
		s.extra = s.extra[n:]
	} else {
		if s.exbuf != nil {
			mpool.ByteSlicePool.Put(uint32(cap(s.exbuf)), s.exbuf)
		}
		s.extra = nil
		s.exbuf = nil
	}
	return n, nil
}

func (s *Stream) Write(b []byte) (int, error) {
	var written int
	for written < len(b) {
		wl := len(b) - written
		if wl > MaxMessageSize {
			wl = MaxMessageSize
		}

		n, err := s.write(b[written : written+wl])
		if err != nil {
			return written, err
		}

		written += n
	}

	return written, nil
}

func (s *Stream) write(b []byte) (int, error) {
	if s.isClosed() {
		return 0, fmt.Errorf("cannot write to closed stream")
	}

	err := s.mp.sendMsg(s.header, b, s.wDeadline)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (s *Stream) isClosed() bool {
	s.clLock.Lock()
	defer s.clLock.Unlock()
	return s.closedLocal
}

func (s *Stream) Close() error {
	err := s.mp.sendMsg((s.id<<3)|Close, nil, time.Time{})

	s.clLock.Lock()
	if s.closedLocal {
		s.clLock.Unlock()
		return nil
	}

	remote := s.closedRemote
	s.closedLocal = true
	s.clLock.Unlock()

	if remote {
		s.mp.chLock.Lock()
		delete(s.mp.channels, s.id)
		s.mp.chLock.Unlock()
	}

	return err
}

func (s *Stream) SetDeadline(t time.Time) error {
	s.rDeadline = t
	s.wDeadline = t
	return nil
}

func (s *Stream) SetReadDeadline(t time.Time) error {
	s.rDeadline = t
	return nil
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.wDeadline = t
	return nil
}
