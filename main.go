package main

import (
	"./echonest"
	"./twitter"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var stats = make(map[string]int)

func resolve(url string) {
	resp, err := http.Head(url)
	if err != nil {
		fmt.Println("ERROR!!!", err.Error())
		return
	}
	if resp.StatusCode != 200 {
		return
	}
	url = resp.Request.URL.String()

	p := strings.Split(url, "/")

	// Skip links that cannot be to a specific resource
	if len(p) < 4 || len(p[3]) == 0 {
		return
	}

	stats[p[2]]++
	//fmt.Printf("%#v\n", stats)

	fmt.Println("URL:", url)
	if strings.HasPrefix(url, "http://www.last.fm") {
		pieces := strings.Split(url[19:], "/")
		switch len(pieces) {
		case 1:
			fmt.Println("Artist:", pieces[0])
		case 2:
			fmt.Println("Album:", pieces[1], "by", pieces[0])
		default:
			fmt.Println("Track:", pieces[2], "by", pieces[0])
		}
	}
}

func main() {
	matchNowPlaying := regexp.MustCompile(`(?:\A| )"?(\pL[\pL!'.-]*(?: \pL[\pL!'.-]*)*)"?\s*(?:'s|[♪–—~|:-]+|by)\s*"?(\pL[\pL!'.-]*(?: \pL[\pL!'.-]*)*)"?(?: |\z)`)
	cleanTweet := regexp.MustCompile(`\A(?i:now playing|listening to|escuchando a)|on (album|#)|del àlbum`)

	en := echonest.GetApi("<echo nest key>")

	// Get the Twitter Stream API
	stream := &twitter.StreamApi{"username", "password"}
	// Create a channel that will be receiving tweets
	tweets := make(chan *twitter.Tweet)
	// Start streaming tweets in a goroutine
	go stream.StatusesFilter([]string{"#nowplaying", "open.spotify.com", "spoti.fi"}, tweets)
	// Fetch tweets as they come in
	for tweet := range tweets {
		fmt.Printf("@%s (%d followers)\n", tweet.User.Screen_name, tweet.User.Followers_count)
		fmt.Println(tweet.Text)
		cleaned := cleanTweet.ReplaceAllString(tweet.Text, "$")
		if matches := matchNowPlaying.FindStringSubmatch(cleaned); matches != nil {
			results := make(chan *echonest.SongSearchQuery)
			go en.SongSearch(matches[1], matches[2], results)
			go en.SongSearch(matches[2], matches[1], results)

			artistResult := make(chan *echonest.ArtistExtractQuery)
			go en.ArtistExtract(matches[1]+" "+matches[2], artistResult)
			qry := <-artistResult
			for _, artist := range qry.Response.Artists {
				fmt.Println("=>", artist.Name)
			}

			doOne := func(query *echonest.SongSearchQuery) {
				fmt.Println("~~~~~")
				fmt.Printf("title: %#v, artist: %#v\n", query.Title, query.Artist)
				if query.Error != nil {
					fmt.Println("FAILED:", query.Error.Error())
					return
				}
				for _, song := range query.Response.Songs {
					fmt.Println("-", song.Title, "by", song.Artist_name)
					uri := strings.Replace(song.Tracks[0].Foreign_id, "-WW", "", 1)
					fmt.Println("  =>", uri)
				}
			}

			doOne(<-results)
			doOne(<-results)
		}
		for _, url := range tweet.Entities.Urls {
			resolve(url.Expanded_url)
		}
		fmt.Println("----")
	}
}
