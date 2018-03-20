package record

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	u "gx/ipfs/QmNiJuT8Ja3hMVpBHXv3Q6dwmperaQ6JjLtpMQgMCD7xvx/go-ipfs-util"
	logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	pb "gx/ipfs/QmUpttFinNDmNPgFwKN8sZK6BUtBmA68Y4KdSBDXa8t9sJ/go-libp2p-record/pb"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

var log = logging.Logger("routing/record")

// ErrBadRecord is returned any time a dht record is found to be
// incorrectly formatted or signed.
var ErrBadRecord = errors.New("bad dht record")

// ErrInvalidRecordType is returned if a DHTRecord keys prefix
// is not found in the Validator map of the DHT.
var ErrInvalidRecordType = errors.New("invalid record keytype")

type ValidationRecord struct {
	Namespace string
	Key       string
	Value     []byte
	// Note: author is only present if the source record is signed
	// Otherwise it will be ""
	Author peer.ID
}

// ValidatorFunc is a function that is called to validate a given
// type of DHTRecord.
type ValidatorFunc func(*ValidationRecord) error

// Validator is an object that helps ensure routing records are valid.
// It is a collection of validator functions, each of which implements
// its own notion of validity.
type Validator map[string]*ValidChecker

type ValidChecker struct {
	Func ValidatorFunc
	Sign bool
}

func splitPath(key string) (string, string, error) {
	if len(key) == 0 || key[0] != '/' {
		return "", "", ErrInvalidRecordType
	}

	key = key[1:]

	i := strings.IndexByte(key, '/')
	if i <= 0 {
		return "", "", ErrInvalidRecordType
	}

	return key[:i], key[i+1:], nil
}

func parseRecord(r *pb.Record) (*ValidationRecord, error) {
	namespace, key, err := splitPath(r.GetKey())
	if err != nil {
		return nil, err
	}

	// Note that the caller is responsible for verifying the
	// signature
	author := peer.ID("")
	if len(r.GetSignature()) > 0 {
		pid, err := peer.IDFromString(r.GetAuthor())
		if err != nil {
			log.Warningf("Could not parse author to peer ID: %s", r.GetAuthor())
			return nil, ErrInvalidRecordType
		}
		author = pid
	}
	return &ValidationRecord{
		Namespace: namespace,
		Key:       key,
		Value:     r.GetValue(),
		Author:    author,
	}, nil
}

// VerifyRecord checks a record and ensures it is still valid.
// It runs needed validators.
// Note that VerifyRecord does not perform signature verification,
// the signature must be verified by the caller.
func (v Validator) VerifyRecord(r *pb.Record) error {
	vr, err := parseRecord(r)
	if err != nil {
		return err
	}
	val, ok := v[vr.Namespace]
	if !ok {
		log.Infof("Unrecognized key prefix: %s", vr.Namespace)
		return ErrInvalidRecordType
	}
	return val.Func(vr)
}

func (v Validator) IsSigned(k string) (bool, error) {
	namespace, _, err := splitPath(k)
	if err != nil {
		return false, err
	}

	val, ok := v[namespace]
	if !ok {
		log.Infof("Unrecognized key prefix: %s", namespace)
		return false, ErrInvalidRecordType
	}

	return val.Sign, nil
}

// ValidatePublicKeyRecord implements ValidatorFunc and
// verifies that the passed in record value is the PublicKey
// that matches the passed in key.
func ValidatePublicKeyRecord(r *ValidationRecord) error {
	if r.Namespace != "pk" {
		return errors.New("namespace not 'pk'")
	}

	keyhash := []byte(r.Key)
	if _, err := mh.Cast(keyhash); err != nil {
		return fmt.Errorf("key did not contain valid multihash: %s", err)
	}

	pkh := u.Hash(r.Value)
	if !bytes.Equal(keyhash, pkh) {
		return errors.New("public key does not match storage key")
	}
	return nil
}

var PublicKeyValidator = &ValidChecker{
	Func: ValidatePublicKeyRecord,
	Sign: false,
}

func CheckRecordSig(r *pb.Record, pk ci.PubKey) error {
	blob := RecordBlobForSig(r)
	if good, err := pk.Verify(blob, r.Signature); err != nil || !good {
		return errors.New("invalid record signature")
	}
	return nil
}
