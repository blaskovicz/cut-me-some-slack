package chat

import (
	"github.com/jinzhu/configor"
	"github.com/joho/godotenv"
)

type Config struct {
	Slack struct {
		Token string `required:"true" env:"SLACK_TOKEN"` //TODO validate scopes
		// TODO DisallowedChannels string `default:"api-testing" env:"SLACK_CHANNEL"`
	}
	Server struct {
		Domain      string `default:"localhost" env:"HEROKU_APP_DOMAIN"`
		Port        uint   `default:"3000" env:"PORT"`
		JWTSecret   string `required:"true" env:"JWT_SECRET"` // for hs256 hmac signing
		LogMessages bool   `env:"LOG_MESSAGES"`
	}
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
