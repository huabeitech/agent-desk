package discord

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

const defaultBaseURL = "https://discord.com/api/v10"

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

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	var user User
	if err := c.doRequest(ctx, http.MethodGet, "/users/@me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) CreateDMChannel(ctx context.Context, recipientID string) (*Channel, error) {
	if strings.TrimSpace(recipientID) == "" {
		return nil, fmt.Errorf("recipient_id is required")
	}
	req := CreateDMRequest{RecipientID: strings.TrimSpace(recipientID)}
	var channel Channel
	if err := c.doRequest(ctx, http.MethodPost, "/users/@me/channels", req, &channel); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (c *Client) SendMessage(ctx context.Context, channelID string, content string) (*Message, error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, fmt.Errorf("channel_id is required")
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	req := SendMessageRequest{Content: content}
	var msg Message
	endpoint := fmt.Sprintf("/channels/%s/messages", channelID)
	if err := c.doRequest(ctx, http.MethodPost, endpoint, req, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Client) SendEmbedMessage(ctx context.Context, channelID string, content string, embeds []Embed) (*Message, error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, fmt.Errorf("channel_id is required")
	}

	req := SendMessageRequest{
		Content: content,
		Embeds:  embeds,
	}
	var msg Message
	endpoint := fmt.Sprintf("/channels/%s/messages", channelID)
	if err := c.doRequest(ctx, http.MethodPost, endpoint, req, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, payload any, result any) error {
	if c.botToken == "" {
		return fmt.Errorf("discord bot token is required")
	}

	endpoint := fmt.Sprintf("%s%s", c.baseURL, path)

	var bodyReader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal discord request failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create discord request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+c.botToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord http request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read discord response failed: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("discord api error (%d): %s", res.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("unmarshal discord response failed: %w (body: %s)", err, string(bodyBytes))
		}
	}
	return nil
}
