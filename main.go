package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eduncan911/podcast"
	"github.com/go-redis/redis"

	"github.com/gorilla/mux"
	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

type YT1SResponse struct {
	Title  string `json:"title"`
	Result struct {
		Num17  string `json:"17"`
		Num18  string `json:"18"`
		Num22  string `json:"22"`
		Num36  string `json:"36"`
		Num43  string `json:"43"`
		Num133 string `json:"133"`
		Num134 string `json:"134"`
		Num135 string `json:"135"`
		Num136 string `json:"136"`
		Num137 string `json:"137"`
		Num140 string `json:"140"`
		Num160 string `json:"160"`
		Num171 string `json:"171"`
		Num242 string `json:"242"`
		Num243 string `json:"243"`
		Num244 string `json:"244"`
		Num247 string `json:"247"`
		Num248 string `json:"248"`
		Num249 string `json:"249"`
		Num250 string `json:"250"`
		Num251 string `json:"251"`
		Num278 string `json:"278"`
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
var PORT = os.Getenv("PORT")
var API_KEY = os.Getenv("API_KEY")

func init() {

	redisURL := os.Getenv("REDISTOGO_URL")
	if redisURL == "" {
		log.Fatalln("$REDISTOGO_URL not set")
	}

	if PORT == "" {
		log.Fatalln("$PORT not set")
	}

	if API_KEY == "" {
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
			time.Sleep(28 * time.Minute)
		}
	}()

	router := mux.NewRouter()

	router.Handle("/", http.RedirectHandler("/feed", http.StatusTemporaryRedirect))

	router.HandleFunc("/feed", serveFeed)

	router.HandleFunc("/dl/{videoid}.mp3", servePodcast)

	http.ListenAndServe(fmt.Sprintf(":%s", PORT), router)
	// 	fmt.Println(v)
	// 	u, _ := fetchMP3File(v)

	// 	ur, _ := url.Parse(fmt.Sprintf("https://api.youtubemultidownloader.com%s", u))

	// 	meta, err := pluto.FetchMeta(ur)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	f, _ := os.Create(meta.Name)

	// 	err = pluto.Download(&pluto.Config{
	// 		Meta:   meta,
	// 		Parts:  50,
	// 		Writer: f,
	// 	})
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// }
}

func seedPodcasts() {

	var err error
	yt := &ytAPI{}
	yt.service, err = youtube.New(&http.Client{
		Transport: &transport.APIKey{Key: API_KEY},
	})
	if err != nil {
		log.Fatalf("error in creating service: %v", err)
	}

	yt.channelID = "UC_BzFbxG2za3bp5NRRRXJSw"
	yt.playlistID = "PL64wiCrrxh4Jisi7OcCJIUpguV_f5jGnZ"
	channel, err := yt.fetchChannelDetails()
	if err != nil {
		log.Println(err)
	}

	title := channel.Items[0].Snippet.Title
	desc := channel.Items[0].Snippet.Description
	u := fmt.Sprintf("https://youtube.com/playlist?list=%s", yt.playlistID)
	pubAt, err := time.Parse(time.RFC3339, channel.Items[0].Snippet.PublishedAt)
	if err != nil {
		log.Println(err.Error())
	}
	currTime := time.Now()
	podcastRSS := podcast.New(title, u, desc, &pubAt, &currTime)

	yt.feed = podcastRSS

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
			Link:        fmt.Sprintf("https://afternoon-brook-65479.herokuapp.com/dl/%s.mp3", v.ContentDetails.VideoId),
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

}

func (yt *ytAPI) fetchPlaylistDetails() ([]*youtube.PlaylistItem, error) {

	videoIDs := []*youtube.PlaylistItem{}

	playlistResp, err := fetchPlaylistItems(yt, "")
	if err != nil {
		return nil, err
	}

	for _, v := range playlistResp.Items {

		// for {
		// 	if playlistResp.NextPageToken == "" {
		// 		break
		// 	}

		// 	playlistResp, err := fetchPlaylistItems(yt, playlistResp.NextPageToken)
		// 	if err != nil {
		// 		return nil, err
		// 	}

		// 	fmt.Println("fetching more videos")
		// 	for _, v2 := range playlistResp.Items {
		// 		videoIDs = append(videoIDs, v2.ContentDetails.VideoId)
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

func fetchMP3File(id string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.youtubemultidownloader.com/video?id=%s", id), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.101 Safari/537.36")
	req.Header.Add("referer", "https://youtubemultidownloader.com/index.html")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ytdlResp := &YT1SResponse{}

	err = json.NewDecoder(resp.Body).Decode(ytdlResp)
	if err != nil {
		return "", err
	}

	return ytdlResp.Result.Num22, nil
}
