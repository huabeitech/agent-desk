package discord

// User represents a Discord user.
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator,omitempty"`
	GlobalName    string `json:"global_name,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	Bot           bool   `json:"bot,omitempty"`
}

// Channel represents a Discord channel (Guild Text, DM, Thread, etc.).
type Channel struct {
	ID      string `json:"id"`
	Type    int    `json:"type"`
	GuildID string `json:"guild_id,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Attachment represents a file or image uploaded to Discord.
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// EmbedMedia represents an image/video/thumbnail inside an Embed.
type EmbedMedia struct {
	URL string `json:"url"`
}

// Embed represents a Discord rich embed object.
type Embed struct {
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	URL         string      `json:"url,omitempty"`
	Color       int         `json:"color,omitempty"`
	Image       *EmbedMedia `json:"image,omitempty"`
}

// Message represents a Discord message.
type Message struct {
	ID          string       `json:"id"`
	ChannelID   string       `json:"channel_id"`
	GuildID     string       `json:"guild_id,omitempty"`
	Author      User         `json:"author"`
	Content     string       `json:"content"`
	Timestamp   string       `json:"timestamp"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Embeds      []Embed      `json:"embeds,omitempty"`
}

// SendMessageRequest represents payload for Discord create message API.
type SendMessageRequest struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

// CreateDMRequest represents payload for Discord create DM channel API.
type CreateDMRequest struct {
	RecipientID string `json:"recipient_id"`
}

// WebhookPayload represents an incoming message/event from Discord Gateway or Webhook.
type WebhookPayload struct {
	ID          string       `json:"id,omitempty"`
	Type        int          `json:"type,omitempty"`
	GuildID     string       `json:"guild_id,omitempty"`
	ChannelID   string       `json:"channel_id,omitempty"`
	Author      *User        `json:"author,omitempty"`
	Content     string       `json:"content,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Embeds      []Embed      `json:"embeds,omitempty"`
	Message     *Message     `json:"message,omitempty"`
}
