package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/blaskovicz/cut-me-some-slack/chat"
)

func startStreamFunc(hub *chat.Hub) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving /stream")
		chat.ServeWs(hub, w, r)
	}
}

func serveHomeFunc(cfg *chat.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving /")
		indexFile, err := os.Open("cmd/cut-me-some-slack/index.html")
		if err != nil {
			log.Printf("error: %s\n", err)
			http.Error(w, "Server Error", 500)
			return
		}
		index, err := ioutil.ReadAll(indexFile)
		if err != nil {
			log.Printf("error: %s\n", err)
			http.Error(w, "Server Error", 500)
			return
		}
		indexTemplate, err := template.New("index").Parse(string(index))
		if err != nil {
			log.Printf("error: %s\n", err)
			http.Error(w, "Server Error", 500)
			return
		}

		wsProto := "ws"
		wsDomain := cfg.Server.Domain
		if wsDomain == "localhost" {
			wsDomain += fmt.Sprintf(":%d", cfg.Server.Port)
		} else {
			wsProto += "s"
		}

		err = indexTemplate.Execute(w, struct{ WebSocketURI string }{WebSocketURI: fmt.Sprintf("%s://%s/stream", wsProto, wsDomain)})
		if err != nil {
			log.Printf("error: %s\n", err)
			http.Error(w, "Server Error", 500)
			return
		}
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
	http.HandleFunc("/", serveHomeFunc(cfg))
	http.HandleFunc("/stream", startStreamFunc(hub))
	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", listenAddr)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
