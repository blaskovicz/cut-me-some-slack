package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nlopes/slack"
)

// currently for one channel, TODO
type MessageBus struct {
	o int64
	m []*slack.MessageEvent
	sync.RWMutex
}

func (m *MessageBus) Read(since int64) ([]*slack.MessageEvent, int64) {
	// TODO block until info is avail here rather than busy waiting
	m.RLock()
	defer m.RUnlock()
	if since > m.o {
		return nil, m.o + 1
	}
	return m.m[since:], m.o + 1
}
func (m *MessageBus) Write(message *slack.MessageEvent) {
	m.Lock()
	defer m.Unlock()
	m.o++
	m.m = append(m.m, message)
	fmt.Printf("[%d] %#v\n", m.o, *message)
	// TODO expire
}

func NewMessageBus() *MessageBus {
	return &MessageBus{m: []*slack.MessageEvent{}, o: -1}
}

func main() {
	restrictToChannel := os.Getenv("SLACK_CHANNEL")
	if restrictToChannel == "" {
		restrictToChannel = "api-testing"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	m := NewMessageBus()
	token := os.Getenv("SLACK_TOKEN") // TODO validate scopes
	if token == "" {
		panic("env.SLACK_TOKEN cannot be empty")
	}
	api := slack.New(token)
	var teamInfo *slack.Info
	channels, err := api.GetChannels(true)
	if err != nil {
		panic(err)
	}

	var channel *slack.Channel
	for _, c := range channels {
		if c.Name == restrictToChannel {
			fmt.Printf("Channel %#v\n", c)
			channel = &c
			break
		}
	}
	if channel == nil {
		panic(fmt.Errorf("Couldn't find channel %s", restrictToChannel))
	}
	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//logger.SetLevel()
	//slack.SetLogger(logger)
	//api.SetDebug(true)
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	rtm := api.NewRTM()
	indexFile, err := os.Open("cmd/web/index.html")
	if err != nil {
		fmt.Println(err)
	}
	index, err := ioutil.ReadAll(indexFile)
	if err != nil {
		panic(err)
	}
	indexTemplate, err := template.New("index").Parse(string(index))
	if err != nil {
		panic(err)
	}
	go rtm.ManageConnection()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wsProto := "ws"
		wsDomain := os.Getenv("HEROKU_APP_DOMAIN")
		if wsDomain == "" {
			wsDomain = fmt.Sprintf("localhost:%s", port)
		} else {
			wsProto += "s"
		}

		err := indexTemplate.Execute(w, struct{ WebSocketURI string }{WebSocketURI: fmt.Sprintf("%s://%s/stream", wsProto, wsDomain)})
		if err != nil {
			fmt.Println(err)
		}
	})

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		// TODO disallow other client subs from same ip
		fmt.Println("Client subscribed")
		var buff bytes.Buffer
		err = json.NewEncoder(&buff).Encode(struct {
			Type    string `json:"type"`
			Slack   string `json:"slack"`
			Channel string `json:"channel"`
		}{Slack: teamInfo.Team.Name, Type: "team-info", Channel: "#" + channel.Name})
		if err != nil {
			// TODO way better error handling everywhere
			fmt.Println(err)
			return
		}
		conn.WriteMessage(websocket.TextMessage, buff.Bytes())
		o := int64(0) // TODO start at messag bus offset?
		connClosed := false
		go func() {
			for {
				if connClosed {
					break
				}
				messages, newO := m.Read(o)
				if messages != nil {
					o = newO
					for i, _ := range messages {
						m := messages[i]
						var buff bytes.Buffer
						err = json.NewEncoder(&buff).Encode(struct {
							Type    string              `json:"type"`
							Message *slack.MessageEvent `json:"message"`
						}{Type: "message", Message: m})
						if err != nil {
							fmt.Println(err)
							continue
						}
						conn.WriteMessage(websocket.TextMessage, buff.Bytes())
					}
				} else {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
		defer func() {
			fmt.Printf("Client disconnected\n")
			connClosed = true
			conn.Close()
		}()
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				return
			}
			if string(msg) == "ping" {
				fmt.Println("ping")
				time.Sleep(2 * time.Second)
				err = conn.WriteMessage(msgType, []byte("pong"))
				if err != nil {
					fmt.Println(err)
					return
				}
			} else {
				fmt.Println(string(msg))
				return
			}
		}
	})

	for msg := range rtm.IncomingEvents {
		//fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		/*
			case *slack.HelloEvent:
				// Ignore hello
		*/
		case *slack.ConnectedEvent:
			if teamInfo != nil {
				continue
			}
			teamInfo = ev.Info
			//fmt.Println("Infos:", ev.Info)
			//fmt.Println("Connection counter:", ev.ConnectionCount)
			// Replace #general with your Channel ID
			// rtm.SendMessage(rtm.NewOutgoingMessage("<slack connected>", channel.ID))
			listen := fmt.Sprintf(":%s", port)
			go http.ListenAndServe(listen, nil)
			fmt.Printf("Server started on :%s\n", listen)

		case *slack.MessageEvent:
			if ev.Channel != channel.ID || ev.SubType == "message_deleted" { // TODO
				continue
			}
			u, err := api.GetUserInfo(ev.User)
			if err == nil {
				ev.User = u.Name // TODO cache / prelim prime with user list
			}
			m.Write(ev)

		/*case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())
		*/
		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return

		default:

			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
}
