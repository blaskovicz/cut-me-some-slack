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

func serveHomeFunc(port string) func(w http.ResponseWriter, r *http.Request) {
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
		wsDomain := os.Getenv("HEROKU_APP_DOMAIN")
		if wsDomain == "" {
			wsDomain = fmt.Sprintf("localhost:%s", port)
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
	slackToken := os.Getenv("SLACK_TOKEN") // TODO validate scopes
	if slackToken == "" {
		panic("env.SLACK_TOKEN cannot be empty")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	allowedChannels := os.Getenv("SLACK_CHANNEL")
	if allowedChannels == "" {
		allowedChannels = "api-testing"
	}

	// start hub
	hub, err := chat.NewHub(slackToken, allowedChannels)
	if err != nil {
		log.Fatal("Failed to launch hub: %s", err)
	}
	go hub.Run()

	// start server
	http.HandleFunc("/", serveHomeFunc(port))
	http.HandleFunc("/stream", startStreamFunc(hub))
	log.Printf("starting server on :%s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
