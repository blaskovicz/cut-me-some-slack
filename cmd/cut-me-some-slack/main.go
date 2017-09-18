package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/blaskovicz/cut-me-some-slack/chat"
)

func startStreamFunc(cfg *chat.Config, hub *chat.Hub) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving /stream")
		chat.ServeWs(cfg, hub, w, r)
	}
}

func main() {
	// load env
	cfg, err := chat.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	// start hub
	hub, err := chat.NewHub(cfg)
	if err != nil {
		log.Fatal("Failed to launch hub: %s", err)
	}
	go hub.Run()

	// stream is mapped to websocket conn
	http.HandleFunc("/stream", startStreamFunc(cfg, hub))
	// anything starting with /static goes to ui/build dir (eg: /static/foo -> ui/build/static/foo)
	http.Handle("/static/", http.FileServer(http.Dir("ui/build")))
	// lastly, re-map anything else directly to the index.html page for single page routing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO add service-worker.js and any other special files in here that
		// don't get copied to static if needed
		http.ServeFile(w, r, "ui/build/index.html")
	})

	// start server
	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", listenAddr)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
