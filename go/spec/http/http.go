package http

import (
	"fmt"
	"net/url"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/datas/remote"
	"github.com/attic-labs/noms/go/spec/lite"
)

type httpProtocol struct{}

func (h *httpProtocol) Parse(name string) (string, error) {
	u, perr := url.Parse("http:" + name)
	if perr != nil {
		return "", perr
	} else if u.Host == "" {
		return "", fmt.Errorf("%s has empty host", name)
	}
	return name, nil
}

func (h *httpProtocol) NewChunkStore(sp spec.Spec) (chunks.ChunkStore, error) {
	return remote.NewHTTPChunkStore(sp.Href(), sp.Options.Authorization), nil
}

func (h *httpProtocol) NewDatabase(sp spec.Spec) (datas.Database, error) {
	return datas.NewDatabase(sp.NewChunkStore()), nil
}

func init() {
	spec.ExternalProtocols["http"] = &httpProtocol{}
	spec.ExternalProtocols["https"] = spec.ExternalProtocols["http"]
}
