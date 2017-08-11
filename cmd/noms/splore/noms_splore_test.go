// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package splore

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/attic-labs/testify/assert"
)

func TestNomsSplore(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "TestNomsSplore")
	d.PanicIfError(err)
	defer os.RemoveAll(dir)

	getNode := func(id string) string {
		lchan := make(chan net.Listener)
		httpServe = func(l net.Listener, h http.Handler) error {
			lchan <- l
			http.Serve(l, h) // this will error because of the l.Close() below
			return nil
		}

		go func() {
			quiet := verbose.Quiet()
			defer verbose.SetQuiet(quiet)
			verbose.SetQuiet(true)
			mux = &http.ServeMux{} // reset mux each run to clear http.Handle calls
			run([]string{"nbs:" + dir})
		}()
		l := <-lchan
		defer l.Close()

		r, err := http.Get(fmt.Sprintf("http://%s/getNode?id=%s", l.Addr().String(), id))
		assert.NoError(err)
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		return string(body)
	}

	test := func(expectJSON string, id string) {
		// The dataset head hash changes whenever the test data changes, so instead
		// of updating it all the time, use string replacement.
		dsHash := sp.GetDataset().HeadRef().TargetHash().String()
		expectJSON = strings.Replace(expectJSON, "{{dsHash}}", dsHash, -1)
		assert.JSONEq(expectJSON, getNode(id))
	}

	// No data yet:
	assert.JSONEq(`{
		"children": [],
		"hasChildren": false,
		"id": "",
		"name": "Map(0)"
	}`, getNode(""))

	// Path not found:
	assert.JSONEq(`{"error": "not found"}`, getNode(".notfound"))

	// Test with real data:
	sp, err := spec.ForDataset(fmt.Sprintf("nbs:%s::ds", dir))
	d.PanicIfError(err)
	defer sp.Close()
	strct := types.NewStruct("StructName", types.StructData{
		"blob":   types.NewBlob(),
		"bool":   types.Bool(true),
		"list":   types.NewList(types.Number(1), types.Number(2)),
		"map":    types.NewMap(types.String("a"), types.String("b"), types.String("c"), types.String("d")),
		"number": types.Number(42),
		"ref":    sp.GetDatabase().WriteValue(types.Bool(true)),
		"set":    types.NewSet(types.Number(3), types.Number(4)),
		"string": types.String("hello world"),
	})
	sp.GetDatabase().CommitValue(sp.GetDataset(), strct)

	// Root => datasets:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "@at(0)@key",
				"name": "\"ds\""
			},
			"label": "",
			"value": {
				"hasChildren": true,
				"id": "@at(0)",
				"name": "Value#{{dsHash}}"
			}
		}
		],
		"hasChildren": true,
		"id": "",
		"name": "Map(1)"
	}`, "")

	// Dataset 0 (ds) => dataset head ref.
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target",
				"name": "Value#{{dsHash}}"
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)",
		"name": "Value#{{dsHash}}"
	}`, "@at(0)")

	// ds head ref => ds head:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "meta",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.meta",
				"name": "Struct{\u2026}"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "parents",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.parents",
				"name": "Set(0)"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "value",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value",
				"name": "StructName"
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target",
		"name": "Commit"
	}`, "@at(0)@target")

	// ds head value => strct.
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "blob",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.blob",
				"name": "Blob(0 B)"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "bool",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.bool",
				"name": "true"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "list",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value.list",
				"name": "List(2)"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "map",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value.map",
				"name": "Map(2)"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "number",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.number",
				"name": "42"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "ref",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value.ref",
				"name": "Bool#g19moobgrm32dn083bokhksuobulq28c"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "set",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value.set",
				"name": "Set(2)"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "string",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.string",
				"name": "\"hello world\""
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target.value",
		"name": "StructName"
	}`, "@at(0)@target.value")

	// strct.blob:
	test(`{
		"children": [],
		"hasChildren": false,
		"id": "@at(0)@target.value.blob",
		"name": "Blob(0 B)"
	}`, "@at(0)@target.value.blob")

	// strct.bool:
	test(`{
		"children": [],
		"hasChildren": false,
		"id": "@at(0)@target.value.bool",
		"name": "true"
	}`, "@at(0)@target.value.bool")

	// strct.list:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.list[0]",
				"name": "1"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.list[1]",
				"name": "2"
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target.value.list",
		"name": "List(2)"
	}`, "@at(0)@target.value.list")

	// strct.map:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "@at(0)@target.value.map@at(0)@key",
				"name": "\"a\""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.map@at(0)",
				"name": "\"b\""
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "@at(0)@target.value.map@at(1)@key",
				"name": "\"c\""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.map@at(1)",
				"name": "\"d\""
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target.value.map",
		"name": "Map(2)"
	}`, "@at(0)@target.value.map")

	// strct.number:
	test(`{
		"children": [],
		"hasChildren": false,
		"id": "@at(0)@target.value.number",
		"name": "42"
	}`, "@at(0)@target.value.number")

	// strct.ref:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": true,
				"id": "@at(0)@target.value.ref@target",
				"name": "Bool#g19moobgrm32dn083bokhksuobulq28c"
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target.value.ref",
		"name": "Bool#g19moobgrm32dn083bokhksuobulq28c"
	}`, "@at(0)@target.value.ref")

	// strct.set:
	test(`{
		"children": [
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.set@at(0)",
				"name": "3"
			}
		},
		{
			"key": {
				"hasChildren": false,
				"id": "",
				"name": ""
			},
			"label": "",
			"value": {
				"hasChildren": false,
				"id": "@at(0)@target.value.set@at(1)",
				"name": "4"
			}
		}
		],
		"hasChildren": true,
		"id": "@at(0)@target.value.set",
		"name": "Set(2)"
	}`, "@at(0)@target.value.set")

	// strct.string:
	test(`{
		"children": [],
		"hasChildren": false,
		"id": "@at(0)@target.value.string",
		"name": "\"hello world\""
	}`, "@at(0)@target.value.string")
}
