package ipfs

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gx/ipfs/QmXporsyf5xMvffd2eiTDoq85dNpYUynGJhfabzDjwP8uR/go-ipfs/repo"
)

// resetRepoConfigPorts adds portIdx to each default port number. The index is added instead of replaced to keep
// idx 0 and 1 from conflicting.
func resetRepoConfigPorts(r repo.Repo, portIdx int) error {
	apiPort := strconv.Itoa(DefaultAPIPort + portIdx)
	gatewayPort := strconv.Itoa(DefaultGatewayPort + portIdx)

	swarmPort := strconv.Itoa(DefaultSwarmPort + portIdx)

	rc, err := r.Config()
	if err != nil {
		return errors.Wrap(err, "error getting IPFS repo config")
	}

	rc.Addresses.API = strings.Replace(rc.Addresses.API, "5001", apiPort, -1)
	rc.Addresses.Gateway = strings.Replace(rc.Addresses.Gateway, "8080", gatewayPort, -1)
	for i, addr := range rc.Addresses.Swarm {
		rc.Addresses.Swarm[i] = strings.Replace(addr, "4001", swarmPort, -1)
	}

	return errors.Wrap(r.SetConfig(rc), "error setting IPFS repo config")
}

// An Option configures the IPFS ChunkStore
type Option interface {
	apply(*config) error
}

type optFunc func(*config) error

func (of optFunc) apply(c *config) error {
	return of(c)
}

// SetLocal makes the ChunkStore only use the local IPFS blockstore for both reads and writes.
func SetLocal() Option {
	return optFunc(func(c *config) error {
		c.local = true
		return nil
	})
}

// SetNetworked makes reads fall through to the network and expose stored blocks to the entire IPFS network.
func SetNetworked() Option {
	return optFunc(func(c *config) error {
		c.local = false
		return nil
	})
}

// SetMaxConcurrent sets the maximum number of concurrent requests used when creating IPFS ChunkStores from a Spec. The
// default is 1. Negative values of n will return an error.
func SetMaxConcurrent(max int) Option {
	return optFunc(func(config *config) error {
		if max < 0 {
			return errors.New("SetMaxConcurrent must be called with max > 0")
		}
		config.maxConcurrent = max
		return nil
	})
}

// SetPortIdx sets the port index to use when creating IPFS ChunkStores from a Spec. If portIdx is a number between 1
// and 8 inclusive, the config file will be modified to add 'portIdx' to each external port's number. The defaults are
// API: 5001, gateway: 8080, swarm: 4001, so a portIdx of 1 would give you 5002, 8081, and 4002.
//
// The default is 0, which stands for IPFS default ports. idx must be between 0 and 8 inclusive; other values will
// result in an error.
func SetPortIdx(portIdx int) Option {
	return optFunc(func(protocol *config) error {
		if portIdx < 0 || portIdx > 8 {
			return errors.New("SetPortIdx must be called with portIdx >= 0 and <= 8")
		}
		protocol.portIdx = portIdx
		return nil
	})
}

// TODO: figure out less grody way of passing options to the external protocol. Maybe even extend Spec so that it supports
// URI query strings, since it's already URI-like?

type config struct {
	portIdx       int
	maxConcurrent int
	local         bool
}

// cfgFrom turns opts into a valid config
func cfgFrom(opts ...Option) (*config, error) {
	c := &config{maxConcurrent: 1}
	for _, opt := range opts {
		if err := opt.apply(c); err != nil {
			return nil, errors.Wrap(err, "error in configuration")
		}
	}

	return c, nil
}
