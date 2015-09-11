package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/attic-labs/noms/Godeps/_workspace/src/golang.org/x/oauth2"
	"github.com/attic-labs/noms/Godeps/_workspace/src/golang.org/x/oauth2/google"
	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/marshal"
	"github.com/attic-labs/noms/types"
)

const maxProcs = 25

var (
	apiKeyFlag         = flag.String("api-key", "", "API keys for Google can be created at https://console.developers.google.com")
	apiKeySecretFlag   = flag.String("api-key-secret", "", "API keys for Google can be created at https://console.developers.google.com")
	albumIdFlag        = flag.String("album-id", "", "Import a specific album, identified by id")
	forceAuthFlag      = flag.Bool("force-auth", false, "Force re-authentication")
	quietFlag          = flag.Bool("quiet", false, "Don't print progress information")
	ds                 *dataset.Dataset
	cachingHttpClient  *http.Client
	authHttpClient     *http.Client
	start              = time.Now()
)

func picasaUsage() {
	credentialSteps := `How to create Google API credentials:
  1) Go to http://console.developers.google.com/start
  2) From the "Select a project" pull down menu, choose "Create a project..."
  3) Fill in the "Project name" field (e.g. Picasa Importer)
  4) Agree to the terms and conditions and hit continue.
  5) Click on the "Select a project" pull down menu and choose "Manage all projects..."
  6) Click on the project you just created. On the new page, in the sidebar menu,
     click “APIs and auth”. In the submenu that opens up, click "Credentials".
  7) In the popup, click on the "Add credentials" pulldown and select "OAuth 2.0 client ID".
  8) Click the "Configure consent screen" button and fill in the "Product name" field.
     All other fields on this page are optional. Now click the save button.
  9) Select "Other" from the list of “Application Type” and fill in the “Name” field
     (e.g. Picasa Importer) and click the “Create” button.
     Your credentials will be displayed in a popup. Copy them to a safe place.`

	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s\n\n", credentialSteps)
}

func main() {
	flag.Usage = picasaUsage
	dsFlags := dataset.NewFlags()
	flag.Parse()
	cachingHttpClient = util.CachingHttpClient()

	if *apiKeyFlag == "" || *apiKeySecretFlag == "" || cachingHttpClient == nil {
		flag.Usage()
		return
	}

	ds = dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}
	defer ds.Close()

	c, refreshToken := doAuthentication()
	authHttpClient = c
	
	// reset start so that we don't count the time to authenticate
	start = time.Now()

	var nomsUser types.Value
	if *albumIdFlag != "" {
		nomsUser, _, _ = getAlbum(*albumIdFlag, 1)
	} else {
		nomsUser = getAlbums()
	}
	
	nomsUser = setValueInNomsMap(nomsUser, "RefreshToken", types.NewString(refreshToken))
	printStats(nomsUser)
	_, ok := ds.Commit(nomsUser)
	d.Exp.True(ok, "Could not commit due to conflicting edit")
}

func getAlbum(albumId string, index int) (types.Value, types.Value, types.Value) {
	aj := AlbumJson{}
	p := fmt.Sprintf("user/default/albumid/%s?alt=json&max-results=0", albumId)
	callPicasaApi(authHttpClient, p, &aj)
	u := User{Id: aj.Feed.UserId.V, Name: aj.Feed.UserName.V}
	a := Album{Id: aj.Feed.Id.V, Title: aj.Feed.Title.V, NumPhotos: aj.Feed.NumPhotos.V}
	npl := a.getPhotos(1) // nomsPhotoList
	na := marshal.Marshal(a) // nomsAlbum
	na = setValueInNomsMap(na, "Photos", npl)
	nu := marshal.Marshal(u) // nomsUser
	nu = setValueInNomsMap(nu, "Albums", types.NewList(na))
	return nu, na, npl
}

func getAlbums() types.Value {
	aj := AlbumListJson{}
	callPicasaApi(authHttpClient, "user/default?alt=json", &aj)
	user := User{Id: aj.Feed.UserId.V, Name: aj.Feed.UserName.V}

	if !*quietFlag {
		fmt.Printf("Found %d albums\n", len(aj.Feed.Entry))
	}
	var nal = types.NewList()  // nomsAlbumList
	for i, entry := range aj.Feed.Entry {
		a := Album{Id: entry.Id.V, Title: entry.Title.V, NumPhotos: entry.NumPhotos.V}
		npl := a.getPhotos(i) // nomsPhotoList
		na := marshal.Marshal(a) // nomsAlbum
		na = setValueInNomsMap(na, "Photos", npl) 
		nal = nal.Append(na)
	}

	nu := marshal.Marshal(user) // nomsUser
	nu = setValueInNomsMap(nu, "Albums", nal)

	return nu
}

func (a *Album) getPhotos(albumIndex int) types.List {
	if (a.NumPhotos <= 0 || len(a.Photos) > 0) {
		return nil
	}
	photos := make([]Photo, 0, a.NumPhotos)
	if !*quietFlag {
		fmt.Printf("Album #%d: %q contains %d photos... ", albumIndex, a.Title, a.NumPhotos)
	}
	for startIndex, foundPhotos := 0, true ; a.NumPhotos > len(photos) && foundPhotos ; startIndex += 1000 {
		foundPhotos = false
		aj := AlbumJson{}
		p := fmt.Sprintf("user/default/albumid/%s?alt=json&max-results=1000&imgmax=d", a.Id)
		if startIndex > 0 {
			p = fmt.Sprintf("%s%s%d", p, "&start-index=", startIndex)
		}
		callPicasaApi(authHttpClient, p, &aj)
		for _, e := range aj.Feed.Entry {
			foundPhotos = true
			tags := splitTags(e.MediaGroup.Tags.V)
			p := Photo{
				NomsName: "Photo",
				Height:   e.Height.V,
				Id:       e.Id.V,
				Tags:     tags,
				Title:    e.Title.V,
				Url:      e.Content.Src,
				Width:    e.Width.V,
			}
			photos = append(photos, p)
		}
	}

	pChan, rChan := getImageFetcher(len(photos))
	for i, p := range photos {
		pChan <- PhotoMessage{i, p}
	}

	refMessages := make([]RefMessage, 0, a.NumPhotos)
	for i, timedOut := 0, false; i < len(photos) && !timedOut; i++ {
		select {
		case rm := <- rChan:
			refMessages = append(refMessages, rm)
		case <- time.After(10 * time.Second):
			timedOut = true
			if !*quietFlag {
				fmt.Println("Not all responses received")
			}
			break;
		}
	}
	sort.Sort(ByIndex(refMessages))
	npl := types.NewList() // nomsPhotoList
	for _, refMsg := range refMessages  {
		npl = npl.Append(types.Ref{refMsg.Ref})
	}

	if !*quietFlag {
		fmt.Printf("fetched %d, elapsed time: %.2f secs\n", npl.Len(), time.Now().Sub(start).Seconds())
	}
	return npl
}

func getImageFetcher(numPhotos int) (pChan chan PhotoMessage, rChan chan RefMessage) {
	pChan = make(chan PhotoMessage, numPhotos)
	rChan = make(chan RefMessage)
	n := min(numPhotos, maxProcs)

	for i := 0; i < n; i++ {
		go func() {
			for timedOut := false; !timedOut; {
				select {
				case msg := <- pChan:
					msg.Photo.Image = getPhoto(msg.Photo.Url)
					nomsPhoto := marshal.Marshal(msg.Photo)
					ref := types.WriteValue(nomsPhoto, ds.Store())
					rChan <- RefMessage{msg.Index, ref}
				case <- time.After(1 * time.Second):
					timedOut = true
					break;
				}
			}
		}()
	}

	return
}

func doAuthentication() (c *http.Client, rt string) {
	if !*forceAuthFlag {
		rt = getRefreshToken()
		c = tryRefreshToken(rt)
	}
	if c == nil {
		c, rt = googleOAuth()
	}
	return c, rt
}

func getRefreshToken() string {
	tj := RefreshTokenJson{}

	if commit, ok := ds.MaybeHead(); ok {
		marshal.Unmarshal(commit.Value(), &tj)
	}
	return tj.RefreshToken
}

func tryRefreshToken(rt string) *http.Client {
	var c *http.Client

	if rt != "" {
		t := oauth2.Token{}
		conf := baseConfig("")
		ct := "application/x-www-form-urlencoded"
		body := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=refresh_token&refresh_token=%s", *apiKeyFlag, *apiKeySecretFlag, rt)
		r, err := cachingHttpClient.Post(google.Endpoint.TokenURL, ct, strings.NewReader(body))
		d.Chk.NoError(err)
		if r.StatusCode == 200 {
			buf, err := ioutil.ReadAll(r.Body)
			d.Chk.NoError(err)
			json.Unmarshal(buf, &t)
			c = conf.Client(oauth2.NoContext, &t)
		}
	}
	return c
}

func googleOAuth() (*http.Client, string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	d.Chk.NoError(err)

	redirectUrl := "http://" + l.Addr().String()
	conf := baseConfig(redirectUrl)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	state := fmt.Sprintf("%v", r.Uint32())
	u := conf.AuthCodeURL(state)

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	fmt.Printf("Visit the following URL to authorize access to your Picasa data: %v\n", u)
	code, returnedState := awaitOAuthResponse(l)
	d.Chk.Equal(state, returnedState, "Oauth state is not correct")

	// Handle the exchange code to initiate a transport.
	t, err := conf.Exchange(oauth2.NoContext, code)
	d.Chk.NoError(err)

	client := conf.Client(oauth2.NoContext, t)
	return client, t.RefreshToken
}

func awaitOAuthResponse(l net.Listener) (string, string) {
	var code, state string

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("code") != "" && r.URL.Query().Get("state") != "" {
			code = r.URL.Query().Get("code")
			state = r.URL.Query().Get("state")
			w.Header().Add("content-type", "text/plain")
			fmt.Fprintf(w, "Authorized")
			l.Close()
		} else if err := r.URL.Query().Get("error"); err == "access_denied" {
			fmt.Fprintln(os.Stderr, "Request for authorization was denied.")
			os.Exit(0)
		} else if err := r.URL.Query().Get("error"); err != "" {
			l.Close()
			d.Chk.Fail(err)
		}
	})}
	srv.Serve(l)

	return code, state
}

func callPicasaApi(client *http.Client, path string, response interface{}) {
	u := "https://picasaweb.google.com/data/feed/api/" + path
	req, err := http.NewRequest("GET", u, nil)
	d.Chk.NoError(err)

	req.Header.Add("GData-Version", "2")
	resp, err := client.Do(req)
	d.Chk.NoError(err)

	msg := func() string {
		body := &bytes.Buffer{}
		_, err := io.Copy(body, resp.Body)
		d.Chk.NoError(err)
		return fmt.Sprintf("could not load %s: %d: %s", u, resp.StatusCode, body)
	}

	switch resp.StatusCode / 100 {
	case 4:
		d.Exp.Fail(msg())
	case 5:
		d.Chk.Fail(msg())
	}

	err = json.NewDecoder(resp.Body).Decode(response)
	d.Chk.NoError(err)
}

func baseConfig(redirectUrl string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     *apiKeyFlag,
		ClientSecret: *apiKeySecretFlag,
		RedirectURL:  redirectUrl,
		Scopes:       []string{"https://picasaweb.google.com/data"},
		Endpoint:     google.Endpoint,
	}
}

func getPhotoReader(url string) io.ReadCloser {
	r, err := cachingHttpClient.Get(url)
	d.Chk.NoError(err)
	return r.Body
}

func getPhoto(url string) *bytes.Reader {
	pr := getPhotoReader(url)
	defer pr.Close()
	buf, err := ioutil.ReadAll(pr)
	d.Chk.NoError(err)
	return bytes.NewReader(buf)
}

func printStats(nomsUser types.Value) {
	if !*quietFlag {
		numPhotos := uint64(0)
		nomsAlbums := getValueInNomsMap(nomsUser, "Albums").(types.List)
		for i := uint64(0); i < nomsAlbums.Len(); i++ {
			nomsAlbum := nomsAlbums.Get(i)
			nomsPhotos := getValueInNomsMap(nomsAlbum, "Photos").(types.List)
			numPhotos = numPhotos + nomsPhotos.Len()
		}

		fmt.Printf("Imported %d album(s), %d photo(s), time: %.2f\n", nomsAlbums.Len(), numPhotos, time.Now().Sub(start).Seconds())
	}
}

// General utility functions

func getValueInNomsMap(m types.Value, field string) types.Value {
	return m.(types.Map).Get(types.NewString(field))
}

func setValueInNomsMap(m types.Value, field string, value types.Value) types.Value {
	return m.(types.Map).Set(types.NewString(field), value)
}

func toJson(str interface{}) string {
	v, err := json.Marshal(str)
	d.Chk.NoError(err)
	return string(v)
}

func min(a, b int) int {
	if (a < b) {
		return a
	}
	return b
}

func splitTags(s string) map[string]bool {
	tags := map[string]bool{}
	for _, s := range strings.Split(s, ",") {
		s1 := strings.Trim(s, " ")
		if s1 != "" {
			tags[s1] = true
		}
	}
	return tags
}

