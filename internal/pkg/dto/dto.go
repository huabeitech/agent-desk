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
	// AgentID 指定该渠道使用的企业微信自建应用。
	// 系统据此找到对应的 corpSecret，换取并缓存该应用独立的 accessToken。
	AgentID string `json:"agentId"`
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

// WechatMiniProgramChannelConfig 微信小程序客服消息渠道配置。
//
//	注意与 WechatMPChannelConfig（微信公众号）区分。
type WechatMiniProgramChannelConfig struct {
	Title              string `json:"title"`
	Subtitle           string `json:"subtitle"`
	ThemeColor         string `json:"themeColor"`
	UserTokenSecret    string `json:"userTokenSecret,omitempty"`
	AppID              string `json:"appId"`
	Token              string `json:"token"`
	EncodingAESKey     string `json:"encodingAESKey"`
	TokenServiceURL    string `json:"tokenServiceUrl"`
	TokenServiceSecret string `json:"tokenServiceSecret,omitempty"`
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
