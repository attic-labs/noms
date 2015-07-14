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
	"reflect"

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
	getPhotosets()
	// commitUser()
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

	authUser()
}

func checkAuth() error {
	response := struct {
		flickrCall
		User struct {
			Id       string `json:"id"`
			Username struct {
				Content string `json:"_content"`
			} `json:"username"`
		} `json:"user"`
	}{}

	err := callFlickrAPI("flickr.test.login", &response)
	if err != nil {
		return err
	}

	user = user.SetId(types.NewString(response.User.Id)).SetName(types.NewString(response.User.Username.Content))
	return nil
}

func getPhotosets() error {
	response := struct {
		flickrCall
		Photosets struct {
			Photoset []struct {
				Id    string
				Title struct {
					Content string `json:"_content"`
				} `json:"title"`
				Description struct {
					Content string `json:"_content"`
				} `json:"description"`
			} `json:"photoset"`
		} `json:"photosets"`
	}{}

	err := callFlickrAPI("flickr.photosets.getList", &response)
	if err != nil {
		return err
	}

	for _, p := range response.Photosets.Photoset {
		fmt.Println(p.Id)
		fmt.Println(p.Title.Content)
		fmt.Println(p.Description.Content)
	}

	return nil
}

func authUser() {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, "http://"+l.Addr().String(), url.Values{
		"perms": []string{"read"},
	})
	if err != nil {
		panic(err)
	}

	authUrl := oauthClient.AuthorizationURL(tempCred, nil)
	fmt.Printf("Go to the following URL to authorize: %v\n", authUrl)

	if err = awaitOAuthResponse(l, tempCred); err != nil {
		panic(err)
	}

	if err = checkAuth(); err != nil {
		panic(err)
	}

	commitUser()
}

func awaitOAuthResponse(l *net.TCPListener, tempCred *oauth.Credentials) error {
	var handlerError error

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain")
		var cred *oauth.Credentials
		cred, _, handlerError = oauthClient.RequestToken(nil, tempCred, r.FormValue("oauth_verifier"))
		if handlerError != nil {
			fmt.Fprintf(w, "%v", handlerError)
		} else {
			fmt.Fprintf(w, "Authorized")
			user = user.SetOAuthToken(types.NewString(cred.Token)).SetOAuthSecret(types.NewString(cred.Secret))
		}
		l.Close()
	})}
	srv.Serve(l)

	return handlerError
}

func commitUser() {
	roots := ds.Roots()
	rootSet := datas.NewRootSet().Insert(
		datas.NewRoot().SetParents(
			roots.NomsValue()).SetValue(
			user.NomsValue()))
	ds.Commit(rootSet)
}

type flickrCall struct {
	Stat string
}

func callFlickrAPI(method string, response interface{}) (err error) {
	tokenCred := &oauth.Credentials{
		user.OAuthToken().String(),
		user.OAuthSecret().String(),
	}

	res, err := oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", url.Values{
		"method": []string{method},
		"format": []string{"json"},
	})

	if err != nil {
		return
	}

	defer res.Body.Close()
	if err = json.Unmarshal(getJSONBytes(res.Body), response); err != nil {
		return
	}

	status := reflect.ValueOf(response).Elem().FieldByName("Stat").Interface().(string)
	if status != "ok" {
		err = errors.New(fmt.Sprintf("Failed flickr API call: %v, status: &v", method, status))
	}
	return
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
