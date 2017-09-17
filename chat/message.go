package chat

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nlopes/slack"
)

type teamMessage struct {
	Type     string            `json:"type"`
	Slack    string            `json:"slack"`
	Users    []chatUser        `json:"users"`
	Channels []chatChannel     `json:"channels"`
	Emoji    map[string]string `json:"emoji"`
}
type chatChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type chatUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar_url"`
	// we have to be careful not to leak private fields
}
type chatMessage struct {
	Type    string       `json:"type"`
	Ts      string       `json:"ts"`
	Text    string       `json:"text"`
	User    *chatUser    `json:"user"`
	Channel *chatChannel `json:"channel"`
}
type authMessage struct {
	Type    string  `json:"type"`
	Token   string  `json:"token"`
	Warning *string `json:"warning"`
}

type ClientMessageAuth struct {
	Token string
}
type ClientMessageHistory struct {
	ChannelID string
}
type ClientMessageSend struct {
	ChannelID string
	Text      string
}

func encode(m interface{}) []byte {
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(m)
	if err != nil {
		log.Printf("encode error: %s\n", err)
		return nil
	}
	return buff.Bytes()
}
func DecodeClientMessage(c *ClientMessage) (typedMessage interface{}, err error) {
	buff := map[string]string{}
	err = json.NewDecoder(bytes.NewReader(c.Raw)).Decode(&buff)
	if err != nil {
		return
	}

	switch t := buff["type"]; t {
	case "message":
		channelID := buff["channel_id"]
		if channelID == "" {
			err = fmt.Errorf("invalid client message received: missing channel_id")
			return
		}
		cms := &ClientMessageSend{ChannelID: channelID, Text: buff["text"]}
		if cms.Text == "" {
			err = fmt.Errorf("invalid client message received: missing text")
		} else {
			typedMessage = cms
		}
	case "history":
		channelID := buff["channel_id"]
		if channelID == "" {
			err = fmt.Errorf("invalid client message received: missing channel_id")
			return
		}
		typedMessage = &ClientMessageHistory{ChannelID: channelID}
	case "auth":
		typedMessage = &ClientMessageAuth{Token: buff["token"]}
	default:
		err = fmt.Errorf("unknown message type %s received (%v)", t, buff)
	}
	return
}
func EncodeWelcomePayload(teamInfo *slack.Info, customEmoji map[string]string) []byte {
	channels := []chatChannel{}
	if teamInfo.Channels != nil {
		for _, c := range teamInfo.Channels {
			channels = append(channels, chatChannel{ID: c.ID, Name: c.Name})
		}
	}
	users := []chatUser{}
	if teamInfo.Users != nil {
		for _, u := range teamInfo.Users {
			users = append(users, chatUser{Username: u.Name, Avatar: u.Profile.ImageOriginal, ID: u.ID})
		}
	}
	tm := teamMessage{
		Slack:    teamInfo.Team.Name,
		Type:     "team-info",
		Users:    users,
		Channels: channels,
		Emoji:    customEmoji,
	}
	return encode(tm)

}
func EncodeMessageEvent(c *slack.Client, m *slack.MessageEvent) []byte {
	cm := chatMessage{Type: "message", Ts: m.Timestamp, Text: m.Text, Channel: &chatChannel{ID: m.Channel}}
	// TODO ask for users/sigils over the wire
	if m.SubType == "bot_message" {
		gravatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=retro", md5.Sum([]byte(m.Username)))
		cm.User = &chatUser{Username: m.Username, Avatar: gravatarURL}
	} else {
		u, err := c.GetUserInfo(m.User)
		if err == nil {
			cm.User = &chatUser{Username: u.Name, Avatar: u.Profile.ImageOriginal, ID: u.ID} // TODO cache / prelim prime with user list
		}
	}
	return encode(cm)
}
func EncodeAuthMessage(token string, warning *string) []byte {
	return encode(authMessage{Type: "auth", Token: token, Warning: warning})
}
