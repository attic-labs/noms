package swarm

import (
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	addrutil "gx/ipfs/Qmcx4KoZ91XS6izLjdkujAMAunAS1YuTgfgASgYaZF5GkR/go-addr-util"
	iconn "gx/ipfs/QmfQAY7YU4fQi3sjGLs1hwkM2Aq7dxgDyoMjaKN4WBWvcB/go-libp2p-interface-conn"
)

// ListenAddresses returns a list of addresses at which this swarm listens.
func (s *Swarm) ListenAddresses() []ma.Multiaddr {
	listeners := s.swarm.Listeners()
	addrs := make([]ma.Multiaddr, 0, len(listeners))
	for _, l := range listeners {
		if l2, ok := l.NetListener().(iconn.Listener); ok {
			addrs = append(addrs, l2.Multiaddr())
		}
	}
	return addrs
}

// InterfaceListenAddresses returns a list of addresses at which this swarm
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func (s *Swarm) InterfaceListenAddresses() ([]ma.Multiaddr, error) {
	return addrutil.ResolveUnspecifiedAddresses(s.ListenAddresses(), nil)
}
