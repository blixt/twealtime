package twitter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Entities struct {
	Hashtags []HashtagEntity
	Urls     []UrlEntity
}

type HashtagEntity struct {
	Text    string
	Indices []int
}

type StreamApi struct {
	Username string
	Password string
}

type Tweet struct {
	Id          uint64
	User        User
	Entities    Entities
	Coordinates interface{}
	Text        string
}

type UrlEntity struct {
	Url          string
	Expanded_url string
	Indices      []int
}

type User struct {
	Id                uint64
	Name              string
	Screen_name       string
	Profile_image_url string
	Followers_count   int
	Friends_count     int
	Listed_count      int
}

const (
	BASE_STREAM_URL = "https://stream.twitter.com/1"
)

func parseTweet(reader *bufio.Reader) (tweet *Tweet, err error) {
	var (
		part   []byte
		prefix bool
	)

	buffer := new(bytes.Buffer)

	for {
		if part, prefix, err = reader.ReadLine(); err != nil {
			break
		}

		buffer.Write(part)
		if !prefix {
			err = json.Unmarshal(buffer.Bytes(), &tweet)
			break
		}
	}

	return
}

func (api *StreamApi) stream(path string, params url.Values, tweets chan *Tweet) {
	client := &http.Client{}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s%s.json", BASE_STREAM_URL, path),
		strings.NewReader(params.Encode()))

	if err != nil {
		panic(err.Error())
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(api.Username, api.Password)

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		panic(err.Error())
	}

	reader := bufio.NewReader(resp.Body)
	for {
		tweet, err := parseTweet(reader)
		if err != nil {
			close(tweets)
			break
		}
		tweets <- tweet
	}
}

func (api *StreamApi) StatusesFilter(track []string, tweets chan *Tweet) {
	api.stream("/statuses/filter", url.Values{"track": track}, tweets)
}
