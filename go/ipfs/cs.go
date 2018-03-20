package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	"gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"

	"gx/ipfs/QmPwNSAKhfSDEjQ2LYx8bemvnoyXYTaL96JxsAvjzphT75/go-ipfs-blockstore"
	"gx/ipfs/QmXporsyf5xMvffd2eiTDoq85dNpYUynGJhfabzDjwP8uR/go-ipfs/core"
	ipfsCfg "gx/ipfs/QmXporsyf5xMvffd2eiTDoq85dNpYUynGJhfabzDjwP8uR/go-ipfs/repo/config"
	"gx/ipfs/QmXporsyf5xMvffd2eiTDoq85dNpYUynGJhfabzDjwP8uR/go-ipfs/repo/fsrepo"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/samples/go/decent/dbg"
	"github.com/pkg/errors"
)

// Default ports for IPFS
const (
	DefaultAPIPort     = 5001
	DefaultGatewayPort = 8080
	DefaultSwarmPort   = 4001
)

// NewChunkStore creates a new ChunkStore backed by IPFS.
//
// Noms chunks written to this ChunkStore are converted to IPFS blocks and
// stored in an IPFS BlockStore.
//
// IPFS database specs have the form:
//   ipfs://<path-to-ipfs-dir> for networked ChunkStores
//   ipfs-local://<path-to-ipfs-dir> for local ChunkStores
// where 'ipfs' or 'ipfs-local' indicates the noms protocol and the path indicates the path to the directory where the
// ipfs repo resides. The chunkstore creates two files in the ipfs directory called 'noms' and 'noms-local' which stores
// the root of the noms database. This should ideally be done with IPNS, but that is currently too slow to be practical.
//
// This function creates an IPFS repo at the appropriate path if one doesn't already exist.
//
// See Option documentation for more information on options.
func NewChunkStore(dbPath string, opts ...Option) (chunks.ChunkStore, error) {
	cfg, err := cfgFrom(opts...)
	if err != nil {
		return nil, err
	}

	return newChunkStore(dbPath, cfg)
}

func newChunkStore(dbPath string, c *config) (chunks.ChunkStore, error) {
	dbg.Debug("Creating new chunk store at %s, config %+v", dbPath, c)
	node, err := OpenIPFSRepo(dbPath, c.portIdx)
	if err != nil {
		return nil, errors.Wrap(err, "error opening IPFS repo")
	}

	return &chunkStore{
		node:        node,
		name:        dbPath,
		c:           *c,
		concLimiter: make(chan struct{}, c.maxConcurrent),
	}, nil
}

// OpenIPFSRepo opens an IPFS repo for use as a noms store, and returns an IPFS node for that repo. Creates a new repo
// at this indicated path if one doesn't already exist. See SetPortIdx for information on portIdx.
func OpenIPFSRepo(path string, portIdx int) (*core.IpfsNode, error) {
	r, err := fsrepo.Open(path)
	dbg.Debug("opening IPFS repo with idx %d at %s", portIdx, path)
	if _, ok := err.(fsrepo.NoRepoError); ok {
		var conf *ipfsCfg.Config
		conf, err = ipfsCfg.Init(os.Stdout, 2048)
		if err != nil {
			return nil, errors.Wrap(err, "error initializing new IPFS config")
		}

		err = fsrepo.Init(path, conf)
		if err != nil {
			return nil, errors.Wrap(err, "error initializing new IPFS repo")
		}

		r, err = fsrepo.Open(path)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error opening IPFS repo")
	}

	resetRepoConfigPorts(r, portIdx)

	cfg := &core.BuildCfg{
		Repo:   r,
		Online: true,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}

	node, err := core.NewNode(context.Background(), cfg)

	if err != nil {
		return nil, errors.Wrap(err, "error creating IPFS node")
	}

	repoCfg, err := node.Repo.Config()

	if err != nil {
		return nil, errors.Wrap(err, "error getting IPFS repo config")
	}

	dbg.Debug("Addresses are %+v", repoCfg.Addresses)

	return node, nil
}

type chunkStore struct {
	root        *hash.Hash
	node        *core.IpfsNode
	name        string
	concLimiter chan struct{}
	c           config
}

func (cs *chunkStore) IPFSNode() *core.IpfsNode {
	return cs.node
}

func (cs *chunkStore) limitConcurrency() func() {
	cs.concLimiter <- struct{}{}
	return func() {
		<-cs.concLimiter
	}
}

func (cs *chunkStore) getBlock(chunkId *cid.Cid, timeout time.Duration) (b blocks.Block, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if cs.c.local {
		b, err = cs.node.Blockstore.Get(chunkId)
		if err == blockstore.ErrNotFound {
			return
		}
	} else {
		b, err = cs.node.Blocks.GetBlock(ctx, chunkId)
	}
	return
}

func (cs *chunkStore) get(h hash.Hash, timeout time.Duration) chunks.Chunk {
	dbg.Debug("starting ipfsCS Get, h: %s, cid: %s, cs.c.local: %t", h, NomsHashToCID(h), cs.c.local)
	var b blocks.Block
	defer cs.limitConcurrency()()

	chunkId := NomsHashToCID(h)
	b, err := cs.getBlock(chunkId, timeout)
	if err == nil {
		dbg.Debug("finished ipfsCS Get, h: %s, cid: %s, cs.c.local: %t, len(b.RawData): %d", h, NomsHashToCID(h), cs.c.local, len(b.RawData()))
		return chunks.NewChunkWithHash(h, b.RawData())
	}
	dbg.Debug("ipfsCS Get, EmptyChunk for h: %s, cid: %s, err: %s, b: %v", h, NomsHashToCID(h), err, b)
	return chunks.EmptyChunk
}

func (cs *chunkStore) Get(h hash.Hash) chunks.Chunk {
	return cs.get(h, time.Second*200)
}

func (cs *chunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	defer dbg.BoxF("ipfs chunkstore GetMany, cs.c.local: %t", cs.c.local)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cs.limitConcurrency()()

	cids := make([]*cid.Cid, 0, len(hashes))
	for h := range hashes {
		c := NomsHashToCID(h)
		cids = append(cids, c)
	}

	if cs.c.local {
		for _, cid := range cids {
			b, err := cs.node.Blockstore.Get(cid)
			d.PanicIfError(err)
			c := chunks.NewChunkWithHash(CidToNomsHash(b.Cid()), b.RawData())
			foundChunks <- &c
		}
	} else {
		for b := range cs.node.Blocks.GetBlocks(ctx, cids) {
			c := chunks.NewChunkWithHash(CidToNomsHash(b.Cid()), b.RawData())
			foundChunks <- &c
		}
	}
}

func (cs *chunkStore) Has(h hash.Hash) bool {
	id := NomsHashToCID(h)
	if cs.c.local {
		defer cs.limitConcurrency()()
		ok, err := cs.node.Blockstore.Has(id)
		d.PanicIfError(err)
		return ok
	} else {
		// BlockService doesn't have Has(), neither does underlying Exchange()
		c := cs.get(h, time.Second*5)
		return !c.IsEmpty()
	}
}

func (cs *chunkStore) HasMany(hashes hash.HashSet) hash.HashSet {
	defer dbg.BoxF("HasMany, len(hashes): %d", len(hashes))()
	misses := hash.HashSet{}
	if cs.c.local {
		for h := range hashes {
			if !cs.Has(h) {
				misses[h] = struct{}{}
			}
		}
	} else {
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(len(hashes))
		for h := range hashes {
			h := h
			go func() {
				defer wg.Done()
				ok := cs.Has(h)
				if !ok {
					mu.Lock()
					misses[h] = struct{}{}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
	}
	return misses
}

func (cs *chunkStore) Put(c chunks.Chunk) {
	defer cs.limitConcurrency()()

	cid := NomsHashToCID(c.Hash())
	b, err := blocks.NewBlockWithCid(c.Data(), cid)
	d.PanicIfError(err)
	if cs.c.local {
		err = cs.node.Blockstore.Put(b)
		d.PanicIfError(err)
	} else {
		err := cs.node.Blocks.AddBlock(b)
		d.PanicIfError(err)
		d.PanicIfFalse(reflect.DeepEqual(cid, b.Cid()))
	}
}

func (cs *chunkStore) Version() string {
	// TODO: Store this someplace in the DB root
	return "7.18"
}

func (cs *chunkStore) Rebase() {
	h := hash.Hash{}
	var sp string
	f := cs.getLocalNameFile(cs.c.local)
	b, err := ioutil.ReadFile(f)
	if !os.IsNotExist(err) {
		d.PanicIfError(err)
		sp = string(b)
	}

	if sp != "" {
		cid, err := cid.Decode(sp)
		d.PanicIfError(err)
		h = CidToNomsHash(cid)
	}
	cs.root = &h
}

func (cs *chunkStore) Root() (h hash.Hash) {
	if cs.root == nil {
		cs.Rebase()
	}
	return *cs.root
}

func CidToNomsHash(id *cid.Cid) (h hash.Hash) {
	dmh, err := mh.Decode([]byte(id.Hash()))
	d.PanicIfError(err)
	copy(h[:], dmh.Digest)
	return
}

func NomsHashToCID(nh hash.Hash) *cid.Cid {
	mhb, err := mh.Encode(nh[:], mh.SHA2_512)
	d.PanicIfError(err)
	return cid.NewCidV1(cid.Raw, mhb)
}

func (cs *chunkStore) Commit(current, last hash.Hash) bool {
	defer dbg.BoxF("chunkstore Commit")()
	// TODO: In a more realistic implementation this would flush queued chunks to storage.
	if cs.root != nil && *cs.root == current {
		fmt.Println("eep, asked to commit current value?")
		return true
	}

	// TODO: Optimistic concurrency?

	cid := NomsHashToCID(current)
	if cs.c.local {
		err := ioutil.WriteFile(cs.getLocalNameFile(true), []byte(cid.String()), 0644)
		d.PanicIfError(err)
	}
	err := ioutil.WriteFile(cs.getLocalNameFile(false), []byte(cid.String()), 0644)
	d.PanicIfError(err)

	cs.root = &current
	return true
}

func (cs *chunkStore) getLocalNameFile(local bool) string {
	if local {
		return path.Join(cs.name, "noms-local")
	}
	return path.Join(cs.name, "noms")
}

type IPFSStats struct {
	Local   bool
	PortIdx int
}

func (cs *chunkStore) Stats() interface{} {
	return IPFSStats{Local: cs.c.local, PortIdx: cs.c.portIdx}
}

func (cs *chunkStore) StatsSummary() string {
	return "Unsupported"
}

func (cs *chunkStore) Close() error {
	return cs.node.Close()
}
