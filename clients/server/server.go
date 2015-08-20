package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

var (
	port = flag.String("port", "8000", "")
)

type server struct {
	cs chunks.ChunkStore
}

func (s server) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	switch r.URL.Path[1:] {
	case "root":
		w.Header().Add("content-type", "text/plain")
		fmt.Fprintf(w, "%v", s.cs.Root().String())
	case "get":
		if refs, ok := r.URL.Query()["ref"]; ok {
			s.handleGetRef(w, refs[0])
		} else {
			http.Error(w, "Missing query param ref", http.StatusBadRequest)
		}
	default:
		http.Error(w, fmt.Sprintf("Unrecognized: %v", r.URL.Path[1:]), http.StatusBadRequest)
	}
}

func (s server) handleGetRef(w http.ResponseWriter, hashString string) {
	err := d.Try(func() {
		r := ref.Parse(hashString)
		reader := s.cs.Get(r)
		if reader == nil {
			http.Error(w, fmt.Sprintf("No such ref: %v", hashString), http.StatusNotFound)
			return
		}

		w.Header().Add("content-type", "application/octet-stream")
		w.Header().Add("cache-control", "max-age=31536000") // 1 year
		io.Copy(w, reader)
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Parse error: %v", err), http.StatusBadRequest)
		return
	}
}

func main() {
	flags := chunks.NewFlags()
	flag.Parse()

	cs := flags.CreateStore()
	if cs == nil {
		flag.Usage()
		return
	}

	http.HandleFunc("/", server{cs}.handle)
	http.ListenAndServe(fmt.Sprintf(":%s", *port), nil)
}
