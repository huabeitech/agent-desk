package config

import "strings"

var current *Config

func SetCurrent(cfg *Config) {
	current = cfg
}

func Current() Config {
	if current == nil {
		panic("config not initialized")
	}
	return *current
}

func GetCurrent() *Config {
	return current
}

// ResolveSlack resolves the Slack app credentials for one channel without
// requiring the caller to nil-check the loaded configuration.
func ResolveSlack(channelBotToken, channelSigningSecret string) SlackConfig {
	if current == nil {
		return SlackConfig{
			BotToken:      strings.TrimSpace(channelBotToken),
			SigningSecret: strings.TrimSpace(channelSigningSecret),
		}
	}
	return current.SlackApp(channelBotToken, channelSigningSecret)
}
