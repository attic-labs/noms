// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package receipts

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"golang.org/x/crypto/nacl/secretbox"
)

// Data stores parsed receipt data.
type Data struct {
	Database string
	Date     time.Time
}

// KeySize is the size in bytes of receipt keys.
const KeySize = 32 // secretbox

// Key is used to encrypt receipt data.
type Key [KeySize]byte

// Don't use a nonce when sealing/opening secretbox because these receipts need
// to be used across sessions.
var emptyNonce [24]byte

// Force all receipts to have the same size, to avoid size attacks.
const receiptSize = 256

// DecodeKey converts a base64 encoded string to a receipt key.
func DecodeKey(s string) (key Key, err error) {
	keySlice, err2 := base64.URLEncoding.DecodeString(s)
	if err2 != nil {
		err = err2
		return
	}

	if len(keySlice) != len(key) {
		err = fmt.Errorf("--key must be %d bytes when decoded, not %d", len(key), len(keySlice))
		return
	}

	copy(key[:], keySlice)
	return
}

// Generate returns a receipt for Data, which is an encrypted query string
// encoded as base64.
func Generate(key Key, data Data) (string, error) {
	d.PanicIfTrue(data.Database == "" || data.Date == (time.Time{}))

	receiptPlainOrig := []byte(url.Values{
		"Database": []string{nomsHash(data.Database)},
		"Date":     []string{data.Date.Format(time.RFC3339Nano)},
		"Salt":     []string{genSalt()},
	}.Encode())
	// This should be a constant size because the database name is hashed, the
	// date format is standard, and the salt is 32 bytes. As of writing this is
	// 140 bytes - you can even tweet them. 256 bytes is plenty of room.
	d.PanicIfTrue(len(receiptPlainOrig) > receiptSize)

	var receiptPlain [receiptSize]byte
	copy(receiptPlain[:], receiptPlainOrig)

	var keyBytes [KeySize]byte = key
	receiptSealed := secretbox.Seal(nil, receiptPlain[:], &emptyNonce, &keyBytes)

	return base64.URLEncoding.EncodeToString(receiptSealed), nil
}

// Verify verifies that a generated receipt grants access to a database.
// The Date field will be populated with the date from the decrypted receipt.
//
// Returns a tuple (ok, error) where ok is true if verification succeeds and
// false if not. Error is non-nil if the receipt itself is invalid.
func Verify(key Key, receiptText string, data *Data) (bool, error) {
	d.PanicIfTrue(data.Database == "")

	receiptSealed, err := base64.URLEncoding.DecodeString(receiptText)
	if err != nil {
		return false, err
	}

	var keyBytes [KeySize]byte = key
	receiptPlain, ok := secretbox.Open(nil, receiptSealed, &emptyNonce, &keyBytes)
	if !ok {
		return false, fmt.Errorf("Failed to decrypt receipt")
	}

	query, err := url.ParseQuery(string(receiptPlain))
	if err != nil {
		return false, fmt.Errorf("Receipt is not a valid query string")
	}

	database := query.Get("Database")
	if database == "" {
		return false, fmt.Errorf("Receipt is missing a Database field")
	}

	dateString := query.Get("Date")
	if dateString == "" {
		return false, fmt.Errorf("Receipt is missing a Date field")
	}

	date, err := time.Parse(time.RFC3339Nano, dateString)
	if err != nil {
		return false, err
	}

	data.Date = date
	return nomsHash(data.Database) == database, nil
}

func nomsHash(s string) string {
	return hash.FromData([]byte(s)).String()
}

func genSalt() string {
	var salt [32]byte
	rand.Read(salt[:])
	return base64.URLEncoding.EncodeToString(salt[:])
}
