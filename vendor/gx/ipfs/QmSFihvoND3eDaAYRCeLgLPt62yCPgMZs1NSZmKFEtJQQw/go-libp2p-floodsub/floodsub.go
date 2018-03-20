package floodsub

import (
	"context"

	pb "gx/ipfs/QmSFihvoND3eDaAYRCeLgLPt62yCPgMZs1NSZmKFEtJQQw/go-libp2p-floodsub/pb"

	host "gx/ipfs/QmNmJZL7FQySMtE2BQuLMuZg2EB2CLEunJJUSVSc9YnnbV/go-libp2p-host"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

const (
	FloodSubID = protocol.ID("/floodsub/1.0.0")
)

// NewFloodSub returns a new PubSub object using the FloodSubRouter
func NewFloodSub(ctx context.Context, h host.Host, opts ...Option) (*PubSub, error) {
	rt := &FloodSubRouter{}
	return NewPubSub(ctx, h, rt, opts...)
}

type FloodSubRouter struct {
	p *PubSub
}

func (fs *FloodSubRouter) Protocols() []protocol.ID {
	return []protocol.ID{FloodSubID}
}

func (fs *FloodSubRouter) Attach(p *PubSub) {
	fs.p = p
}

func (fs *FloodSubRouter) AddPeer(peer.ID, protocol.ID) {}

func (fs *FloodSubRouter) RemovePeer(peer.ID) {}

func (fs *FloodSubRouter) HandleRPC(rpc *RPC) {}

func (fs *FloodSubRouter) Publish(from peer.ID, msg *pb.Message) {
	tosend := make(map[peer.ID]struct{})
	for _, topic := range msg.GetTopicIDs() {
		tmap, ok := fs.p.topics[topic]
		if !ok {
			continue
		}

		for p := range tmap {
			tosend[p] = struct{}{}
		}
	}

	out := rpcWithMessages(msg)
	for pid := range tosend {
		if pid == from || pid == peer.ID(msg.GetFrom()) {
			continue
		}

		mch, ok := fs.p.peers[pid]
		if !ok {
			continue
		}

		select {
		case mch <- out:
		default:
			log.Infof("dropping message to peer %s: queue full", pid)
			// Drop it. The peer is too slow.
		}
	}
}
