package swarm

import (
	"context"
	"sync"

	addrutil "gx/ipfs/QmVJGsPeK3vwtEyyTxpCs47yjBYMmYsAhEouPDF3Gb2eK3/go-addr-util"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	iconn "gx/ipfs/QmfQAY7YU4fQi3sjGLs1hwkM2Aq7dxgDyoMjaKN4WBWvcB/go-libp2p-interface-conn"
)

type dialResult struct {
	Conn iconn.Conn
	Addr ma.Multiaddr
	Err  error
}

type dialJob struct {
	addr    ma.Multiaddr
	peer    peer.ID
	ctx     context.Context
	resp    chan dialResult
	success bool
}

func (dj *dialJob) cancelled() bool {
	select {
	case <-dj.ctx.Done():
		return true
	default:
		return false
	}
}

type dialLimiter struct {
	rllock      sync.Mutex
	fdConsuming int
	fdLimit     int
	waitingOnFd []*dialJob

	dialFunc func(context.Context, peer.ID, ma.Multiaddr) (iconn.Conn, error)

	activePerPeer      map[peer.ID]int
	perPeerLimit       int
	waitingOnPeerLimit map[peer.ID][]*dialJob
}

type dialfunc func(context.Context, peer.ID, ma.Multiaddr) (iconn.Conn, error)

func newDialLimiter(df dialfunc) *dialLimiter {
	return newDialLimiterWithParams(df, concurrentFdDials, defaultPerPeerRateLimit)
}

func newDialLimiterWithParams(df dialfunc, fdLimit, perPeerLimit int) *dialLimiter {
	return &dialLimiter{
		fdLimit:            fdLimit,
		perPeerLimit:       perPeerLimit,
		waitingOnPeerLimit: make(map[peer.ID][]*dialJob),
		activePerPeer:      make(map[peer.ID]int),
		dialFunc:           df,
	}
}

func (dl *dialLimiter) finishedDial(dj *dialJob) {
	dl.rllock.Lock()
	defer dl.rllock.Unlock()

	if addrutil.IsFDCostlyTransport(dj.addr) {
		dl.fdConsuming--

		if len(dl.waitingOnFd) > 0 {
			next := dl.waitingOnFd[0]
			dl.waitingOnFd = dl.waitingOnFd[1:]
			if len(dl.waitingOnFd) == 0 {
				dl.waitingOnFd = nil // clear out memory
			}
			dl.fdConsuming++

			go dl.executeDial(next)
		}
	}

	// release tokens in reverse order than we take them
	dl.activePerPeer[dj.peer]--
	if dl.activePerPeer[dj.peer] == 0 {
		delete(dl.activePerPeer, dj.peer)
	}

	waitlist := dl.waitingOnPeerLimit[dj.peer]
	if !dj.success && len(waitlist) > 0 {
		next := waitlist[0]

		if len(waitlist) == 1 {
			delete(dl.waitingOnPeerLimit, dj.peer)
		} else {
			dl.waitingOnPeerLimit[dj.peer] = waitlist[1:]
		}
		dl.activePerPeer[dj.peer]++ // just kidding, we still want this token

		if addrutil.IsFDCostlyTransport(next.addr) {
			if dl.fdConsuming >= dl.fdLimit {
				dl.waitingOnFd = append(dl.waitingOnFd, next)
				return
			}

			// take token
			dl.fdConsuming++
		}

		// can kick this off right here, dials in this list already
		// have the other tokens needed
		go dl.executeDial(next)
	}
}

// AddDialJob tries to take the needed tokens for starting the given dial job.
// If it acquires all needed tokens, it immediately starts the dial, otherwise
// it will put it on the waitlist for the requested token.
func (dl *dialLimiter) AddDialJob(dj *dialJob) {
	dl.rllock.Lock()
	defer dl.rllock.Unlock()

	if dl.activePerPeer[dj.peer] >= dl.perPeerLimit {
		wlist := dl.waitingOnPeerLimit[dj.peer]
		dl.waitingOnPeerLimit[dj.peer] = append(wlist, dj)
		return
	}
	dl.activePerPeer[dj.peer]++

	if addrutil.IsFDCostlyTransport(dj.addr) {
		if dl.fdConsuming >= dl.fdLimit {
			dl.waitingOnFd = append(dl.waitingOnFd, dj)
			return
		}

		// take token
		dl.fdConsuming++
	}

	// take second needed token and start dial!
	go dl.executeDial(dj)
}

// executeDial calls the dialFunc, and reports the result through the response
// channel when finished. Once the response is sent it also releases all tokens
// it held during the dial.
func (dl *dialLimiter) executeDial(j *dialJob) {
	defer dl.finishedDial(j)
	if j.cancelled() {
		return
	}

	con, err := dl.dialFunc(j.ctx, j.peer, j.addr)
	select {
	case j.resp <- dialResult{Conn: con, Addr: j.addr, Err: err}:
	case <-j.ctx.Done():
	}
}
