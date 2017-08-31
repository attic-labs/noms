package relay

import (
	"fmt"
	"net"

	pb "gx/ipfs/QmYBGpZ3hV89RZTd3CTjjgjgcX64QnbcBS9dCqU7mCgS1n/go-libp2p-circuit/pb"

	filter "gx/ipfs/QmQBB2dQLmQHJgs2gqZ3iqL2XiuCtUCvXzWt5kMXDf5Zcr/go-maddr-filter"
	tpt "gx/ipfs/QmQVm7pWYKPStMeMrXNRpvAJE5rSm9ThtQoNmjNHC7sh3k/go-libp2p-transport"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

var _ tpt.Listener = (*RelayListener)(nil)

type RelayListener Relay

func (l *RelayListener) Relay() *Relay {
	return (*Relay)(l)
}

func (r *Relay) Listener() *RelayListener {
	return (*RelayListener)(r)
}

func (l *RelayListener) Accept() (tpt.Conn, error) {
	select {
	case c := <-l.incoming:
		err := l.Relay().writeResponse(c.Stream, pb.CircuitRelay_SUCCESS)
		if err != nil {
			log.Debugf("error writing relay response: %s", err.Error())
			// this won't prevent the other side from continuing to write
			// TODO fully close the stream when Reset is implemented
			c.Stream.Close()
			return nil, err
		}

		log.Infof("accepted relay connection: %s", c.ID())

		return c, nil
	case <-l.ctx.Done():
		return nil, l.ctx.Err()
	}
}

func (l *RelayListener) Addr() net.Addr {
	return &NetAddr{
		Relay:  "any",
		Remote: "any",
	}
}

func (l *RelayListener) Multiaddr() ma.Multiaddr {
	a, err := ma.NewMultiaddr(fmt.Sprintf("/p2p-circuit/ipfs/%s", l.self.Pretty()))
	if err != nil {
		panic(err)
	}
	return a
}

func (l *RelayListener) LocalPeer() peer.ID {
	return l.self
}

func (l *RelayListener) SetAddrFilters(f *filter.Filters) {
	// noop ?
}

func (l *RelayListener) Close() error {
	// TODO: noop?
	return nil
}
