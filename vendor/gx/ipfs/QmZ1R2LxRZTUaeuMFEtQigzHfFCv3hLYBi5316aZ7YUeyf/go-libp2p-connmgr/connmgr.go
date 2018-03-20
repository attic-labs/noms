package connmgr

import (
	"context"
	"sort"
	"sync"
	"time"

	logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	inet "gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	ifconnmgr "gx/ipfs/Qmax8X1Kfahf5WfSB68EWDG3d3qyS3Sqs1v412fjPTfRwx/go-libp2p-interface-connmgr"
)

var log = logging.Logger("connmgr")

type BasicConnMgr struct {
	highWater int
	lowWater  int

	gracePeriod time.Duration

	peers     map[peer.ID]*peerInfo
	connCount int

	lk sync.Mutex

	lastTrim time.Time
}

var _ ifconnmgr.ConnManager = (*BasicConnMgr)(nil)

func NewConnManager(low, hi int, grace time.Duration) *BasicConnMgr {
	return &BasicConnMgr{
		highWater:   hi,
		lowWater:    low,
		gracePeriod: grace,
		peers:       make(map[peer.ID]*peerInfo),
	}
}

type peerInfo struct {
	tags  map[string]int
	value int

	conns map[inet.Conn]time.Time

	firstSeen time.Time
}

func (cm *BasicConnMgr) TrimOpenConns(ctx context.Context) {
	defer log.EventBegin(ctx, "connCleanup").Done()
	for _, c := range cm.getConnsToClose(ctx) {
		log.Info("closing conn: ", c.RemotePeer())
		log.Event(ctx, "closeConn", c.RemotePeer())
		c.Close()
	}
}

func (cm *BasicConnMgr) getConnsToClose(ctx context.Context) []inet.Conn {
	cm.lk.Lock()
	defer cm.lk.Unlock()
	if cm.lowWater == 0 || cm.highWater == 0 {
		// disabled
		return nil
	}
	now := time.Now()
	cm.lastTrim = now

	if len(cm.peers) < cm.lowWater {
		log.Info("open connection count below limit")
		return nil
	}

	var infos []*peerInfo

	for _, inf := range cm.peers {
		infos = append(infos, inf)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].value < infos[j].value
	})

	closeCount := len(infos) - cm.lowWater
	toclose := infos[:closeCount]

	// 2x number of peers we're disconnecting from because we may have more
	// than one connection per peer. Slightly over allocating isn't an issue
	// as this is a very short-lived array.
	closed := make([]inet.Conn, 0, len(toclose)*2)

	for _, inf := range toclose {
		// TODO: should we be using firstSeen or the time associated with the connection itself?
		if inf.firstSeen.Add(cm.gracePeriod).After(now) {
			continue
		}

		// TODO: if a peer has more than one connection, maybe only close one?
		for c := range inf.conns {
			// TODO: probably don't want to always do this in a goroutine
			closed = append(closed, c)
		}
	}

	return closed
}

func (cm *BasicConnMgr) GetTagInfo(p peer.ID) *ifconnmgr.TagInfo {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	pi, ok := cm.peers[p]
	if !ok {
		return nil
	}

	out := &ifconnmgr.TagInfo{
		FirstSeen: pi.firstSeen,
		Value:     pi.value,
		Tags:      make(map[string]int),
		Conns:     make(map[string]time.Time),
	}

	for t, v := range pi.tags {
		out.Tags[t] = v
	}
	for c, t := range pi.conns {
		out.Conns[c.RemoteMultiaddr().String()] = t
	}

	return out
}

func (cm *BasicConnMgr) TagPeer(p peer.ID, tag string, val int) {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	pi, ok := cm.peers[p]
	if !ok {
		log.Info("tried to tag conn from untracked peer: ", p)
		return
	}

	pi.value += (val - pi.tags[tag])
	pi.tags[tag] = val
}

func (cm *BasicConnMgr) UntagPeer(p peer.ID, tag string) {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	pi, ok := cm.peers[p]
	if !ok {
		log.Info("tried to remove tag from untracked peer: ", p)
		return
	}

	pi.value -= pi.tags[tag]
	delete(pi.tags, tag)
}

type CMInfo struct {
	LowWater    int
	HighWater   int
	LastTrim    time.Time
	GracePeriod time.Duration
	ConnCount   int
}

func (cm *BasicConnMgr) GetInfo() CMInfo {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	return CMInfo{
		HighWater:   cm.highWater,
		LowWater:    cm.lowWater,
		LastTrim:    cm.lastTrim,
		GracePeriod: cm.gracePeriod,
		ConnCount:   cm.connCount,
	}
}

func (cm *BasicConnMgr) Notifee() inet.Notifiee {
	return (*cmNotifee)(cm)
}

type cmNotifee BasicConnMgr

func (nn *cmNotifee) cm() *BasicConnMgr {
	return (*BasicConnMgr)(nn)
}

func (nn *cmNotifee) Connected(n inet.Network, c inet.Conn) {
	cm := nn.cm()

	cm.lk.Lock()
	defer cm.lk.Unlock()

	pinfo, ok := cm.peers[c.RemotePeer()]
	if !ok {
		pinfo = &peerInfo{
			firstSeen: time.Now(),
			tags:      make(map[string]int),
			conns:     make(map[inet.Conn]time.Time),
		}
		cm.peers[c.RemotePeer()] = pinfo
	}

	_, ok = pinfo.conns[c]
	if ok {
		log.Error("received connected notification for conn we are already tracking: ", c.RemotePeer())
		return
	}

	pinfo.conns[c] = time.Now()
	cm.connCount++

	if cm.connCount > nn.highWater {
		if cm.lastTrim.IsZero() || time.Since(cm.lastTrim) > time.Second*10 {
			go cm.TrimOpenConns(context.Background())
		}
	}
}

func (nn *cmNotifee) Disconnected(n inet.Network, c inet.Conn) {
	cm := nn.cm()

	cm.lk.Lock()
	defer cm.lk.Unlock()

	cinf, ok := cm.peers[c.RemotePeer()]
	if !ok {
		log.Error("received disconnected notification for peer we are not tracking: ", c.RemotePeer())
		return
	}

	_, ok = cinf.conns[c]
	if !ok {
		log.Error("received disconnected notification for conn we are not tracking: ", c.RemotePeer())
		return
	}

	delete(cinf.conns, c)
	cm.connCount--
	if len(cinf.conns) == 0 {
		delete(cm.peers, c.RemotePeer())
	}
}

func (nn *cmNotifee) Listen(n inet.Network, addr ma.Multiaddr)      {}
func (nn *cmNotifee) ListenClose(n inet.Network, addr ma.Multiaddr) {}
func (nn *cmNotifee) OpenedStream(inet.Network, inet.Stream)        {}
func (nn *cmNotifee) ClosedStream(inet.Network, inet.Stream)        {}
