package pnet

import (
	"io"

	tconn "gx/ipfs/QmQVm7pWYKPStMeMrXNRpvAJE5rSm9ThtQoNmjNHC7sh3k/go-libp2p-transport"
	ipnet "gx/ipfs/QmQq9YzmdFdWNTDdArueGyD7L5yyiRQigrRHJnTGkxcEjT/go-libp2p-interface-pnet"
)

var _ ipnet.Protector = (*protector)(nil)

// NewProtector creates ipnet.Protector instance from a io.Reader stream
// that should include Multicodec encoded V1 PSK.
func NewProtector(input io.Reader) (ipnet.Protector, error) {
	psk, err := decodeV1PSKKey(input)
	if err != nil {
		return nil, err
	}
	f := fingerprint(psk)

	return &protector{
		psk:         psk,
		fingerprint: f,
	}, nil
}

type protector struct {
	psk         *[32]byte
	fingerprint []byte
}

func (p protector) Protect(in tconn.Conn) (tconn.Conn, error) {
	return newPSKConn(p.psk, in)
}
func (p protector) Fingerprint() []byte {
	return p.fingerprint
}
