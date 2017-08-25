package chat

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/nlopes/slack"
)

type teamMessage struct {
	Type    string `json:"type"`
	Slack   string `json:"slack"`
	Channel string `json:"channel"`
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

func EncodeWelcomePayload(teamInfo *slack.Info, channel *slack.Channel) []byte {
	tm := teamMessage{Slack: teamInfo.Team.Name, Type: "team-info", Channel: "#" + channel.Name}
	return encode(tm)

}
func EncodeMessageEvent(c *slack.Client, m *slack.MessageEvent) []byte {
	cm := chatMessage{Type: "message", Ts: m.Timestamp, Text: m.Text}
	u, err := c.GetUserInfo(m.User)
	if err == nil {
		cm.User = &chatUserMessage{Username: u.Name, Avatar: u.Profile.ImageOriginal} // TODO cache / prelim prime with user list
	}
	return encode(cm)
}
