package chat

import (
	"crypto/md5"
	"fmt"
	"log"

	"github.com/nlopes/slack"
)

// hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from slack to the clients.
	broadcast chan []byte

	// Inbound messages from clients to slack
	inbox chan *ClientMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Slack RTM Client
	slack *slack.Client

	// TODO support multiple here
	slackChannel *slack.Channel

	slackInfo   *slack.Info
	customEmoji map[string]string
}

func NewHub(cfg *Config) (*Hub, error) {
	h := &Hub{
		slack:      slack.New(cfg.Slack.Token),
		inbox:      make(chan *ClientMessage),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//logger.SetLevel()
	//slack.SetLogger(logger)
	//api.SetDebug(true)

	err := h.loadSlackInfo(cfg.Slack.AllowedChannels)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Hub) loadSlackInfo(allowedChannels string) error {
	channels, err := h.slack.GetChannels(true)
	if err != nil {
		return err
	}

	h.customEmoji, err = h.slack.GetEmoji()
	if err != nil {
		return fmt.Errorf("Couldn't load emojis: %s", err)
	}

	for _, c := range channels {
		if c.Name == allowedChannels {
			h.slackChannel = &c
			break
		}
	}
	if h.slackChannel == nil {
		return fmt.Errorf("Couldn't find channel %s", allowedChannels)
	}
	return nil
}

func (h *Hub) runSlack() {
	slackSock := h.slack.NewRTM()
	go slackSock.ManageConnection()
	for slackEvent := range slackSock.IncomingEvents {
		//log.Println("slack event.")
		h.handleSlackEvent(slackEvent)
	}
}

func (h *Hub) Run() {
	go h.runSlack()
	// TODO wait until we get the welcome payload or else risk panic

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("client registered.")
			client.send <- h.welcomePayload(client)
			for _, m := range h.previousMessages() {
				client.send <- m
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				log.Println("client unregistered.")
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.inbox:
			h.handleInbox(message)
		case message := <-h.broadcast:
			// mux message to all clients
			log.Printf("flushing broadcast.")
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) handleInbox(c *ClientMessage) {
	// log.Printf("got message from client %s: %s.", c.Client.Username, string(c.Raw))
	m, err := DecodeClientMessage(c)
	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}
	if m == "" {
		return
	}
	log.Printf("sending as client %s: %s.", c.Client.Username, m)
	gravatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=retro", md5.Sum([]byte(c.Client.Username)))
	_, _, err = h.slack.PostMessage(h.slackChannel.ID, m, slack.PostMessageParameters{
		Username: c.Client.Username,
		IconURL:  gravatarURL,
	})
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}
func (h *Hub) previousMessages() [][]byte {
	previous := [][]byte{}
	history, err := h.slack.GetChannelHistory(h.slackChannel.ID, slack.NewHistoryParameters())
	if err != nil {
		log.Printf("error: %s\n", err)
		return previous
	}
	if history.Messages == nil {
		return previous
	}

	// push oldest -> newest
	for i := len(history.Messages) - 1; i >= 0; i-- {
		m := history.Messages[i]
		// TODO
		if m.SubType != "" && m.SubType != "bot_message" {
			//log.Printf("dropping %#v", ev)
			continue
		}
		previous = append(previous, EncodeMessageEvent(h.slack, (*slack.MessageEvent)(&m)))
	}
	return previous
}
func (h *Hub) welcomePayload(c *Client) []byte {
	return EncodeWelcomePayload(h.slackInfo, h.slackChannel, h.customEmoji, c.Username)
}
func (h *Hub) handleSlackEvent(msg slack.RTMEvent) {
	switch ev := msg.Data.(type) {
	/*
	   case *slack.HelloEvent:
	     // Ignore hello
	*/
	case *slack.ConnectedEvent:
		if h.slackInfo != nil {
			break
		}
		h.slackInfo = ev.Info
		//fmt.Println("Infos:", ev.Info)
		//fmt.Println("Connection counter:", ev.ConnectionCount)
		// Replace #general with your Channel ID
		// rtm.SendMessage(rtm.NewOutgoingMessage("<slack connected>", channel.ID))

	case *slack.MessageEvent:
		if ev.Channel != h.slackChannel.ID { // TODO
			//log.Printf("dropping %#v", ev)
			break
		} else if ev.SubType != "" && ev.SubType != "bot_message" {
			log.Printf("dropping %#v", ev)
			break
		}
		h.broadcast <- EncodeMessageEvent(h.slack, (*slack.MessageEvent)(ev))

	/*case *slack.PresenceChangeEvent:
	    fmt.Printf("Presence Change: %v\n", ev)

	  case *slack.LatencyReport:
	    fmt.Printf("Current latency: %v\n", ev.Value)

	  case *slack.RTMError:
	    fmt.Printf("Error: %s\n", ev.Error())
	*/
	case *slack.InvalidAuthEvent:
		log.Println("rtm error: invalid credentials")
		break

	default:

		// Ignore other events..
		// fmt.Printf("Unexpected: %v\n", msg.Data)
	}
}
