package main

import (
	"./spotify"
	"./twealtime"
	"./twitter"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func getSpotifyUri(urlString string) (uri string) {
	resp, err := http.Head(urlString)
	if err != nil || resp.StatusCode != 200 {
		return
	}

	info, _ := url.Parse(resp.Request.URL.String())
	if info.Host == "open.spotify.com" {
		parts := strings.Split(info.Path, "/")
		if len(parts) != 3 || parts[1] != "track" {
			return
		}
		uri = "spotify:track:" + parts[2]
	}

	return
}

func main() {
	// Regexp that cleans up a tweet before scanning it for artist/track names.
	cleanTweet := regexp.MustCompile(`\A(?i:now playing|listening to|escuchando a)|on (album|#)|del àlbum`)
	// Regexp that finds artist/track names in a tweet.
	matchNowPlaying := regexp.MustCompile(`(?:\A| )"?(\pL[\pL!'.-]*(?: \pL[\pL!'.()-]*)*)"?\s*(?:'s|[♪–—~|:/-]+|by)\s*"?(\pL[\pL!'.-]*(?: \pL[\pL!'.()-]*)*)"?(?:[ #]|\z)`)

	// Create a Spotify API object for searching the Spotify catalog.
	sp := spotify.GetApi()

	// Create a web socket server that will be serving the data to other apps.
	server := twealtime.NewServer()
	go server.Serve(":1337")

	// Get the Twitter Stream API.
	twitterStream := &twitter.StreamApi{"username", "password"}
	// Create a channel that will be receiving tweets.
	tweets := make(chan *twitter.Tweet, 100)
	// Start streaming tweets in a goroutine.
	go twitterStream.StatusesFilter([]string{"#spotify", "#nowplaying", "open.spotify.com", "spoti.fi"}, tweets)
	// Fetch tweets as they come in.
	for tweet := range tweets {
		cleaned := cleanTweet.ReplaceAllString(tweet.Text, "$")

		// TODO: Make this nicer than an anonymous goroutine.
		go func(tweet *twitter.Tweet) {
			// First of all, try to find a Spotify URL directly in the tweet.
			var uri string
			for _, url := range tweet.Entities.Urls {
				uri = getSpotifyUri(url.ExpandedUrl)
				if uri != "" {
					break
				}
			}

			// Find artist and track name in the tweet.
			matches := matchNowPlaying.FindStringSubmatch(cleaned)
			// Only do this if we didn't find a URI already.
			if uri == "" && matches != nil {
				// Spotify search query format.
				const format = `title:"%s" AND artist:"%s"`

				// Create a channel for receiving results.
				results := make(chan *spotify.SearchTrackQuery)
				// Send off two simultaneous search requests to the Spotify search API, trying artist/track and the reverse
				// (since we don't know if people wrote "Track - Artist" or "Artist - Track")
				go sp.SearchTrack(fmt.Sprintf(format, matches[1], matches[2]), results)
				go sp.SearchTrack(fmt.Sprintf(format, matches[2], matches[1]), results)
				// Wait for the results to come in.
				result1 := <-results
				result2 := <-results

				if result1.Error != nil || result2.Error != nil {
					fmt.Println("!!", tweet.Text)
					fmt.Println()
					return
				}

				// Get the track of the result with the most results (which is most likely to be the correct one.)
				if result1.Info.NumResults > result2.Info.NumResults {
					uri = result1.Tracks[0].Href
				} else if result2.Info.NumResults > result1.Info.NumResults {
					uri = result2.Tracks[0].Href
				}
			}

			// No URI was found; don't do anything.
			if uri == "" {
				fmt.Println(":/", tweet.Text)
				fmt.Println()
				return
			}

			// Send tweet info and track URI through the web socket server.
			fmt.Println("<<", tweet.Text)
			fmt.Println(">>", uri)
			fmt.Println()

			server.Send(twealtime.TrackMention{
				Tweet:            tweet.Text,
				TwitterUser:      tweet.User.ScreenName,
				TwitterFollowers: tweet.User.FollowersCount,
				TrackUri:         uri,
			})
		}(tweet)
	}
}
