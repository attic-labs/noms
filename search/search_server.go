package search

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

const (
	maxConcurrentPuts = 64
	rootPath          = "/root/"
	refPath           = "/ref/"
	getRefsPath       = "/getRefs/"
	postRefsPath      = "/postRefs/"
)

type searchServer struct {
	cs         Searcher
	port       int
	l          *net.Listener
	conns      map[net.Conn]http.ConnState
	writeLimit chan struct{}
}

func NewSearchServer(cs Searcher, port int) *searchServer {
	return &searchServer{
		cs, port, nil, map[net.Conn]http.ConnState{}, make(chan struct{}, maxConcurrentPuts),
	}
}

func (s *searchServer) handleGetReachable(r ref.Ref, w http.ResponseWriter, req *http.Request) {
	excludeRef := ref.Ref{}
	exclude := req.URL.Query().Get("exclude")
	if exclude != "" {
		excludeRef = ref.Parse(exclude)
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	writer := w.(io.Writer)
	if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Add("Content-Encoding", "gzip")
		gw := gzip.NewWriter(w)
		defer gw.Close()
		writer = gw
	}

	sz := chunks.NewSerializer(writer)
	s.cs.CopyReachableChunksP(r, excludeRef, sz, 512)
	sz.Close()
}

func (s *searchServer) handleRef(w http.ResponseWriter, req *http.Request) {
	err := d.Try(func() {
		refStr := ""
		pathParts := strings.Split(req.URL.Path[1:], "/")
		if len(pathParts) > 1 {
			refStr = pathParts[1]
		}
		r := ref.Parse(refStr)

		switch req.Method {
		case "GET":
			all := req.URL.Query().Get("all")
			if all == "true" {
				s.handleGetReachable(r, w, req)
				return
			}
			chunk := s.cs.Get(r)
			if chunk.IsEmpty() {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			_, err := io.Copy(w, bytes.NewReader(chunk.Data()))
			d.Chk.NoError(err)
			w.Header().Add("Content-Type", "application/octet-stream")
			w.Header().Add("Cache-Control", "max-age=31536000") // 1 year

		case "HEAD":
			if !s.cs.Has(r) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		default:
			d.Exp.Fail("Unexpected method: ", req.Method)
		}
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
}

func (s *searchServer) handlePostRefs(w http.ResponseWriter, req *http.Request) {
	err := d.Try(func() {
		d.Exp.Equal("POST", req.Method)

		var reader io.Reader = req.Body
		if strings.Contains(req.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(reader)
			d.Exp.NoError(err)
			defer gr.Close()
			reader = gr
		}

		chunks.Deserialize(reader, s.cs)
		w.WriteHeader(http.StatusCreated)
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
}

func (s *searchServer) handleGetRefs(w http.ResponseWriter, req *http.Request) {
	err := d.Try(func() {
		d.Exp.Equal("POST", req.Method)

		req.ParseForm()
		refStrs := req.PostForm["ref"]
		d.Exp.True(len(refStrs) > 0)

		refs := make([]ref.Ref, 0)
		for _, refStr := range refStrs {
			refs = append(refs, ref.Parse(refStr))
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		writer := w.(io.Writer)
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Add("Content-Encoding", "gzip")
			gw := gzip.NewWriter(w)
			defer gw.Close()
			writer = gw
		}

		sz := chunks.NewSerializer(writer)
		for _, r := range refs {
			c := s.cs.Get(r)
			if !c.IsEmpty() {
				sz.Put(c)
			}
		}
		sz.Close()
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
}

func (s *searchServer) handleRoot(w http.ResponseWriter, req *http.Request) {
	err := d.Try(func() {
		switch req.Method {
		case "GET":
			rootRef := s.cs.Root()
			fmt.Fprintf(w, "%v", rootRef.String())
			w.Header().Add("content-type", "text/plain")

		case "POST":
			params := req.URL.Query()
			tokens := params["last"]
			d.Exp.Len(tokens, 1)
			last := ref.Parse(tokens[0])
			tokens = params["current"]
			d.Exp.Len(tokens, 1)
			current := ref.Parse(tokens[0])

			if !s.cs.UpdateRoot(current, last) {
				w.WriteHeader(http.StatusConflict)
				return
			}
		}
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
}

func (s *searchServer) connState(c net.Conn, cs http.ConnState) {
	switch cs {
	case http.StateNew, http.StateActive, http.StateIdle:
		s.conns[c] = cs
	default:
		delete(s.conns, c)
	}
}

// Blocks while the searchServer is listening. Running on a separate go routine is supported.
func (s *searchServer) Run() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	d.Chk.NoError(err)
	s.l = &l

	mux := http.NewServeMux()

	mux.HandleFunc(refPath, http.HandlerFunc(s.handleRef))
	mux.HandleFunc(getRefsPath, http.HandlerFunc(s.handleGetRefs))
	mux.HandleFunc(postRefsPath, http.HandlerFunc(s.handlePostRefs))
	mux.HandleFunc(rootPath, http.HandlerFunc(s.handleRoot))

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			mux.ServeHTTP(w, r)
		}),
		ConnState: s.connState,
	}
	srv.Serve(l)
}

// Will cause the searchServer to stop listening and an existing call to Run() to continue.
func (s *searchServer) Stop() {
	(*s.l).Close()
	for c, _ := range s.conns {
		c.Close()
	}
}
