package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/constants"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/julienschmidt/httprouter"
)

const (
	dsPathPrefix = "/-ds"
	serveCmd     = "serve"
)

var (
	hostFlag = flag.String("host", "localhost:0", "Host to listen on")
)

type dataStoreRecord struct {
	ds    datas.DataStore
	alias string
}

type dataStoreRecords map[string]dataStoreRecord

func main() {
	usage := func() {
		flag.PrintDefaults()
		fmt.Printf("Usage: %s %s <view-dir> arg1=val1 arg2=val2...\n", os.Args[0], serveCmd)
	}

	flag.Parse()
	flag.Usage = usage

	if len(flag.Args()) < 2 || flag.Arg(0) != serveCmd {
		usage()
		os.Exit(1)
	}

	viewDir := flag.Arg(1)
	qsValues, stores := constructQueryString(flag.Args()[2:])

	router := &httprouter.Router{
		HandleMethodNotAllowed: true,
		NotFound:               http.FileServer(http.Dir(viewDir)),
		RedirectFixedPath:      true,
	}

	prefix := dsPathPrefix + "/:store"
	router.GET(prefix+constants.RefPath+":ref", routeToStore(stores, datas.HandleRef))
	router.OPTIONS(prefix+constants.RefPath+":ref", routeToStore(stores, datas.HandleRef))
	router.POST(prefix+constants.PostRefsPath, routeToStore(stores, datas.HandlePostRefs))
	router.OPTIONS(prefix+constants.PostRefsPath, routeToStore(stores, datas.HandlePostRefs))
	router.POST(prefix+constants.GetHasPath, routeToStore(stores, datas.HandleGetHasRefs))
	router.OPTIONS(prefix+constants.GetHasPath, routeToStore(stores, datas.HandleGetHasRefs))
	router.POST(prefix+constants.GetRefsPath, routeToStore(stores, datas.HandleGetRefs))
	router.OPTIONS(prefix+constants.GetRefsPath, routeToStore(stores, datas.HandleGetRefs))
	router.GET(prefix+constants.RootPath, routeToStore(stores, datas.HandleRootGet))
	router.POST(prefix+constants.RootPath, routeToStore(stores, datas.HandleRootPost))
	router.OPTIONS(prefix+constants.RootPath, routeToStore(stores, datas.HandleRootGet))

	l, err := net.Listen("tcp", *hostFlag)
	d.Chk.NoError(err)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			router.ServeHTTP(w, req)
		}),
	}

	qs := ""
	if len(qsValues) > 0 {
		qs = "?" + qsValues.Encode()
	}

	fmt.Printf("Starting view %s at http://%s%s\n", viewDir, l.Addr().String(), qs)
	log.Fatal(srv.Serve(l))
}

func constructQueryString(args []string) (url.Values, dataStoreRecords) {
	qsValues := url.Values{}
	stores := dataStoreRecords{}

	for _, arg := range args {
		k, v, ok := split2(arg, "=")
		if !ok {
			continue
		}

		// Magically assume that ldb: prefixed arguments are references to ldb stores. If so, construct
		// httpstore proxies to them, and rewrite the path to the client.
		if strings.HasPrefix(v, "ldb:") {
			_, path, _ := split2(v, ":")
			record, ok := stores[path]
			if !ok {
				record.ds = datas.NewDataStore(chunks.NewLevelDBStore(path, "", 24, false))
				hash := sha1.Sum([]byte(path))
				record.alias = hex.EncodeToString(hash[:])
				stores[path] = record
			}
			v = fmt.Sprintf("%s/%s", dsPathPrefix, record.alias)
		}

		qsValues.Add(k, v)
	}

	return qsValues, stores
}

func routeToStore(stores dataStoreRecords, handler datas.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		store := params.ByName("store")
		for _, record := range stores {
			if record.alias == store {
				handler(w, r, params, record.ds)
				return
			}
		}
		d.Chk.Fail("No store named", store)
	}
}

func split2(s, sep string) (string, string, bool) {
	substrs := strings.SplitN(s, sep, 2)
	if len(substrs) != 2 {
		fmt.Println("Invalid arg %s, must be of form k%sv", s, sep)
		return "", "", false
	}
	return substrs[0], substrs[1], true
}
