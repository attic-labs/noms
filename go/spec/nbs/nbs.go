package nbs

import (
	"os"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/spec/lite"
)

type nbsProtocol struct{}

func (n *nbsProtocol) NewChunkStore(sp spec.Spec) (chunks.ChunkStore, error) {
	os.MkdirAll(sp.DatabaseName, 0777)
	return nbs.NewLocalStore(sp.DatabaseName, 1<<28), nil
}

func (n *nbsProtocol) NewDatabase(sp spec.Spec) (datas.Database, error) {
	return datas.NewDatabase(sp.NewChunkStore()), nil
}

func init() {
	spec.ExternalProtocols["nbs"] = &nbsProtocol{}
}
