package ipfs

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/ipfs/go-ipfs/core"
	"github.com/pkg/errors"
)

const (
	// NetworkProtoName is the default protocol name for networked IPFS ChunkStores. SetLocal for more information
	NetworkProtoName = "ipfs"

	// LocalProtoName is the default protocol name for local IPFS ChunkStores. See SetLocal for more information.
	// information.
	LocalProtoName = "ipfs-local"
)

// Implementing HasIPFSNode indicates that a type has an underlying IPFS node. It is implemented by the ChunkStore and
// Database returned by Protocol (i.e. all ChunkStores and Databases created via a Spec) and
// NewChunkStore/ChunkStoreFromIPFSNode
type HasIPFSNode interface {
	IPFSNode() *core.IpfsNode
}

// Protocol implements spec.ProtocolImpl for the IPFS ChunkStore
type Protocol struct {
	*config
}

// NewProtocol returns a new Protocol with the given options. The protocol defaults to port index 0, and a maximum
// concurrent request count of 1. See documentation under Option for more information on options
func NewProtocol(opts ...Option) (*Protocol, error) {
	cfg, err := cfgFrom(opts...)
	if err != nil {
		return nil, err
	}

	p := &Protocol{cfg}

	return p, nil
}

// NewChunkStore returns a new ChunkStore backed by IPFS using the options passed in via NewProtocol
// The returned ChunkStore implements HasIPFSNode.
//
// See the package-level NewChunkStore and Options for more information.
func (p Protocol) NewChunkStore(sp spec.Spec) (chunks.ChunkStore, error) {
	if sp.DatabaseName == "" {
		return nil, errors.New("no database in spec")
	}

	return newChunkStore(sp.DatabaseName, p.config)
}

// NewDatabase returns a new Database backed by an IPFS ChunkStore. The returned database implements HasIPFSNode.
//
// See the package-level NewChunkStore and Options for more information.
func (p Protocol) NewDatabase(sp spec.Spec) (datas.Database, error) {
	if sp.DatabaseName == "" {
		return nil, errors.New("no database in spec")
	}
	cs, err := p.NewChunkStore(sp)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ChunkStore")
	}
	return newIPFSDb(cs), nil
}

type ipfsDb struct {
	// NOTE: if datas.Database had a public ChunkStore() method, this'd be easier
	datas.Database
	node *core.IpfsNode
}

func newIPFSDb(cs chunks.ChunkStore) ipfsDb {
	return ipfsDb{Database: datas.NewDatabase(cs), node: cs.(HasIPFSNode).IPFSNode()}
}

func (db ipfsDb) IPFSNode() *core.IpfsNode {
	return db.node
}

// RegisterProtocols registers the "ipfs" and "ipfs-local" protocols as external protocols for Specs, with the given
// options applied to both local and networked protocols. The protocols default to port index 0 and a maximum concurrent
// request count of 1. SetLocal and SetNetworked are ignored.
//
// The external protocol is implemented by the Protocol type.
func RegisterProtocols(opts ...Option) error {
	if err := register(NetworkProtoName, false, opts...); err != nil {
		return err
	}

	if err := register(LocalProtoName, true, opts...); err != nil {
		return err
	}
	return nil
}

// UnregisterProtocols unregisters the "ipfs" and "ipfs-local" protocols as external protocols for Specs.
func UnregisterProtocols() {
	spec.UnregisterExternalProtocol(NetworkProtoName)
	spec.UnregisterExternalProtocol(LocalProtoName)
}

func register(name string, local bool, opts ...Option) error {
	p, err := NewProtocol(opts...)
	p.config.local = local

	if err != nil {
		return errors.Wrapf(err, "error creating protocol '%s'", name)
	}

	err = spec.RegisterExternalProtocol(name, p)

	return errors.Wrapf(err, "error registering protocol '%s'")
}
