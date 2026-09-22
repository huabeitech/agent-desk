package dto

import "agent-desk/internal/pkg/enums"

type AuthPrincipal struct {
	UserID      int64
	Username    string
	Nickname    string
	Avatar      string
	UserType    enums.UserType
	Status      enums.Status
	Roles       []string
	Permissions []string
}

type WxWorkKFChannelConfig struct {
	OpenKfID string `json:"openKfId"`
}

type WebChannelConfig struct {
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	ThemeColor      string `json:"themeColor"`
	Position        string `json:"position"`
	Width           string `json:"width"`
	UserTokenSecret string `json:"userTokenSecret,omitempty"`
}

type WechatMPChannelConfig struct {
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	ThemeColor      string `json:"themeColor"`
	UserTokenSecret string `json:"userTokenSecret,omitempty"`
}

type TelegramChannelConfig struct {
	BotToken       string `json:"botToken"`
	BotUsername    string `json:"botUsername,omitempty"`
	WebhookSecret  string `json:"webhookSecret,omitempty"`
	WelcomeMessage string `json:"welcomeMessage,omitempty"`
}

type ZaloOAChannelConfig struct {
	AppID          string `json:"appId,omitempty"`
	OAID           string `json:"oaId,omitempty"`
	SecretKey      string `json:"secretKey,omitempty"`
	AccessToken    string `json:"accessToken"`
	RefreshToken   string `json:"refreshToken,omitempty"`
	WebhookSecret  string `json:"webhookSecret,omitempty"`
	WelcomeMessage string `json:"welcomeMessage,omitempty"`
}
type EmailChannelConfig struct {
	EmailAddress      string `json:"emailAddress"`                // e.g. help@dos.crove.io or support@company.com
	ForwardingAddress string `json:"forwardingAddress,omitempty"` // e.g. help@dos.crove.io
	SenderName        string `json:"senderName,omitempty"`        // e.g. Crove Desk Support
	Provider          string `json:"provider,omitempty"`          // default | smtp | brevo | sendgrid | resend | postmark | mailgun
	APIKey            string `json:"apiKey,omitempty"`            // Brevo / ESP API Key
	SMTPHost          string `json:"smtpHost,omitempty"`          // SMTP Server Host
	SMTPPort          int    `json:"smtpPort,omitempty"`          // SMTP Port (587/465)
	SMTPUser          string `json:"smtpUser,omitempty"`          // SMTP Username
	SMTPPassword      string `json:"smtpPassword,omitempty"`      // SMTP Password
	WebhookSecret     string `json:"webhookSecret,omitempty"`     // Inbound Webhook Secret
	WelcomeMessage    string `json:"welcomeMessage,omitempty"`    // Auto-responder / welcome message
}

type DiscordChannelConfig struct {
	GuildID        string `json:"guildId,omitempty"`
	GuildName      string `json:"guildName,omitempty"`
	ChannelScope   string `json:"channelScope,omitempty"` // all | dm_only
	BotToken       string `json:"botToken,omitempty"`
	ApplicationID  string `json:"applicationId,omitempty"`
	PublicKey      string `json:"publicKey,omitempty"`
	WebhookSecret  string `json:"webhookSecret,omitempty"`
	WelcomeMessage string `json:"welcomeMessage,omitempty"`
}
