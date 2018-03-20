package spec

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
)

// ProtocolImpl is the interface that external protocols should implement.
type ProtocolImpl interface {
	NewChunkStore(sp Spec) (chunks.ChunkStore, error)
	NewDatabase(sp Spec) (datas.Database, error)
}

// RegisterExternalProtocol registers an external protocol implementation with the given name and ProtocolImpl.
// Trying to register a protocol with a name that is already taken will return an error.
// Thread-safe.
func RegisterExternalProtocol(name string, p ProtocolImpl) error {
	return externalProtocols.set(name, p)
}

// UnregisterExternalProtocol will remove the protocol handler for the protocol with the given name. It will return
// true if there was a protocol with that name, false otherwise.
// Thread-safe
func UnregisterExternalProtocol(name string) bool {
	return externalProtocols.remove(name)
}

var externalProtocols protoHolder

// protoHolder handles thread-safe external protocol set/get/remove
type protoHolder struct {
	protos map[string]ProtocolImpl
	sync.RWMutex
	sync.Once
}

func (ph *protoHolder) init() {
	ph.Do(func() {
		ph.Lock()
		ph.protos = make(map[string]ProtocolImpl)
		ph.Unlock()
	})
}

func (ph *protoHolder) get(name string) (p ProtocolImpl, ok bool) {
	ph.RLock()
	p, ok = ph.protos[name]
	ph.RUnlock()
	return
}

func (ph *protoHolder) set(name string, pi ProtocolImpl) error {
	// TODO: check that name isn't one of the hard-coded ones
	ph.init()
	ph.Lock()
	defer ph.Unlock()
	if _, ok := ph.protos[name]; ok {
		return fmt.Errorf("external protocol '%s' already defined", name)
	}
	ph.protos[name] = pi
	return nil
}

func (ph *protoHolder) remove(name string) bool {
	ph.Lock()
	defer ph.Unlock()
	if _, found := ph.protos[name]; !found {
		return false
	}
	delete(ph.protos, name)
	return true
}
