package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/jpillora/opts"
	"github.com/jpillora/ytmp3/handler"
)

var VERSION = "0.0.0"

type Config struct {
	Port int `help:"Listening port" env:"PORT"`
}

func main() {
	o := Config{
		Port: 3000,
	}
	opts.New(&o).
		Name("ytmp3").
		Repo("github.com/jpillora/ytmp3").
		Version(VERSION).
		Parse()
	//create an http.Handler
	yt := ytmp3.New(ytmp3.Config{})

	log.Printf("listening on %d...", o.Port)
	http.ListenAndServe(":"+strconv.Itoa(o.Port), yt)
}
