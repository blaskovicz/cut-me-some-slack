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
type chatMessage struct {
	Type    string              `json:"type"`
	Message *slack.MessageEvent `json:"message"`
}

func EncodeWelcomePayload(teamInfo *slack.Info, channel *slack.Channel) []byte {
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(teamMessage{Slack: teamInfo.Team.Name, Type: "team-info", Channel: "#" + channel.Name})
	if err != nil {
		log.Printf("error: %s\n", err)
		return nil
	}
	return buff.Bytes()
}
func EncodeMessageEvent(m *slack.MessageEvent) []byte {
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(chatMessage{Type: "message", Message: m})
	if err != nil {
		log.Printf("error: %s\n", err)
		return nil
	}
	return buff.Bytes()
}
