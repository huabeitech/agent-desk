package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-desk/internal/pkg/config"
)

var WebhookNotifyService = newWebhookNotifyService()

type webhookNotifyService struct {
	client *http.Client
	now    func() time.Time
}

func newWebhookNotifyService() *webhookNotifyService {
	return &webhookNotifyService{
		client: http.DefaultClient,
		now:    time.Now,
	}
}

func (s *webhookNotifyService) Enabled() bool {
	cfg := config.Current().Notify.Webhook
	return cfg.Enabled && strings.TrimSpace(cfg.URL) != ""
}

func (s *webhookNotifyService) SendText(eventType, title, body string, metadata map[string]any) error {
	if !s.Enabled() {
		return nil
	}
	cfg := config.Current().Notify.Webhook
	content := s.buildTextContent(title, body)
	if content == "" {
		return nil
	}
	payload, err := s.buildPayload(cfg, eventType, title, body, content, metadata)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	timeout := s.normalizeTimeout(cfg.TimeoutMS)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(cfg.URL), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "agent-desk-webhook-notify/1.0")
	for key, value := range cfg.Headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if secret := strings.TrimSpace(cfg.Secret); secret != "" {
		timestamp := fmt.Sprint(s.now().Unix())
		req.Header.Set("X-Agent-Desk-Timestamp", timestamp)
		req.Header.Set("X-Agent-Desk-Signature", s.signPayload(secret, timestamp, raw))
	}
	client := s.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	limited, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("webhook notify failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(limited)))
}

func (s *webhookNotifyService) buildPayload(cfg config.WebhookNotifyConfig, eventType, title, body, content string, metadata map[string]any) (any, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", "generic", "json":
		if metadata == nil {
			metadata = map[string]any{}
		}
		return map[string]any{
			"eventType": strings.TrimSpace(eventType),
			"title":     strings.TrimSpace(title),
			"content":   strings.TrimSpace(body),
			"text":      content,
			"metadata":  metadata,
			"timestamp": s.now().Format(time.RFC3339),
		}, nil
	case "wecom", "wecom_robot", "wechat_work", "dingtalk", "text":
		return map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		}, nil
	case "feishu", "lark":
		return map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported webhook notify format: %s", cfg.Format)
	}
}

func (s *webhookNotifyService) buildTextContent(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	switch {
	case title == "" && body == "":
		return ""
	case title == "":
		return s.truncateRunes(body, 4000)
	case body == "":
		return s.truncateRunes(title, 4000)
	default:
		return s.truncateRunes(title+"\n\n"+body, 4000)
	}
}

func (s *webhookNotifyService) normalizeTimeout(value int) time.Duration {
	if value <= 0 {
		return 5 * time.Second
	}
	if value < 1000 {
		return time.Second
	}
	if value > 30000 {
		return 30 * time.Second
	}
	return time.Duration(value) * time.Millisecond
}

func (s *webhookNotifyService) signPayload(secret string, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (s *webhookNotifyService) truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}
