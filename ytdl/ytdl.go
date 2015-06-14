package ytdl

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

var path = "/usr/local/bin/youtube-dl"

func Check(p string) error {

	if info, err := os.Stat(p); err == nil {
		if info.IsDir() {
			return fmt.Errorf("Invalid path: %s is a directory", p)
		}
		return nil
	}

	log.Printf("downloading youtube-dl...")

	htmlb, err := http.Get("https://rg3.github.io/youtube-dl/download.html")
	if err != nil {
		return fmt.Errorf("Cannot fetch manifest")
	}
	defer htmlb.Body.Close()
	if err != nil {
		return fmt.Errorf("Cannot fetch manifest content")
	}

	html, err := ioutil.ReadAll(htmlb.Body)

	var link = regexp.MustCompile(`(https:\/\/yt-dl\.org\/downloads\/[\d\.]+/youtube-dl)`)
	m := link.FindStringSubmatch(string(html))
	if len(m) == 0 {
		return fmt.Errorf("Cannot fetch link")
	}

	url := m[1]
	bin, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Cannot fetch bin")
	}
	defer bin.Body.Close()

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("Cannot create file: %s", err)
	}
	defer f.Close()
	if err := f.Chmod(0777); err != nil {
		return fmt.Errorf("Cannot set perms")
	}

	io.Copy(f, bin.Body)

	path = p

	if out, err := Run("--version"); err != nil {
		return fmt.Errorf("Failed to run youtube-dl: %s", err)
	} else if !regexp.MustCompile(`^[\d\.]+`).Match(out) {
		return fmt.Errorf("Failed to run youtube-dl: %s", out)
	}

	return nil
	// https: //rg3.github.io/youtube-dl/download.html
}

func Run(args ...string) ([]byte, error) {
	return exec.Command(path, args...).Output()
}
