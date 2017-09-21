**Decentralized Use Case:** [About](about.md)&nbsp; | &nbsp;[Quickstart](quickstart.md)&nbsp; | &nbsp;[Architectures](architectures.md)&nbsp; | &nbsp;[P2P Chat Demo](demo-p2p-chat.md)&nbsp; | &nbsp;[IPFS Chat Demo](demo-ipfs-chat.md)
<br><br>
[![Build Status](http://jenkins3.noms.io/buildStatus/icon?job=NomsMasterBuilder)](http://jenkins3.noms.io/job/NomsMasterBuilder/)
[![codecov](https://codecov.io/gh/attic-labs/noms/branch/master/graph/badge.svg)](https://codecov.io/gh/attic-labs/noms)
[![GoDoc](https://godoc.org/github.com/attic-labs/noms?status.svg)](https://godoc.org/github.com/attic-labs/noms)
[![Slack](http://slack.noms.io/badge.svg)](http://slack.noms.io)

# Demo App: P2P Decentralized Chat

This sample demonstrates the simplest possible case of building a p2p app on top of Noms. Each node stores a complete copy of the data it is interested in, and peers find each other using [IPFS pubsub](https://ipfs.io/blog/25-pubsub/).

Currently, nodes have to have a publicly routable IP, but it should be possible to use [libP2P](https://github.com/libp2p) or similar to connect to most nodes.

# Build and Run

Demo app code is in the
[p2p](https://github.com/attic-labs/noms/tree/master/samples/go/decent/p2p-chat)
directory. To get it up and running take the following steps:
* Use git to clone the noms repository onto your computer:
```
go get github.com/attic-labs/noms/samples/go/decent/p2p-chat
```
* From the noms/samples/go/decent/p2p-chat directory, build the program with the following command:
```
go build
```
* Run the p2p client with the following command:
```
mkdir /tmp/noms1
./p2p client --username=<aname1> --node-idx=2 /tmp/noms1 >& /tmp/err1
```
* Run a second p2p client with the following command:
```
mkdir /tmp/noms2
./p2p client --username=<aname2> --node-idx=3 /tmp/noms2 >& /tmp/err2
```
  
Note: the p2p client relies on IPFS for it's pub/sub implementation. The
'node-idx' argument ensures that each IPFS-based node uses a distinct set
of ports. This is useful when running multiple IPFS-based programs on
the same machine.
