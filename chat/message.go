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
	Type     string `json:"type"`
	Slack    string `json:"slack"`
	Channel  string `json:"channel"`
	Username string `json:"username"`
}
type chatUserMessage struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar_url"`
	// we have to be careful not to leak private fields
}
type chatMessage struct {
	Type string           `json:"type"`
	Ts   string           `json:"ts"`
	Text string           `json:"text"`
	User *chatUserMessage `json:"user"`
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
func DecodeClientMessage(c *ClientMessage) (string, error) {
	buff := map[string]string{}
	err := json.NewDecoder(bytes.NewReader(c.Raw)).Decode(&buff)
	if err != nil {
		return "", err
	}
	if t := buff["type"]; t == "message" {
		if m := buff["text"]; m != "" {
			return m, nil
		}
	}
	return "", nil
}
func EncodeWelcomePayload(teamInfo *slack.Info, channel *slack.Channel, username string) []byte {
	tm := teamMessage{Slack: teamInfo.Team.Name, Type: "team-info", Channel: "#" + channel.Name, Username: "@" + username}
	return encode(tm)

}
func EncodeMessageEvent(c *slack.Client, m *slack.MessageEvent) []byte {
	cm := chatMessage{Type: "message", Ts: m.Timestamp, Text: m.Text}
	// TODO ask for users/sigils over the wire
	if m.SubType == "bot_message" {
		gravatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=retro", md5.Sum([]byte(m.Username)))
		cm.User = &chatUserMessage{Username: m.Username, Avatar: gravatarURL}
	} else {
		u, err := c.GetUserInfo(m.User)
		if err == nil {
			cm.User = &chatUserMessage{Username: u.Name, Avatar: u.Profile.ImageOriginal} // TODO cache / prelim prime with user list
		}
	}
	return encode(cm)
}
