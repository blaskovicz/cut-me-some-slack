package chat

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/nlopes/slack"
)

type teamMessage struct {
	Type     string            `json:"type"`
	Slack    string            `json:"slack"`
	Icon     string            `json:"icon"`
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
	Limit     int
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
		var limit int
		if rawLimit := buff["limit"]; rawLimit != "" {
			limit, err = strconv.Atoi(rawLimit)
			if err != nil || limit <= 0 || limit > 1000 {
				err = fmt.Errorf("invalid client message received: limit has incorrect bounds")
				return
			}
		} else {
			limit = 10
		}
		typedMessage = &ClientMessageHistory{ChannelID: channelID, Limit: limit}
	case "auth":
		typedMessage = &ClientMessageAuth{Token: buff["token"]}
	default:
		err = fmt.Errorf("unknown message type %s received (%v)", t, buff)
	}
	return
}
func EncodeWelcomePayload(slackInfo *slack.Info, customEmoji map[string]string, teamInfo *slack.TeamInfo) []byte {
	channels := []chatChannel{}
	if slackInfo.Channels != nil {
		for _, c := range slackInfo.Channels {
			channels = append(channels, chatChannel{ID: c.ID, Name: c.Name})
		}
	}
	users := []chatUser{}
	if slackInfo.Users != nil {
		for _, u := range slackInfo.Users {
			users = append(users, chatUser{Username: u.Name, Avatar: u.Profile.ImageOriginal, ID: u.ID})
		}
	}
	tm := teamMessage{
		Slack:    teamInfo.Name,
		Icon:     teamInfo.Icon["image_88"].(string),
		Type:     "team-info",
		Users:    users,
		Channels: channels,
		Emoji:    customEmoji,
	}
	return encode(tm)

}
func ClientHandlesMessage(ev *slack.Message) bool {
	// TODO: handle other types :)
	return ((ev.SubType == "" || ev.SubType == "bot_message") && ev.Text != "" && ev.Timestamp != "")
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
