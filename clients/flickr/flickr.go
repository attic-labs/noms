package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	. "github.com/attic-labs/noms/dbg"
	"github.com/attic-labs/noms/types"
	"github.com/garyburd/go-oauth/oauth"
)

//go:generate go run gen/types.go -o types.go

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

var (
	tempCred            *oauth.Credentials
	jsonResponsePrefix  = []byte("jsonFlickrApi(")
	jsonResponsePostfix = []byte(")")
	ds                  *datas.DataStore
	user                User
)

func main() {
	datasetDataStoreFlags := dataset.DatasetDataFlags()
	flag.Parse()
	ds = datasetDataStoreFlags.CreateStore()

	getUser()
}

func getUser() {
	roots := ds.Roots()
	if roots.Len() > uint64(0) {
		user = UserFromVal(roots.Any().Value())
		if checkAuth() == nil {
			fmt.Println("OAuth credentials are still good.")
			return
		}
	} else {
		user = NewUser()
	}

	fmt.Println("OAuth authentication required.")
	authUser()
}

func checkAuth() error {
	response := struct {
		User struct {
			Id       string `json:"id"`
			Username struct {
				Content string `json:"_content"`
			} `json:"username"`
		} `json:"user"`
		Stat string `json:"stat"`
	}{}

	res, err := callFlickrAPI("flickr.test.login")
	if err != nil {
		return err
	}

	defer res.Body.Close()
	err = json.Unmarshal(getJSONBytes(res.Body), &response)
	if err != nil {
		return err
	}

	if response.Stat != "ok" {
		return errors.New(fmt.Sprintf("Failed test login. Status %v", response.Stat))
	}

	user = user.SetId(types.NewString(response.User.Id)).SetName(types.NewString(response.User.Username.Content))
	return nil
}

func authUser() {
	var err error

	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	callback := "http://" + l.Addr().String()

	tempCred, err = oauthClient.RequestTemporaryCredentials(nil, callback, url.Values{
		"perms": []string{"read"},
	})

	if err != nil {
		panic(fmt.Sprintf("Error getting temp cred, %v\n", err.Error()))
	}

	authUrl := oauthClient.AuthorizationURL(tempCred, nil)
	fmt.Printf("Go to the following URL to authorize: %v\n", authUrl)

	newHandler := func(l *net.TCPListener) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tokenCred, _, err := oauthClient.RequestToken(nil, tempCred, r.FormValue("oauth_verifier"))

			user = user.SetOAuthToken(types.NewString(tokenCred.Token)).SetOAuthSecret(types.NewString(tokenCred.Secret))

			if err != nil {
				http.Error(w, "Error getting request token, "+err.Error(), 500)
				return
			}

			l.Close()
			// TODO: handle error
		}
	}

	srv := &http.Server{Handler: newHandler(l)}
	srv.Serve(l)

	checkAuth()
	commitUser()
}

func commitUser() {
	roots := ds.Roots()
	rootSet := datas.NewRootSet().Insert(
		datas.NewRoot().SetParents(
			roots.NomsValue()).SetValue(
			user.NomsValue()))
	ds.Commit(rootSet)
}

func callFlickrAPI(method string) (*http.Response, error) {
	tokenCred := &oauth.Credentials{
		user.OAuthToken().String(),
		user.OAuthSecret().String(),
	}

	return oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", url.Values{
		"method": []string{method},
		"format": []string{"json"},
	})
}

func getJSONBytes(reader io.Reader) []byte {
	buff, err := ioutil.ReadAll(reader)
	Chk.NoError(err)
	if !bytes.Equal(buff[0:len(jsonResponsePrefix)], jsonResponsePrefix) ||
		!bytes.Equal(buff[len(buff)-len(jsonResponsePostfix):], jsonResponsePostfix) {
		panic(fmt.Sprintf("Unexpect json response: %v", buff))
	}

	return buff[len(jsonResponsePrefix) : len(buff)-len(jsonResponsePostfix)]
}

func callGetPhotoSetList() {
	res, err := callFlickrAPI("flickr.photosets.getList")
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
