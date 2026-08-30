package utils

import "testing"

func TestBuildAvatarURL(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "", want: ""},
		{value: "https://example.com/avatar.png", want: "https://example.com/avatar.png"},
		{value: "asset_1", want: "/api/avatar/user/1"},
	}
	for _, tt := range tests {
		if got := BuildAvatarURL(tt.value, "/api/avatar/user/1"); got != tt.want {
			t.Fatalf("BuildAvatarURL(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestAvatarAssetID(t *testing.T) {
	if got := AvatarAssetID("asset_1"); got != "asset_1" {
		t.Fatalf("AvatarAssetID() = %q, want asset_1", got)
	}
	if got := AvatarAssetID("https://example.com/avatar.png"); got != "" {
		t.Fatalf("AvatarAssetID() = %q, want empty string", got)
	}
}
