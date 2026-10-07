package models

import "testing"

func TestUserAvatarMethods(t *testing.T) {
	user := User{ID: 3, Avatar: "asset_1"}
	if got := user.UserAvatarAssetID(); got != "asset_1" {
		t.Fatalf("UserAvatarAssetID() = %q", got)
	}
	if got := user.UserAvatarURL(); got != "/api/avatar/user/3" {
		t.Fatalf("UserAvatarURL() = %q", got)
	}

	user.Avatar = "https://example.com/avatar.png"
	if got := user.UserAvatarAssetID(); got != "" {
		t.Fatalf("UserAvatarAssetID() = %q, want empty", got)
	}
	if got := user.UserAvatarURL(); got != "https://example.com/avatar.png" {
		t.Fatalf("UserAvatarURL() = %q", got)
	}
}

func TestAgentAvatarMethods(t *testing.T) {
	profile := AgentProfile{ID: 5, Avatar: "asset_2"}
	if got := profile.AgentAvatarAssetID(); got != "asset_2" {
		t.Fatalf("AgentAvatarAssetID() = %q", got)
	}
	if got := profile.AgentAvatar(); got != "/api/avatar/agent/5" {
		t.Fatalf("AgentAvatar() = %q", got)
	}
}
