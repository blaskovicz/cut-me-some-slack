package chat

import (
	"github.com/jinzhu/configor"
	"github.com/joho/godotenv"
)

// anything not json-hidden here will be exposed to the world
type Config struct {
	Slack struct {
		Token string `json:"-" required:"true" env:"SLACK_TOKEN"` //TODO validate scopes
		// TODO DisallowedChannels string `default:"api-testing" env:"SLACK_CHANNEL"`
	} `json:"-"`
	Server struct {
		Domain      string `json:"domain" default:"localhost" env:"HEROKU_APP_DOMAIN"`
		Port        uint   `json:"port" default:"3000" env:"PORT"`
		JWTSecret   string `json:"-" required:"true" env:"JWT_SECRET"` // for hs256 hmac signing
		LogMessages bool   `json:"-" env:"LOG_MESSAGES"`
		Identity    struct {
			ClientID     string `json:"client_id" env:"IDENTITY_CLIENT_ID"`
			ClientSecret string `json:"-" env:"IDENTITY_CLIENT_SECRET"`
			Domain       string `json:"domain" env:"IDENTITY_DOMAIN"`
			CallbackURL  string `json:"callback_url" env:"IDENTITY_CALLBACK_URL"`
		} `json:"identity"`
	} `json:"server"`
}

func LoadConfig() (*Config, error) {
	var cfg Config

	// ignore the error
	godotenv.Load()

	err := configor.Load(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
