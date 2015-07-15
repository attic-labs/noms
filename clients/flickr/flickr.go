package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

var (
	apiKeyFlag       *string = flag.String("api-key", "", "API keys for flickr can be created at https://www.flickr.com/services/apps/create/apply")
	apiKeySecretFlag *string = flag.String("api-key-secret", "", "API keys for flickr can be created at https://www.flickr.com/services/apps/create/apply")
	albumIdFlag      *string = flag.String("album-id", "", "Import a specific album, identified by id")
	ds               *dataset.Dataset
	user             User
	oauthClient      oauth.Client
)

type flickrCall struct {
	Stat string
}

func main() {
	dsFlags := dataset.Flags()
	flag.Parse()

	if *apiKeyFlag == "" || *apiKeySecretFlag == "" {
		flag.Usage()
		return
	}

	oauthClient = oauth.Client{
		TemporaryCredentialRequestURI: "https://www.flickr.com/services/oauth/request_token",
		ResourceOwnerAuthorizationURI: "https://www.flickr.com/services/oauth/authorize",
		TokenRequestURI:               "https://www.flickr.com/services/oauth/access_token",
		Credentials: oauth.Credentials{
			Token:  *apiKeyFlag,
			Secret: *apiKeySecretFlag,
		},
	}

	ds = dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}

	getUser()
	if *albumIdFlag != "" {
		getPhotoset(*albumIdFlag)
	} else {
		getPhotosets()
	}
	commitUser()
}

func getUser() {
	roots := ds.Roots()
	if roots.Len() > uint64(0) {
		user = UserFromVal(roots.Any().Value())
		if checkAuth() {
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
	Chk.NoError(err)

	callbackURL := "http://" + l.Addr().String()
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, callbackURL, url.Values{
		"perms": []string{"read"},
	})
	// If we ever hear anything from the oauth handshake, it'll be acceptance. The user declining will mean we never get called.
	Chk.NoError(err)

	authUrl := oauthClient.AuthorizationURL(tempCred, nil)
	fmt.Printf("Go to the following URL to authorize: %v\n", authUrl)
	err = awaitOAuthResponse(l, tempCred)
	Chk.NoError(err)

	if !checkAuth() {
		Chk.Fail("checkAuth failed after oauth succeded")
	}
}

func getPhotoset(id string) {
	response := struct {
		flickrCall
		Photoset struct {
			Id    string `json:"id"`
			Title struct {
				Content string `json:"_content"`
			} `json:"title"`
		} `json:"photoset"`
	}{}

	err := callFlickrAPI("flickr.photosets.getInfo", &response, &map[string]string{
		"photoset_id": id,
		"user_id":     user.Id().String(),
	})
	Chk.NoError(err)

	fmt.Printf("\nPhotoset: %v\n", response.Photoset.Title)

	// TODO: Retrieving a field which hasn't been set will crash, so we have to reach inside and test the untyped
	var photosets PhotosetSet
	if !user.NomsValue().Has(types.NewString("photosets")) {
		photosets = NewPhotosetSet()
	} else {
		photosets = user.Photosets()
	}

	photos := getPhotosetPhotos(id)
	photoset := NewPhotoset().SetId(types.NewString(id)).SetTitle(types.NewString(response.Photoset.Title.Content)).SetPhotos(photos)
	photosets = photosets.Insert(photoset)
	user = user.SetPhotosets(photosets)
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

	for _, p := range response.Photosets.Photoset {
		getPhotoset(p.Id)
	}
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

	// TODO: Implement paging. This call returns a maximum of 500 pictures in each response.
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
		"method":         []string{method},
		"format":         []string{"json"},
		"nojsoncallback": []string{"1"},
	}

	if args != nil {
		for k, v := range *args {
			values[k] = []string{v}
		}
	}

	res, err := oauthClient.Get(nil, tokenCred, "https://api.flickr.com/services/rest/", values)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	buff, err := ioutil.ReadAll(res.Body)
	Chk.NoError(err)
	if err = json.Unmarshal(buff, response); err != nil {
		return err
	}

	status := reflect.ValueOf(response).Elem().FieldByName("Stat").Interface().(string)
	if status != "ok" {
		err = errors.New(fmt.Sprintf("Failed flickr API call: %v, status: &v", method, status))
	}
	return nil
}
