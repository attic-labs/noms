package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
	"github.com/garyburd/go-oauth/oauth"
)

// Session state keys.
const (
	tempCredKey  = "tempCred"
	tokenCredKey = "tokenCred"
)

var oauthClient = oauth.Client{
	TemporaryCredentialRequestURI: "https://www.flickr.com/services/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://www.flickr.com/services/oauth/authorize",
	TokenRequestURI:               "https://www.flickr.com/services/oauth/access_token",
	Credentials: oauth.Credentials{
		Token:  "b404ebc4edc07f75f9ae6e14820ef591",
		Secret: "0aacb9788ab8d010",
	},
}

var tempCred *oauth.Credentials

func main() {
	var err error

	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	callback := "http://" + l.Addr().String()

	tempCred, err = oauthClient.RequestTemporaryCredentials(nil, callback, url.Values{
		"perms": []string{"read"},
	})

	if err != nil {
		fmt.Printf("Error getting temp cred, %v\n", err.Error())
		return
	}

	url := oauthClient.AuthorizationURL(tempCred, nil)
	fmt.Printf("Go to the following URL to authorize: %v\n", url)

	srv := &http.Server{Handler: newHandler(l)}
	srv.Serve(l)
}

func callGetPhotoSetList(tokenCred *oauth.Credentials) {
	fmt.Println("flickr.photosets.getList")

	res, err := oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", url.Values{
		"method": []string{"flickr.photosets.getList"},
		"format": []string{"json"},
	})
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(res.Body)
	text := string(body)

	fmt.Println(text)
}

func callAPI(tokenCred *oauth.Credentials) {
	fmt.Println("flickr.test.login")

	res, err := oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", url.Values{
		"method": []string{"flickr.test.login"},
		"format": []string{"json"},
	})
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(res.Body)
	text := string(body)

	fmt.Println(text)

	datasetDataStoreFlags := dataset.DatasetDataFlags()
	flag.Parse()

	ds := datasetDataStoreFlags.CreateStore()
	roots := ds.Roots()

	appRoot := types.NewString("Hello, AppRoot!")
	ds.Commit(datas.NewRootSet().Insert(
		datas.NewRoot().SetParents(
			roots.NomsValue()).SetValue(
			appRoot)))

	callGetPhotoSetList(tokenCred)
}

func newHandler(l *net.TCPListener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenCred, _, err := oauthClient.RequestToken(nil, tempCred, r.FormValue("oauth_verifier"))
		if err != nil {
			http.Error(w, "Error getting request token, "+err.Error(), 500)
			return
		}

		callAPI(tokenCred)
		l.Close()
	}
}
