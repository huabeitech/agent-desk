package slack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostMessageSendsThreadedReply(t *testing.T) {
	var (
		gotPath string
		gotAuth string
		gotBody map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C0123","ts":"1725260000.000100"}`))
	}))
	defer server.Close()

	client := NewClient("xoxb-test-token")
	client.SetBaseURL(server.URL)

	resp, err := client.PostMessage(context.Background(), "C0123", "hello there", "1725260000.000200")
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}
	if resp == nil || resp.TS != "1725260000.000100" {
		t.Fatalf("response = %+v, want the posted message ts", resp)
	}

	if gotPath != "/chat.postMessage" {
		t.Errorf("path = %q, want /chat.postMessage", gotPath)
	}
	if gotAuth != "Bearer xoxb-test-token" {
		t.Errorf("authorization = %q, want the bot token as a bearer", gotAuth)
	}
	if gotBody["channel"] != "C0123" {
		t.Errorf("channel = %v, want C0123", gotBody["channel"])
	}
	if gotBody["text"] != "hello there" {
		t.Errorf("text = %v, want 'hello there'", gotBody["text"])
	}
	// A reply has to stay in the customer's thread, otherwise the workspace
	// timeline fills up with support answers.
	if gotBody["thread_ts"] != "1725260000.000200" {
		t.Errorf("thread_ts = %v, want the parent ts", gotBody["thread_ts"])
	}
}

func TestPostMessageOmitsThreadTSForTopLevel(t *testing.T) {
	var gotRaw string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotRaw = string(raw)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient("xoxb-test-token")
	client.SetBaseURL(server.URL)

	if _, err := client.PostMessage(context.Background(), "C0123", "top level", ""); err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}
	// thread_ts is omitempty on the request struct, so a top-level post must not
	// carry an empty one.
	var body map[string]any
	if err := json.Unmarshal([]byte(gotRaw), &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if _, present := body["thread_ts"]; present {
		t.Errorf("request body carried thread_ts for a top-level message: %s", gotRaw)
	}
}

func TestPostMessageSurfacesSlackAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slack returns HTTP 200 with ok:false for application-level failures, so
		// the client has to read the envelope rather than the status code.
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer server.Close()

	client := NewClient("xoxb-test-token")
	client.SetBaseURL(server.URL)

	_, err := client.PostMessage(context.Background(), "C_MISSING", "hello", "")
	if err == nil {
		t.Fatalf("expected an error when Slack answers ok:false")
	}
	if got := err.Error(); got != "slack api error: channel_not_found" {
		t.Errorf("error = %q, want the slack error code to be carried through", got)
	}
}

func TestPostMessageValidatesInputs(t *testing.T) {
	client := NewClient("xoxb-test-token")

	if _, err := client.PostMessage(context.Background(), "", "hello", ""); err == nil {
		t.Errorf("expected an error for an empty channel")
	}
	if _, err := client.PostMessage(context.Background(), "C0123", "   ", ""); err == nil {
		t.Errorf("expected an error for blank text")
	}

	noToken := NewClient("")
	if _, err := noToken.PostMessage(context.Background(), "C0123", "hello", ""); err == nil {
		t.Errorf("expected an error when no bot token is configured")
	}
}
