package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/pkg/config"
)

func TestWebhookNotifySendGenericPayload(t *testing.T) {
	var got map[string]any
	var signature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		signature = r.Header.Get("X-Agent-Desk-Signature")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode payload error = %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	setWebhookNotifyTestConfig(t, config.WebhookNotifyConfig{
		Enabled: true,
		URL:     server.URL,
		Format:  "generic",
		Secret:  "merchant-secret",
	})
	svc := newWebhookNotifyService()
	svc.now = func() time.Time { return time.Unix(123, 0).UTC() }

	if err := svc.SendText("sales_lead_created", "高意向销售线索提醒", "客户: 李女士", map[string]any{"leadId": float64(7)}); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
	if got["eventType"] != "sales_lead_created" || got["title"] != "高意向销售线索提醒" {
		t.Fatalf("unexpected payload: %#v", got)
	}
	if !strings.Contains(got["text"].(string), "客户: 李女士") {
		t.Fatalf("expected text content, got %#v", got["text"])
	}
	if signature == "" || !strings.HasPrefix(signature, "sha256=") {
		t.Fatalf("expected signature header, got %q", signature)
	}
}

func TestWebhookNotifySendRobotTextPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode payload error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setWebhookNotifyTestConfig(t, config.WebhookNotifyConfig{
		Enabled: true,
		URL:     server.URL,
		Format:  "wecom_robot",
	})
	if err := newWebhookNotifyService().SendText("conversation_assigned", "会话分配提醒", "会话: #3", nil); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
	if got["msgtype"] != "text" {
		t.Fatalf("unexpected robot payload: %#v", got)
	}
	text, ok := got["text"].(map[string]any)
	if !ok || !strings.Contains(text["content"].(string), "会话: #3") {
		t.Fatalf("unexpected text field: %#v", got["text"])
	}
}

func TestWebhookNotifyReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad webhook", http.StatusBadGateway)
	}))
	defer server.Close()

	setWebhookNotifyTestConfig(t, config.WebhookNotifyConfig{
		Enabled: true,
		URL:     server.URL,
	})
	err := newWebhookNotifyService().SendText("sales_lead_created", "提醒", "内容", nil)
	if err == nil || !strings.Contains(err.Error(), "status=502") {
		t.Fatalf("expected http error, got %v", err)
	}
}

func setWebhookNotifyTestConfig(t *testing.T, webhook config.WebhookNotifyConfig) {
	t.Helper()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: webhook,
		},
	})
	t.Cleanup(func() {
		config.SetCurrent(&config.Config{})
	})
}
