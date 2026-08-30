package utils

import "testing"

func TestBuildAvatarURL(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "", want: ""},
		{value: "https://example.com/avatar.png", want: "https://example.com/avatar.png"},
		{value: "asset_1", want: "/api/asset/asset_1"},
	}
	for _, tt := range tests {
		if got := BuildAvatarURL(tt.value); got != tt.want {
			t.Fatalf("BuildAvatarURL(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestNormalizeAvatarValue(t *testing.T) {
	if got := NormalizeAvatarValue("/api/asset/asset_1"); got != "asset_1" {
		t.Fatalf("NormalizeAvatarValue() = %q, want asset_1", got)
	}
	if got := NormalizeAvatarValue("https://example.com/avatar.png"); got != "https://example.com/avatar.png" {
		t.Fatalf("NormalizeAvatarValue() = %q, want external URL", got)
	}
}
