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
	clients     map[*Client]bool
	clientCount int

	// Inbound messages from slack to the clients.
	broadcast chan []byte

	// Inbound messages from clients to slack
	inbox chan *ClientMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// once we're up and running
	slackConnected chan interface{}

	// log slack inbound and outbound messages
	logMessages bool

	// for jwt hs256 hmac signing
	jwtSecret []byte

	// Slack RTM Client
	slack *slack.Client

	// information pushed during welcome
	slackInfo   *slack.Info
	teamInfo    *slack.TeamInfo
	customEmoji map[string]string
}

func NewHub(cfg *Config) (*Hub, error) {
	h := &Hub{
		logMessages:    cfg.Server.LogMessages,
		jwtSecret:      []byte(cfg.Server.JWTSecret),
		slack:          slack.New(cfg.Slack.Token),
		inbox:          make(chan *ClientMessage),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]bool),
		slackConnected: make(chan interface{}),
	}
	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//logger.SetLevel()
	//slack.SetLogger(logger)
	//api.SetDebug(true)

	return h, nil
}

func (h *Hub) loadSlackInfo() {
	var err error
	h.customEmoji, err = h.slack.GetEmoji()
	if err != nil {
		log.Printf("error: couldn't load emojis: %s\n", err)
	}
	h.teamInfo, err = h.slack.GetTeamInfo()
	if err != nil {
		log.Printf("error: couldn't load extra team info: %s\n", err)
	}
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
	h.loadSlackInfo()
	go h.runSlack()
	<-h.slackConnected

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.clientCount++
			log.Printf("client registered (count=%d)\n", h.clientCount)
			go func() {
				client.send <- h.welcomePayload(client)
			}()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.clientCount--
				log.Printf("client unregistered (count=%d)\n", h.clientCount)
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.inbox:
			go func() {
				h.handleInbox(message)
			}()
		case message := <-h.broadcast:
			// mux message to all clients
			log.Printf("flushing message broadcast to all clients (count=%d)\n", h.clientCount)
			go func() {
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						h.clientCount--
						close(client.send)
						delete(h.clients, client)
					}
				}
			}()
		}
	}
}

// check if we're using a valid slack channel
// from either name or ID and then use the canonical ID
func (h *Hub) resolveSlackChannel(idOrName string) (id string) {
	for _, c := range h.slackInfo.Channels {
		if c.Name == idOrName || c.ID == idOrName {
			return c.ID
		}
	}
	return ""
}

func (h *Hub) handleInbox(c *ClientMessage) {
	// TODO send error message events to client
	raw, err := DecodeClientMessage(c)
	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}
	switch m := raw.(type) {
	case *ClientMessageHistory:
		channelID := h.resolveSlackChannel(m.ChannelID)
		if channelID == "" {
			log.Printf("error: no channel found matching %s\n", m.ChannelID)
			return
		}
		var username string
		if c.Client.User != nil {
			username = c.Client.User.Username
		} else {
			username = "<anonymous>"
		}
		log.Printf("sending previous messages for channel %s to client %s\n", channelID, username)
		for _, prevMessage := range h.previousMessages(channelID, m.Limit) {
			c.Client.send <- prevMessage
		}
	case *ClientMessageSend:
		if c.Client.User == nil {
			log.Printf("warn: skipping message send because user is un-authed\n")
			return
		}
		channelID := h.resolveSlackChannel(m.ChannelID)
		if channelID == "" {
			log.Printf("error: no channel found matching %s (skipping sending as client %s)\n", m.ChannelID, c.Client.User.Username)
			return
		}
		log.Printf("sending as client %s to %s\n", c.Client.User.Username, channelID)
		gravatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=retro", md5.Sum([]byte(c.Client.User.Username)))
		_, _, err = h.slack.PostMessage(channelID, m.Text, slack.PostMessageParameters{
			Username: c.Client.User.Username,
			IconURL:  gravatarURL,
		})
		if err != nil {
			log.Printf("error: failed to send - %s\n", err)
		}
	case *ClientMessageAuth:
		if m.Token == "" {
			// generate new identity
			// TODO make sure identity isn't already in use for anon sockets
			user, signedToken, err := generateSignedJWT(h.jwtSecret)
			if err != nil {
				log.Printf("error: failed to generate new token on auth request - %s\n", err)
				return
			}
			log.Printf("sending new identity %s to client\n", user.Username)
			c.Client.User = user
			c.Client.send <- EncodeAuthMessage(signedToken, nil)
		} else {
			// check provided identity, optionally generating a new jwt
			user, _, err := verifySignedJWT(h.jwtSecret, m.Token)
			if err != nil {
				// TODO probably just send back an error instead of generating an interm identity
				log.Printf("error: failed to verify jwt on auth request, generating new token (%s) - %s\n", m.Token, err)
				user, signedToken, err := generateSignedJWT(h.jwtSecret)
				if err != nil {
					log.Printf("error: failed to generate new token on auth request - %s\n", err)
					return
				}
				log.Printf("sending re-generated identity %s to client\n", user.Username)
				warn := "invalid identity provided. generated new identity."
				c.Client.User = user
				c.Client.send <- EncodeAuthMessage(signedToken, &warn)
			} else {
				// TODO this could be where we extend the exp claim
				log.Printf("verified token for identity %s\n", user.Username)
				c.Client.User = user
				c.Client.send <- EncodeAuthMessage(m.Token, nil)
			}
		}
	}
}

func (h *Hub) previousMessages(channelID string, limit int) [][]byte {
	previous := [][]byte{}
	messageQuery := slack.NewHistoryParameters()
	messageQuery.Count = limit
	// TODO allow requesting older history upon scroll
	history, err := h.slack.GetChannelHistory(channelID, messageQuery)
	if err != nil {
		log.Printf("error: %s\n", err)
		return previous
	} else if history.Messages == nil {
		return previous
	}

	// push oldest -> newest
	for i := len(history.Messages) - 1; i >= 0; i-- {
		m := history.Messages[i]
		if !ClientHandlesMessage(&m) {
			//log.Printf("history %s: dropping %#v", channelID, ev)
			continue
		}
		m.Channel = channelID // channel is unset in slack response, but our client expects it
		previous = append(previous, EncodeMessageEvent(h.slack, (*slack.MessageEvent)(&m)))
	}
	return previous
}
func (h *Hub) welcomePayload(c *Client) []byte {
	return EncodeWelcomePayload(h.slackInfo, h.customEmoji, h.teamInfo)
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

		// first time, now we're ready for clients
		if ev.ConnectionCount == 1 {
			go func() {
				log.Println("signaling slack connected event")
				h.slackConnected <- struct{}{}
			}()
		}

	case *slack.MessageEvent:
		if !ClientHandlesMessage((*slack.Message)(ev)) {
			if h.logMessages {
				log.Printf("message %s: dropping %#v", ev.Channel, ev)
			}
			break
		}
		if h.logMessages {
			log.Printf("message %s: %#v\n", ev.Channel, ev)
		}
		h.broadcast <- EncodeMessageEvent(h.slack, ev)

	// TODO periodically update users, emoji, channels, etc and push to client
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
