package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://slack.com/api"

type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

func NewClient(botToken string) *Client {
	return &Client{
		botToken:   strings.TrimSpace(botToken),
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SetBaseURL(url string) {
	if strings.TrimSpace(url) != "" {
		c.baseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

func (c *Client) PostMessage(ctx context.Context, channel string, text string, threadTS string) (*SendMessageResponse, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return nil, fmt.Errorf("slack channel is required")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("message text is required")
	}

	payload := SendMessageRequest{
		Channel:  channel,
		Text:     text,
		ThreadTS: threadTS,
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, "/chat.postMessage", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("slack api error: %s", resp.Error)
	}
	return &resp, nil
}

func (c *Client) doRequest(ctx context.Context, path string, payload any, result any) error {
	if c.botToken == "" {
		return fmt.Errorf("slack bot token is required")
	}

	endpoint := fmt.Sprintf("%s%s", c.baseURL, path)

	var bodyReader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal slack request failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create slack request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.botToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack http request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read slack response failed: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("slack api error (%d): %s", res.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("unmarshal slack response failed: %w (body: %s)", err, string(bodyBytes))
		}
	}
	return nil
}
