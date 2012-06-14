package echonest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Status struct {
	Version string
	Code    int
	Message string
}

type Artist struct {
	Id   string
	Name string
}

type Song struct {
	Id          string
	Title       string
	Artist_id   string
	Artist_name string
	Tracks      []Track
}

type Track struct {
	Catalog    string
	Foreign_id string
	Id         string
}

type ArtistsResponse struct {
	Status  Status
	Artists []Artist
}

type SongsResponse struct {
	Status Status
	Songs  []Song
}

type query interface {
	GetCallInfo() (path string, params map[string][]string)
	setError(error)
}

type queryBase struct {
	Error error
}

func (query *queryBase) setError(err error) {
	query.Error = err
}

type ArtistExtractQuery struct {
	queryBase
	Text     string
	Response ArtistsResponse
}

func (query *ArtistExtractQuery) GetCallInfo() (path string, params map[string][]string) {
	path = "/artist/extract"
	params = map[string][]string{
		"results": {"5"},
		"text":    {query.Text}}
	return
}

type SongSearchQuery struct {
	queryBase
	Title    string
	Artist   string
	Response SongsResponse
}

func (query *SongSearchQuery) GetCallInfo() (path string, params map[string][]string) {
	path = "/song/search"
	params = map[string][]string{
		"results": {"5"},
		"bucket":  {"tracks", "id:spotify-WW"},
		"limit":   {"true"},
		"title":   {query.Title},
		"artist":  {query.Artist}}
	return
}

type Api struct {
	Key string
}

const (
	BASE_URL = "http://developer.echonest.com/api/v4"
)

func GetApi(key string) *Api {
	return &Api{key}
}

func (api *Api) call(query query) {
	path, params := query.GetCallInfo()

	v := url.Values{}
	v.Set("api_key", api.Key)
	for key, values := range params {
		for _, value := range values {
			v.Add(key, value)
		}
	}

	resp, err := http.Get(fmt.Sprintf("%s%s?%s", BASE_URL, path, v.Encode()))
	if err != nil {
		query.setError(err)
		return
	}
	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err != nil {
		query.setError(err)
	} else {
		json.Unmarshal(data, &query)
	}
}

func (api *Api) ArtistExtract(text string, out chan *ArtistExtractQuery) {
	query := &ArtistExtractQuery{
		Text: text}

	api.call(query)
	out <- query
}

func (api *Api) SongSearch(title, artist string, out chan *SongSearchQuery) {
	query := &SongSearchQuery{
		Title:  title,
		Artist: artist}

	api.call(query)
	out <- query
}
