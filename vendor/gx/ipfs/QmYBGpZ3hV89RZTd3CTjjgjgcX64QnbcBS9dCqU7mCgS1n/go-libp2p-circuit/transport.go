package relay

import (
	"context"
	"fmt"

	tpt "gx/ipfs/QmQVm7pWYKPStMeMrXNRpvAJE5rSm9ThtQoNmjNHC7sh3k/go-libp2p-transport"
	swarm "gx/ipfs/QmVJefKHXEx28RFpmj5GeRg43AqeBH3npPwvgJ875fBPm7/go-libp2p-swarm"
	host "gx/ipfs/QmW8Rgju5JrSMHP7RDNdiwwXyenRqAbtSaPfdQKQC7ZdH6/go-libp2p-host"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	addrutil "gx/ipfs/Qmcx4KoZ91XS6izLjdkujAMAunAS1YuTgfgASgYaZF5GkR/go-addr-util"
)

const P_CIRCUIT = 290

var Protocol = ma.Protocol{
	Code:  P_CIRCUIT,
	Size:  0,
	Name:  "p2p-circuit",
	VCode: ma.CodeToVarint(P_CIRCUIT),
}

func init() {
	ma.AddProtocol(Protocol)

	// Add dialer transport
	const unspecific = "/p2p-circuit/ipfs"
	const proto = "/ipfs/p2p-circuit/ipfs"

	tps := addrutil.SupportedTransportStrings

	err := addrutil.AddTransport(unspecific)
	if err != nil {
		panic(err)
	}

	err = addrutil.AddTransport(proto)
	if err != nil {
		panic(err)
	}

	for _, tp := range tps {
		err = addrutil.AddTransport(tp + proto)
		if err != nil {
			panic(err)
		}
	}
}

var _ tpt.Transport = (*RelayTransport)(nil)

type RelayTransport Relay

func (t *RelayTransport) Relay() *Relay {
	return (*Relay)(t)
}

func (r *Relay) Transport() *RelayTransport {
	return (*RelayTransport)(r)
}

func (t *RelayTransport) Dialer(laddr ma.Multiaddr, opts ...tpt.DialOpt) (tpt.Dialer, error) {
	if !t.Matches(laddr) {
		return nil, fmt.Errorf("%s is not a relay address", laddr)
	}
	return t.Relay().Dialer(), nil
}

func (t *RelayTransport) Listen(laddr ma.Multiaddr) (tpt.Listener, error) {
	if !t.Matches(laddr) {
		return nil, fmt.Errorf("%s is not a relay address", laddr)
	}
	return t.Relay().Listener(), nil
}

func (t *RelayTransport) Matches(a ma.Multiaddr) bool {
	return t.Relay().Dialer().Matches(a)
}

// AddRelayTransport constructs a relay and adds it as a transport to the host network.
func AddRelayTransport(ctx context.Context, h host.Host, opts ...RelayOpt) error {
	// the necessary methods are not part of the Network interface, only exported by Swarm
	// TODO: generalize the network interface for adding tranports
	n, ok := h.Network().(*swarm.Network)
	if !ok {
		return fmt.Errorf("%v is not a swarm network", h.Network())
	}

	s := n.Swarm()

	r, err := NewRelay(ctx, h, opts...)
	if err != nil {
		return err
	}

	s.AddTransport(r.Transport())
	return s.AddListenAddr(r.Listener().Multiaddr())
}
