package discord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot test_token" {
			t.Errorf("expected Bot test_token, got %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/channels/789/messages" {
			t.Errorf("expected path /channels/789/messages, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123456","channel_id":"789","content":"hello"}`))
	}))
	defer server.Close()

	client := NewClient("test_token")
	client.SetBaseURL(server.URL)

	resp, err := client.SendMessage(context.Background(), "789", "hello")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if resp.ID != "123456" {
		t.Errorf("expected ID 123456, got %s", resp.ID)
	}
}

func TestDiscordSendEmbedMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot test_token" {
			t.Errorf("expected Bot test_token, got %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/channels/789/messages" {
			t.Errorf("expected path /channels/789/messages, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"embed_123","channel_id":"789","content":"Check image"}`))
	}))
	defer server.Close()

	client := NewClient("test_token")
	client.SetBaseURL(server.URL)

	embed := Embed{
		Title: "Screenshot",
		Image: &EmbedMedia{URL: "https://example.com/img.png"},
	}
	resp, err := client.SendEmbedMessage(context.Background(), "789", "Check image", []Embed{embed})
	if err != nil {
		t.Fatalf("SendEmbedMessage failed: %v", err)
	}
	if resp.ID != "embed_123" {
		t.Errorf("expected ID embed_123, got %s", resp.ID)
	}
}

func TestDiscordCreateDMChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot test_token" {
			t.Errorf("expected Bot test_token, got %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/users/@me/channels" {
			t.Errorf("expected path /users/@me/channels, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"dm_chan_123","type":1}`))
	}))
	defer server.Close()

	client := NewClient("test_token")
	client.SetBaseURL(server.URL)

	resp, err := client.CreateDMChannel(context.Background(), "user_999")
	if err != nil {
		t.Fatalf("CreateDMChannel failed: %v", err)
	}
	if resp.ID != "dm_chan_123" {
		t.Errorf("expected ID dm_chan_123, got %s", resp.ID)
	}
}
