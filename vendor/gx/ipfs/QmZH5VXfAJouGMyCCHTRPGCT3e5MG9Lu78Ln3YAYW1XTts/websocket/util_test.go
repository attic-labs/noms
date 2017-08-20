// Copyright 2014 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"net/http"
	"reflect"
	"testing"
)

var tokenListContainsValueTests = []struct {
	value string
	ok    bool
}{
	{"WebSocket", true},
	{"WEBSOCKET", true},
	{"websocket", true},
	{"websockets", false},
	{"x websocket", false},
	{"websocket x", false},
	{"other,websocket,more", true},
	{"other, websocket, more", true},
}

func TestTokenListContainsValue(t *testing.T) {
	for _, tt := range tokenListContainsValueTests {
		h := http.Header{"Upgrade": {tt.value}}
		ok := tokenListContainsValue(h, "Upgrade", "websocket")
		if ok != tt.ok {
			t.Errorf("tokenListContainsValue(h, n, %q) = %v, want %v", tt.value, ok, tt.ok)
		}
	}
}

var parseExtensionTests = []struct {
	value      string
	extensions []map[string]string
}{
	{`foo`, []map[string]string{map[string]string{"": "foo"}}},
	{`foo, bar; baz=2`, []map[string]string{
		map[string]string{"": "foo"},
		map[string]string{"": "bar", "baz": "2"}}},
	{`foo; bar="b,a;z"`, []map[string]string{
		map[string]string{"": "foo", "bar": "b,a;z"}}},
	{`foo , bar; baz = 2`, []map[string]string{
		map[string]string{"": "foo"},
		map[string]string{"": "bar", "baz": "2"}}},
	{`foo, bar; baz=2 junk`, []map[string]string{
		map[string]string{"": "foo"}}},
	{`foo junk, bar; baz=2 junk`, nil},
	{`mux; max-channels=4; flow-control, deflate-stream`, []map[string]string{
		map[string]string{"": "mux", "max-channels": "4", "flow-control": ""},
		map[string]string{"": "deflate-stream"}}},
	{`permessage-foo; x="10"`, []map[string]string{
		map[string]string{"": "permessage-foo", "x": "10"}}},
	{`permessage-foo; use_y, permessage-foo`, []map[string]string{
		map[string]string{"": "permessage-foo", "use_y": ""},
		map[string]string{"": "permessage-foo"}}},
	{`permessage-deflate; client_max_window_bits; server_max_window_bits=10 , permessage-deflate; client_max_window_bits`, []map[string]string{
		map[string]string{"": "permessage-deflate", "client_max_window_bits": "", "server_max_window_bits": "10"},
		map[string]string{"": "permessage-deflate", "client_max_window_bits": ""}}},
}

func TestParseExtensions(t *testing.T) {
	for _, tt := range parseExtensionTests {
		h := http.Header{http.CanonicalHeaderKey("Sec-WebSocket-Extensions"): {tt.value}}
		extensions := parseExtensions(h)
		if !reflect.DeepEqual(extensions, tt.extensions) {
			t.Errorf("parseExtensions(%q)\n    = %v,\nwant %v", tt.value, extensions, tt.extensions)
		}
	}
}
