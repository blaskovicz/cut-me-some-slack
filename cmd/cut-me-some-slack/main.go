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

	// start server
	http.HandleFunc("/stream", startStreamFunc(cfg, hub))
	http.Handle("/", http.FileServer(http.Dir("ui/build")))

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", listenAddr)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
