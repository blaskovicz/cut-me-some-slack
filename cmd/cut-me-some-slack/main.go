package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/blaskovicz/cut-me-some-slack/chat"
	"github.com/julienschmidt/httprouter"
)

func startStreamFunc(cfg *chat.Config, hub *chat.Hub) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	// load our index.html file
	indexTemplate, err := template.ParseFiles("ui/build/index.html")
	if err != nil {
		log.Fatal("Failed to load index.html: %s", err)
	}
	configBytes, err := json.Marshal(&cfg)
	if err != nil {
		log.Fatal("Failed to marshal config: %s", err)
	}

	// start hub
	hub, err := chat.NewHub(cfg)
	if err != nil {
		log.Fatal("Failed to launch hub: %s", err)
	}
	go hub.Run()

	router := httprouter.New()
	// stream is mapped to websocket conn
	router.GET("/stream", startStreamFunc(cfg, hub))
	// anything starting with /static goes to ui/build dir (eg: /static/foo -> ui/build/static/foo)
	router.ServeFiles("/static/*filepath", http.Dir("ui/build/static/"))
	// map anything else directly to the index.html page for single page routing
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("serving index.html")
		// TODO add service-worker.js and any other special files in here that
		// don't get copied to static if needed
		if err := indexTemplate.Execute(w, configBytes); err != nil {
			log.Printf("warning: failed to execute index template - %s", err)
		}
	})

	// start server
	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", listenAddr)
	err = http.ListenAndServe(listenAddr, router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
