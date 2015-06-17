package ytmp3

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)
import "github.com/jpillora/ytmp3/ytdl"
import "github.com/jpillora/ytmp3/static"
import "github.com/jpillora/go-realtime"

type Config struct {
	Foo       int    `help:"foo"`
	YoutubeDL string `help:"path to youtube-dl"`
	FFMPEG    string `help:"path to ffmpeg"`
}

func New(c Config) http.Handler {

	if c.YoutubeDL == "" {
		c.YoutubeDL = "youtube-dl"
	}
	// if c.FFMPEG == "" {
	// 	c.FFMPEG = "ffmpeg"
	// }

	m := http.NewServeMux()

	y := &ytHandler{
		Config:   c,
		ServeMux: m,
		fs:       static.FileSystemHandler(),
	}

	y.rt, _ = realtime.Sync(y.shared)

	if err := ytdl.Check(c.YoutubeDL); err != nil {
		log.Fatalf("youtube-dl check failed: %s", err)
	}

	// if _, err := exec.LookPath(c.FFMPEG); err != nil {
	// 	log.Fatal("cannot locate %s", c.FFMPEG)
	// }

	return http.HandlerFunc(y.handle)
}

type ytHandler struct {
	Config
	*http.ServeMux
	fs http.Handler
	//state
	rt      *realtime.Realtime
	joblock sync.Mutex
	shared  struct {
		Running bool
		Total   int
		Job     struct {
			State string
			ID    string
		}
	}
}

func (y *ytHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/realtime" {
		y.rt.ServeHTTP(w, r)
		return
	} else if r.URL.Path == "/realtime.js" {
		realtime.JS.ServeHTTP(w, r)
		return
	} else if r.URL.Path == "/ytdl" {
		y.ytdl(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/mp3/") {
		r.URL.Path = r.URL.Path[5:]
		y.mp3(w, r)
		return
	}
	y.fs.ServeHTTP(w, r)
}

var ytid = regexp.MustCompile(`^[A-Za-z0-9\-\_]{11}$`)

func (y *ytHandler) ytdl(w http.ResponseWriter, r *http.Request) {
	cmd := r.URL.Query().Get("cmd")
	if cmd == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("cmd missing"))
		return
	}
	args := strings.Split(cmd, " ")
	out, err := ytdl.Run(args...)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error() + "\n"))
	}
	w.Write(out)
}

func (y *ytHandler) mp3(w http.ResponseWriter, r *http.Request) {

	id := ""
	if ytid.MatchString(r.URL.Path) {
		id = r.URL.Path
	} else if v := r.URL.Query().Get("v"); ytid.MatchString(v) {
		id = v
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing ID"))
		return
	}

	out, err := ytdl.Run("-f", "140", "-g", "https://www.youtube.com/watch?v="+id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to retrieve content URLs\n" + string(out)))
		return
	}

	videoURL := ""
	audioURL := ""
	for _, s := range strings.Split(string(out), "\n") {
		if s == "" {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			log.Printf("Not a valid URL: %s", s)
			continue
		}
		mime := u.Query().Get("mime")
		switch mime {
		case "video/mp4":
			videoURL = s
		case "audio/mp4":
			audioURL = s
		default:
			log.Printf("Unhandled mime type: %s", mime)
		}
	}

	//if we have audio, proxy straight through
	if audioURL != "" {
		resp, err := http.Get(audioURL)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Invalid audio URL: " + err.Error()))
			return
		}
		w.Header().Set("Accept-Ranges", resp.Header.Get("Accept-Ranges"))
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		io.Copy(w, resp.Body)
		return
	}

	//no audio and video, exit
	if videoURL == "" {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Failed to extract content URLs"))
		return
	}

	//others wait here
	// y.joblock.Lock()
	//start!
	// y.shared.Running = true
	// y.shared.Total++
	// y.shared.Job.State = "VIDEO"
	// y.shared.Job.ID = id
	// y.rt.Update()

	// _release := func() {
	// 	y.shared.Running = false
	// 	y.rt.Update()
	// 	y.joblock.Unlock()
	// }

	// var s sync.Once{}
	// release := func() {
	// 	s.Do(release)
	// }

	//video! must extract audio out
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Video to audio transcoding not implemented yet"))
	return
}
