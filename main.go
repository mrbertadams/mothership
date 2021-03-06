package main

import (
	"flag"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"gopkg.in/airbrake/glog.v2"
	"gopkg.in/airbrake/gobrake.v2"

	"github.com/zefer/mothership/handlers"
	"github.com/zefer/mothership/mpd"
	"github.com/zefer/mothership/websocket"
)

var (
	client  *mpd.Client
	mpdAddr = flag.String("mpdaddr", "127.0.0.1:6600", "MPD address")
	port    = flag.String("port", ":8080", "listen port")

	abProjectID = flag.Int64("abprojectid", 0, "Airbrake project ID")
	abApiKey    = flag.String("abapikey", "", "Airbrake API key")
	abEnv       = flag.String("abenv", "development", "Airbrake environment name")
)

func main() {
	flag.Parse()
	defer glog.Flush()
	glog.Infof("Starting API for MPD at %s.", *mpdAddr)

	if *abProjectID > int64(0) && *abApiKey != "" {
		airbrake := gobrake.NewNotifier(*abProjectID, *abApiKey)
		defer airbrake.Close()
		defer airbrake.NotifyOnPanic()
		airbrake.AddFilter(func(n *gobrake.Notice) *gobrake.Notice {
			n.Context["environment"] = *abEnv
			return n
		})
		glog.Gobrake = airbrake
	}

	// Send the browser the MPD state when they first connect.
	websocket.OnConnect(func(c *websocket.Conn) {
		b, err := mpdStatusJSON(client.C)
		if err == nil {
			c.Send(b)
		}
	})

	// This watcher notifies us when MPD's state changes, without polling.
	watch := mpd.NewWatcher(*mpdAddr)
	defer watch.Close()
	// When mpd state changes, broadcast it to all websockets.
	watch.OnStateChange(func(s string) {
		glog.Info("MPD state change in subsystem: ", s)
		b, err := mpdStatusJSON(client.C)
		if err != nil {
			glog.Errorln(err)
			return
		}
		websocket.Broadcast(b)
	})

	// This client connection provides an API to MPD's commands.
	client = mpd.NewClient(*mpdAddr)
	defer client.Close()

	http.HandleFunc("/websocket", websocket.Serve)
	http.Handle("/next", handlers.NextHandler(client))
	http.Handle("/previous", handlers.PreviousHandler(client))
	http.Handle("/play", handlers.PlayHandler(client))
	http.Handle("/pause", handlers.PauseHandler(client))
	http.Handle("/randomOn", handlers.RandomOnHandler(client))
	http.Handle("/randomOff", handlers.RandomOffHandler(client))
	http.Handle("/files", handlers.FileListHandler(client))
	http.Handle("/playlist", handlers.PlayListHandler(client))
	http.Handle("/library/updated", handlers.LibraryUpdateHandler(client))

	// The front-end assets are served from a go-bindata file.
	http.Handle("/", http.FileServer(&assetfs.AssetFS{
		Asset: Asset, AssetDir: AssetDir, Prefix: "",
	}))

	glog.Infof("Listening on %s.", *port)
	err := http.ListenAndServe(*port, nil)
	if err != nil {
		glog.Errorf("http.ListenAndServe %s failed: %s", *port, err)
		return
	}
}
