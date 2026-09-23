package config

import "testing"

func TestSlackAppChannelValuesWinOverDeployment(t *testing.T) {
	cfg := Config{Slack: SlackConfig{
		ClientID:      "deploy-client-id",
		ClientSecret:  "deploy-client-secret",
		BotToken:      "xoxb-deploy",
		SigningSecret: "deploy-signing-secret",
	}}

	app := cfg.SlackApp("xoxb-channel", "channel-signing-secret")
	if app.ClientID != "deploy-client-id" || app.ClientSecret != "deploy-client-secret" {
		t.Fatalf("the app itself is deployment-wide, got %+v", app)
	}
	if app.BotToken != "xoxb-channel" {
		t.Fatalf("channel bot token must win over the deployment one, got %q", app.BotToken)
	}
	if app.SigningSecret != "channel-signing-secret" {
		t.Fatalf("channel signing secret must win over the deployment one, got %q", app.SigningSecret)
	}

	fallback := cfg.SlackApp("  ", "")
	if fallback.BotToken != "xoxb-deploy" || fallback.SigningSecret != "deploy-signing-secret" {
		t.Fatalf("blank channel values must fall back to the deployment app, got %+v", fallback)
	}
}

// ResolveSlack must stay usable before Load has run (tests, standalone
// commands): the channel values pass through and nothing panics.
func TestResolveSlackWithoutLoadedConfig(t *testing.T) {
	saved := GetCurrent()
	defer SetCurrent(saved)

	SetCurrent(nil)
	app := ResolveSlack("xoxb-channel", "channel-signing-secret")
	if app.BotToken != "xoxb-channel" || app.SigningSecret != "channel-signing-secret" {
		t.Fatalf("expected channel values to pass through, got %+v", app)
	}
	if app.ClientID != "" || app.ClientSecret != "" {
		t.Fatalf("no deployment app exists before Load, got %+v", app)
	}
}
