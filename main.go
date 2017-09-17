package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/eduncan911/podcast"
	"github.com/go-redis/redis"

	"github.com/gorilla/mux"
	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

type YtInMp3Response struct {
	Title  string `json:"title"`
	Result struct {
		Num140 string `json:"140"`
	} `json:"result"`
	Subtitle struct {
	} `json:"subtitle"`
	Status bool `json:"status"`
}

type ytAPI struct {
	service    *youtube.Service
	feed       podcast.Podcast
	channelID  string
	playlistID string
}

var client *redis.Client
var port = os.Getenv("PORT")
var api_key = os.Getenv("API_KEY")

func init() {

	redisURL := os.Getenv("REDISTOGO_URL")
	if redisURL == "" {
		log.Fatalln("$REDISTOGO_URL not set")
	}

	if port == "" {
		log.Fatalln("$PORT not set")
	}

	if api_key == "" {
		log.Fatalln("$API_KEY not set")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("error in parsing $REDISTOGO_URL: %v", err)
	}
	client = redis.NewClient(opt)

	err = client.Ping().Err()
	if err != nil {
		log.Fatalf("error in connecting to redis: %v", err)
	}
}

func main() {

	go func() {
		for {
			seedPodcasts()
			time.Sleep(1 * time.Minute)
		}
	}()

	router := mux.NewRouter()

	router.Handle("/", http.RedirectHandler("/feed", http.StatusTemporaryRedirect))

	router.HandleFunc("/feed", serveFeed)

	router.HandleFunc("/dl/{videoid}.mp3", servePodcast)

	http.ListenAndServe(fmt.Sprintf(":%s", port), router)
}

func seedPodcasts() {

	var err error
	yt := &ytAPI{}
	yt.service, err = youtube.New(&http.Client{
		Transport: &transport.APIKey{Key: api_key},
	})
	if err != nil {
		log.Fatalf("error in creating service: %v", err)
	}

	yt.channelID = "UC_BzFbxG2za3bp5NRRRXJSw"
	yt.playlistID = "PL64wiCrrxh4Jisi7OcCJIUpguV_f5jGnZ"
	channel, err := yt.fetchChannelDetails()
	if err != nil {
		log.Println(err)
		return
	}

	title := channel.Items[0].Snippet.Title
	desc := channel.Items[0].Snippet.Description
	u := fmt.Sprintf("https://youtube.com/playlist?list=%s", yt.playlistID)
	pubAt, err := time.Parse(time.RFC3339, channel.Items[0].Snippet.PublishedAt)
	if err != nil {
		log.Println(err.Error())
		return
	}
	cover := channel.Items[0].Snippet.Thumbnails.High.Url
	currTime := time.Now()

	yt.feed = podcast.New(title, u, desc, &pubAt, &currTime)
	yt.feed.AddImage(cover)

	playlistItems, err := yt.fetchPlaylistDetails()
	if err != nil {
		log.Println(err)
	}

	for _, v := range playlistItems {
		t, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
		if err != nil {
			log.Println(err)
		}
		item := podcast.Item{
			Title:       v.Snippet.Title,
			GUID:        v.ContentDetails.VideoId,
			Source:      fmt.Sprintf("https://youtube.com/watch?v=%s", v.ContentDetails.VideoId),
			Link:        fmt.Sprintf("https://justforfunc.herokuapp.com/dl/%s.mp3", v.ContentDetails.VideoId),
			Description: v.Snippet.Description,
			PubDate:     &t,
		}
		yt.feed.AddItem(item)
	}
	a := yt.feed.String()
	err = client.Set("rss_feed", a, time.Duration(0)).Err()
	if err != nil {
		log.Printf("error in saving rss feed: %v", err)
	}
}

func serveFeed(w http.ResponseWriter, r *http.Request) {
	val, err := client.Get("rss_feed").Result()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(val))
}

func servePodcast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["videoid"]

	if id == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	absFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.mp3", id))
	f, err := os.OpenFile(absFilePath, os.O_RDONLY, 0x666)
	if err != nil {
		if os.IsNotExist(err) {

			// Bug: When more than one requests come asking for the same podcast, It'll redownload them
			// TODO: Fix it..
			resp, err := fetchMP3File(id)
			if err != nil {
				log.Printf("error in downloading %s: %v", id, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			nf, err := os.Create(absFilePath)
			if err != nil {
				log.Printf("error in creating new file: %v", err)
			}
			defer nf.Close()

			writers := io.MultiWriter(w, nf)

			io.Copy(writers, resp.Body)

			return
		}
		log.Printf("error in fetching %s: %v", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	io.Copy(w, f)
}

func (yt *ytAPI) fetchPlaylistDetails() ([]*youtube.PlaylistItem, error) {

	videoIDs := []*youtube.PlaylistItem{}

	playlistResp, err := fetchPlaylistItems(yt, "")
	if err != nil {
		return nil, err
	}

	for _, v := range playlistResp.Items {

		// Buggy code, It has to be fixed if you want to make a podcast of a playlist with 50+ items
		// for {
		// 	if playlistResp.NextPageToken == "" {
		// 		break
		// 	}
		// 	playlistResp, err := fetchPlaylistItems(yt, playlistResp.NextPageToken)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	for _, v2 := range playlistResp.Items {
		// 		videoIDs = append(videoIDs, v2)
		// 	}
		// }
		videoIDs = append(videoIDs, v)
	}

	return videoIDs, nil
}

func fetchPlaylistItems(yt *ytAPI, nextPageToken string) (*youtube.PlaylistItemListResponse, error) {

	if nextPageToken != "" {
		call := yt.service.PlaylistItems.List("contentDetails,snippet").PlaylistId(yt.playlistID).MaxResults(50).PageToken(nextPageToken)
		return call.Do()
	}
	call := yt.service.PlaylistItems.List("contentDetails,snippet").PlaylistId(yt.playlistID).MaxResults(50)
	return call.Do()
}

func (yt *ytAPI) fetchChannelDetails() (*youtube.ChannelListResponse, error) {
	call := yt.service.Channels.List("snippet").Id(yt.channelID)

	return call.Do()
}

func fetchMP3File(id string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://api.youtubemultidownloader.com/video?id=%s", id), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ytdlResp := &YtInMp3Response{}

	err = json.NewDecoder(resp.Body).Decode(ytdlResp)
	if err != nil {
		return nil, err
	}

	dlResp, err := http.Get(ytdlResp.Result.Num140)
	if err != nil {
		return nil, err
	}

	return dlResp, nil
}
