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

var (
	jsonResponsePrefix  = []byte("jsonFlickrApi(")
	jsonResponsePostfix = []byte(")")
	ds                  *datas.DataStore
	user                User
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

type flickrCall struct {
	Stat string
}

func main() {
	datasetDataStoreFlags := dataset.DatasetDataFlags()
	flag.Parse()
	ds = datasetDataStoreFlags.CreateStore()

	getUser()
	getPhotosets()
	commitUser()
}

func getUser() {
	roots := ds.Roots()
	if roots.Len() > uint64(0) {
		user = UserFromVal(roots.Any().Value())
		if checkAuth() {
			fmt.Println("OAuth credentials are still good.")
			return
		}
	} else {
		user = NewUser()
	}

	authUser()
}

func checkAuth() bool {
	response := struct {
		flickrCall
		User struct {
			Id       string `json:"id"`
			Username struct {
				Content string `json:"_content"`
			} `json:"username"`
		} `json:"user"`
	}{}

	err := callFlickrAPI("flickr.test.login", &response, nil)
	if err != nil {
		return false
	}

	user = user.SetId(types.NewString(response.User.Id)).SetName(types.NewString(response.User.Username.Content))
	return true
}

func authUser() {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	callbackURL := "http://" + l.Addr().String()
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, callbackURL, url.Values{
		"perms": []string{"read"},
	})
	Chk.NoError(err)

	authUrl := oauthClient.AuthorizationURL(tempCred, nil)
	fmt.Printf("Go to the following URL to authorize: %v\n", authUrl)
	err = awaitOAuthResponse(l, tempCred)
	Chk.NoError(err)

	if !checkAuth() {
		panic(errors.New("checkAuth failed after oauth succeded"))
	}
	Chk.NoError(err)
}

func getPhotosets() {
	response := struct {
		flickrCall
		Photosets struct {
			Photoset []struct {
				Id    string `json:"id"`
				Title struct {
					Content string `json:"_content"`
				} `json:"title"`
			} `json:"photoset"`
		} `json:"photosets"`
	}{}

	err := callFlickrAPI("flickr.photosets.getList", &response, nil)
	Chk.NoError(err)

	photosets := NewPhotosetSet()
	for _, p := range response.Photosets.Photoset {
		fmt.Printf("\nPhotoset: %v\n", p.Title)

		photos := getPhotosetPhotos(p.Id)
		photoset := NewPhotoset().SetId(types.NewString(p.Id)).SetTitle(types.NewString(p.Title.Content)).SetPhotos(photos)
		photosets = photosets.Insert(photoset)
	}
	user = user.SetPhotosets(photosets)
}

func getPhotosetPhotos(id string) PhotoSet {
	response := struct {
		flickrCall
		Photoset struct {
			Photo []struct {
				Id    string `json:"id"`
				Title string `json:"title"`
			} `json:"photo"`
		} `json:"photoset"`
	}{}

	err := callFlickrAPI("flickr.photosets.getPhotos", &response, &map[string]string{
		"photoset_id": id,
		"user_id":     user.Id().String(),
	})
	Chk.NoError(err)

	photoSet := NewPhotoSet()
	for _, p := range response.Photoset.Photo {
		url := getOriginalUrl(p.Id)
		fmt.Printf(" . %v\n", url)
		photoBytes := getPhotoBytes(url)
		photo := NewPhoto().SetId(types.NewString(p.Id)).SetTitle(types.NewString(p.Title)).SetUrl(types.NewString(url)).SetImage(types.NewBlob(photoBytes))
		photoSet = photoSet.Insert(photo)
	}
	return photoSet
}

func getOriginalUrl(id string) string {
	response := struct {
		flickrCall
		Sizes struct {
			Size []struct {
				Label  string `json:"label"`
				Source string `json:"source"`
				// TODO: For some reason json unmarshalling was getting confused about types. Not sure why.
				// Width  int `json:"width"`
				// Height int `json:"height"`
			} `json:"size"`
		} `json:"sizes"`
	}{}

	err := callFlickrAPI("flickr.photos.getSizes", &response, &map[string]string{
		"photo_id": id,
	})
	if err != nil {
		panic(err)
	}

	for _, p := range response.Sizes.Size {
		if p.Label == "Original" {
			return p.Source
		}
	}
	panic(errors.New(fmt.Sprintf("No Original image size found photo: %v", id)))
}

func getPhotoBytes(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var buff bytes.Buffer
	buff.ReadFrom(resp.Body)
	return buff.Bytes()
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

func callFlickrAPI(method string, response interface{}, args *map[string]string) error {
	tokenCred := &oauth.Credentials{
		user.OAuthToken().String(),
		user.OAuthSecret().String(),
	}

	values := url.Values{
		"method": []string{method},
		"format": []string{"json"},
	}
	if args != nil {
		for k, v := range *args {
			values[k] = []string{v}
		}
	}

	res, err := oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", values)
	Chk.NoError(err)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if err = json.Unmarshal(getJSONBytes(res.Body), response); err != nil {
		return err
	}

	status := reflect.ValueOf(response).Elem().FieldByName("Stat").Interface().(string)
	if status != "ok" {
		err = errors.New(fmt.Sprintf("Failed flickr API call: %v, status: &v", method, status))
	}
	return nil
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
