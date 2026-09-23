package config

import (
	"agent-desk/internal/pkg/enums"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	Language        string                `yaml:"language"`
	Server          ServerConfig          `yaml:"server"`
	DB              DBConfig              `yaml:"db"`
	Logger          LoggerConfig          `yaml:"logger"`
	Auth            AuthConfig            `yaml:"auth"`
	Storage         StorageConfig         `yaml:"storage"`
	VectorDB        VectorDBConfig        `yaml:"vectorDB"`
	MCP             MCPConfig             `yaml:"mcp"`
	WxWork          WxWorkConfig          `yaml:"wxWork"`
	OIDC            OIDCConfig            `yaml:"oidc"`
	CustomerSession CustomerSessionConfig `yaml:"customerSession"`
	Webhook         WebhookConfig         `yaml:"webhook"`
	Discord         DiscordConfig         `yaml:"discord"`
	Slack           SlackConfig           `yaml:"slack"`
}

func (c Config) LanguageOrDefault() string {
	switch strings.ToLower(strings.TrimSpace(c.Language)) {
	case "zh", "zh-cn", "zh_cn", "zh-hans":
		return "zh-CN"
	case "en", "en-us", "en_us":
		return "en-US"
	default:
		return "zh-CN"
	}
}

type WxWorkNotifyConfig struct {
	Enabled                bool    `yaml:"enabled"`
	ToUsers                []int64 `yaml:"toUsers"`
	Safe                   bool    `yaml:"safe"`
	EnableDuplicateCheck   bool    `yaml:"enableDuplicateCheck"`
	DuplicateCheckInterval int     `yaml:"duplicateCheckInterval"`
}

type ServerConfig struct {
	Port           int        `yaml:"port"`
	CompanyName    string     `yaml:"companyName"`
	CompanyLogoURL string     `yaml:"companyLogoUrl"`
	CORS           CORSConfig `yaml:"cors"`
	// TrustedProxies are the CIDR blocks of the reverse proxies that sit in front
	// of the application. Gin's own default is 0.0.0.0/0 and ::/0, which trusts
	// every peer and makes ClientIP() return the leftmost X-Forwarded-For value -
	// a header any caller can set.
	TrustedProxies []string `yaml:"trustedProxies"`
	// TrustedPlatform names an edge that overwrites rather than appends the real
	// client address, for example "cloudflare". When set it takes precedence over
	// X-Forwarded-For entirely.
	TrustedPlatform string          `yaml:"trustedPlatform"`
	RateLimit       RateLimitConfig `yaml:"rateLimit"`
}

// defaultTrustedProxies covers loopback, RFC1918, IPv6 unique-local and
// link-local ranges. That is the shape of almost every real deployment - a
// sidecar tunnel, a compose network, a local nginx - and a client on the public
// internet cannot present one of these addresses as its direct peer, so
// X-Forwarded-For stays honest. When the app is exposed directly the peer is a
// public address, is not trusted, and Gin falls back to it.
var defaultTrustedProxies = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

func (s ServerConfig) TrustedProxiesOrDefault() []string {
	proxies := make([]string, 0, len(s.TrustedProxies))
	for _, proxy := range s.TrustedProxies {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return defaultTrustedProxies
	}
	return proxies
}

// TrustedPlatformHeader resolves the configured platform name to the header Gin
// should read the client address from. Recognised names map to Gin's own
// constants; any other non-empty value is passed through as a literal header
// name, which is what Gin's TrustedPlatform field expects.
func (s ServerConfig) TrustedPlatformHeader() string {
	platform := strings.TrimSpace(s.TrustedPlatform)
	if platform == "" {
		return ""
	}
	switch strings.ToLower(platform) {
	case "cloudflare", "cf":
		return "CF-Connecting-IP"
	case "fly.io", "flyio", "fly-io":
		return "Fly-Client-IP"
	case "google-app-engine", "appengine", "gae":
		return "X-Appengine-Remote-Addr"
	default:
		return platform
	}
}

func (s ServerConfig) Address() string {
	if s.Port <= 0 {
		return ":8080"
	}
	return fmt.Sprintf(":%d", s.Port)
}

type CORSConfig struct {
	// AllowedOrigins 是允许浏览器跨域访问的 Origin 白名单，必须包含协议和域名。
	// 留空表示不允许跨域请求；同源请求通常不会携带 Origin，不受影响。
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

// RateLimitConfig bounds how often one client address may call the public,
// unauthenticated endpoints. It deliberately does not cover channel webhooks,
// websockets or authenticated dashboard routes: a platform that receives a 429
// from a webhook endpoint stops retrying and eventually disables the webhook.
type RateLimitConfig struct {
	// Enabled defaults to true. Set it to false to switch the limits off without
	// taking them out of the route table.
	Enabled *bool `yaml:"enabled"`
	// WindowSeconds is the length of the counting window. Zero or negative means
	// one minute.
	WindowSeconds int `yaml:"windowSeconds"`
}

func (r RateLimitConfig) IsEnabled() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

func (r RateLimitConfig) WindowSecondsOrDefault() int {
	if r.WindowSeconds <= 0 {
		return 60
	}
	return r.WindowSeconds
}

type DBConfig struct {
	Type                   string `yaml:"type"`
	DSN                    string `yaml:"dsn"`
	MaxIdleConns           int    `yaml:"maxIdleConns"`
	MaxOpenConns           int    `yaml:"maxOpenConns"`
	ConnMaxIdleTimeSeconds int    `yaml:"connMaxIdleTimeSeconds"`
	ConnMaxLifetimeSeconds int    `yaml:"connMaxLifetimeSeconds"`
}

type LoggerConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AddSource bool   `yaml:"addSource"`
}

type AuthConfig struct {
	PasswordLoginEnabled *bool `yaml:"passwordLoginEnabled"`
	TokenTTLHours        int   `yaml:"tokenTTLHours"`
	MaxFailedAttempts    int   `yaml:"maxFailedAttempts"`
	// MaxFailedAttemptsPerIP bounds failures from one client address across every
	// username, which is what credential stuffing looks like. Zero or unset
	// derives four times MaxFailedAttempts; it is disabled when MaxFailedAttempts
	// is disabled.
	MaxFailedAttemptsPerIP int `yaml:"maxFailedAttemptsPerIP"`
	CredentialLockMinute   int `yaml:"credentialLockMinute"`
}

// MaxFailedAttemptsPerIPOrDefault derives the per-address threshold from the
// per-account one so that a deployment which only tunes MaxFailedAttempts still
// gets a coherent pair of limits.
func (a AuthConfig) MaxFailedAttemptsPerIPOrDefault() int {
	if a.MaxFailedAttemptsPerIP > 0 {
		return a.MaxFailedAttemptsPerIP
	}
	if a.MaxFailedAttempts <= 0 {
		return 0
	}
	return a.MaxFailedAttempts * 4
}

func (a AuthConfig) IsPasswordLoginEnabled() bool {
	if a.PasswordLoginEnabled == nil {
		return true
	}
	return *a.PasswordLoginEnabled
}

type CustomerSessionConfig struct {
	Secret                  string `yaml:"secret"`
	TTLMinutes              int    `yaml:"ttlMinutes"`
	RefreshThresholdMinutes int    `yaml:"refreshThresholdMinutes"`
}

func (c CustomerSessionConfig) TTL() int {
	if c.TTLMinutes <= 0 {
		return 120
	}
	return c.TTLMinutes
}

func (c CustomerSessionConfig) RefreshThreshold() int {
	if c.RefreshThresholdMinutes <= 0 {
		return 30
	}
	return c.RefreshThresholdMinutes
}

type StorageConfig struct {
	Default         enums.AssetProvider `yaml:"default"`
	MaxUploadSizeMB int64               `yaml:"maxUploadSizeMB"`
	Local           LocalStorageConfig  `yaml:"local"`
	OSS             OSSStorageConfig    `yaml:"oss"`
}

func (s StorageConfig) MaxUploadSizeBytes() int64 {
	if s.MaxUploadSizeMB <= 0 {
		return 5 << 20
	}
	return s.MaxUploadSizeMB << 20
}

func (s StorageConfig) MaxRequestBodySizeBytes() int64 {
	limit := s.MaxUploadSizeBytes()
	return limit + (1 << 20)
}

type LocalStorageConfig struct {
	Root    string `yaml:"root"`
	BaseURL string `yaml:"baseUrl"`
}

type OSSStorageConfig struct {
	Endpoint        string `yaml:"endpoint"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"accessKeyId"`
	AccessKeySecret string `yaml:"accessKeySecret"`
	BaseURL         string `yaml:"baseUrl"`
	Private         bool   `yaml:"private"`
	SignedURLExpire int    `yaml:"signedUrlExpireSeconds"`
}

type VectorDBConfig struct {
	Type    string                `yaml:"type"`
	Qdrant  QdrantVectorDBConfig  `yaml:"qdrant"`
	LanceDB LanceDBVectorDBConfig `yaml:"lancedb"`
}

type QdrantVectorDBConfig struct {
	Host     string `yaml:"host"`
	GrpcPort int    `yaml:"grpcPort"`
	APIKey   string `yaml:"apiKey"`
	UseTLS   bool   `yaml:"useTls"`
}

type LanceDBVectorDBConfig struct {
	Path string `yaml:"path"`
}

type MCPConfig struct {
	Enabled bool                       `yaml:"enabled"`
	Servers map[string]MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Endpoint  string            `yaml:"endpoint"`
	TimeoutMS int               `yaml:"timeoutMs"`
	Headers   map[string]string `yaml:"headers"`
}

type OIDCConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Issuer       string   `yaml:"issuer"`
	ClientID     string   `yaml:"clientId"`
	ClientSecret string   `yaml:"clientSecret"`
	RedirectURL  string   `yaml:"redirectUrl"`
	StateSecret  string   `yaml:"stateSecret"`
	Scopes       []string `yaml:"scopes"`
}

// WxWorkConfig 定义企业微信接入配置。
//
// 当前主要用于后台管理台的企业微信登录流程：
// 1. /api/auth/wxwork/login 生成企业微信授权地址
// 2. 企业微信回调到 OAuthRedirect
// 3. 后端通过 code 换取企业成员身份并完成系统登录
//
// 其中 OAuthRedirect、CorpID、CorpSecret、AgentID 为登录流程核心配置。
type WxWorkConfig struct {
	// Enabled 表示是否启用企业微信登录能力。
	// false 时不会初始化企业微信 SDK，相关登录接口不可用。
	Enabled bool `yaml:"enabled"`
	// CorpID 为企业微信公司 ID，例如 wwxxxxxxxxxxxxxxxx。
	CorpID string `yaml:"corpId"`
	// CorpSecret 为企业微信应用 Secret，用于换取 access_token。
	CorpSecret string `yaml:"corpSecret"`
	// AgentID 为企业微信自建应用 AgentID。
	AgentID string `yaml:"agentId"`
	// OAuthRedirect 为企业微信网页授权回调地址。
	// 必须填写完整 URL，且通常指向后端接口 /api/auth/wxwork/callback。
	OAuthRedirect string `yaml:"oauthRedirect"`
	// StateSecret 为登录 state 的签名密钥，用于防止篡改和重放。
	// 建议填写独立随机字符串；留空时业务代码会退回使用 CorpSecret。
	StateSecret string `yaml:"stateSecret"`
	// RSAPrivateKey 为企业微信回调解密私钥。
	// 当前登录流程未使用，保留给消息回调等场景。
	RSAPrivateKey string `yaml:"rsaPrivateKey"`
	// Token 为企业微信回调 Token。
	// 当前登录流程未使用，保留给消息回调等场景。
	Token string `yaml:"token"`
	// EncodingAESKey 为企业微信消息加解密密钥。
	// 当前登录流程未使用，保留给消息回调等场景。
	EncodingAESKey string `yaml:"encodingAESKey"`
	// Notify 为企业微信应用消息通知配置。
	Notify WxWorkNotifyConfig `yaml:"notify"`
}

type WebhookConfig struct {
	OrgSyncSecret    string `yaml:"orgSyncSecret"`
	DOSOrgSyncSecret string `yaml:"dosOrgSyncSecret"`
}

// DiscordConfig holds deployment-wide Discord bot credentials. A channel may
// carry its own bot token, which takes precedence; these are the fallback for a
// single shared bot.
type DiscordConfig struct {
	ClientID     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	BotToken     string `yaml:"botToken"`
	PublicKey    string `yaml:"publicKey"`
}

// SlackConfig holds deployment-wide Slack app credentials. A channel may carry
// its own bot token and signing secret, which take precedence; these are the
// fallback for a single shared Slack app.
type SlackConfig struct {
	ClientID      string `yaml:"clientId"`
	ClientSecret  string `yaml:"clientSecret"`
	BotToken      string `yaml:"botToken"`
	SigningSecret string `yaml:"signingSecret"`
}

// SlackApp resolves the Slack app credentials for one channel. A channel value
// wins over the deployment-wide one, so a single deployment can serve several
// workspaces while still having a default app.
func (c Config) SlackApp(channelBotToken, channelSigningSecret string) SlackConfig {
	return SlackConfig{
		ClientID:      strings.TrimSpace(c.Slack.ClientID),
		ClientSecret:  strings.TrimSpace(c.Slack.ClientSecret),
		BotToken:      firstNonBlank(channelBotToken, c.Slack.BotToken),
		SigningSecret: firstNonBlank(channelSigningSecret, c.Slack.SigningSecret),
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func Load(path string) (*Config, error) {
	loadDotEnv(path)

	v := viper.New()
	bindConfigDefaults(v)

	if strings.TrimSpace(path) != "" {
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("AGENT_DESK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindEnvironmentAliases(v)

	if strings.TrimSpace(path) != "" {
		if err := v.ReadInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !os.IsNotExist(err) && !errors.As(err, &configFileNotFoundError) {
				return nil, err
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	normalizeLoadedConfig(cfg)
	return cfg, nil
}

func loadDotEnv(configPath string) {
	if envFile := os.Getenv("AGENT_DESK_ENV_FILE"); envFile != "" {
		_ = gotenv.Load(envFile)
		return
	}
	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		_ = gotenv.Load(envFile)
		return
	}
	_ = gotenv.Load(".env")
	_ = gotenv.Load("../.env")
	_ = gotenv.Load("../../.env")
	if configPath != "" {
		dir := filepath.Dir(configPath)
		if dir != "." && dir != "" {
			_ = gotenv.Load(filepath.Join(dir, ".env"))
			_ = gotenv.Load(filepath.Join(dir, "..", ".env"))
		}
	}
}

func bindConfigDefaults(v *viper.Viper) {
	v.SetDefault("language", "zh-CN")
	v.SetDefault("server.port", 8083)
	v.SetDefault("server.companyName", "")
	v.SetDefault("server.companyLogoUrl", "")
	v.SetDefault("server.cors.allowedOrigins", []string{})
	v.SetDefault("server.trustedProxies", []string{})
	v.SetDefault("server.trustedPlatform", "")
	v.SetDefault("server.rateLimit.windowSeconds", 60)
	v.SetDefault("db.type", "sqlite")
	v.SetDefault("db.dsn", "file:./data/app.db?_busy_timeout=5000")
	v.SetDefault("db.maxIdleConns", 5)
	v.SetDefault("db.maxOpenConns", 20)
	v.SetDefault("db.connMaxIdleTimeSeconds", 300)
	v.SetDefault("db.connMaxLifetimeSeconds", 1800)
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "text")
	v.SetDefault("logger.addSource", false)
	v.SetDefault("auth.tokenTTLHours", 12)
	v.SetDefault("auth.maxFailedAttempts", 5)
	v.SetDefault("auth.maxFailedAttemptsPerIP", 0)
	v.SetDefault("auth.credentialLockMinute", 15)
	v.SetDefault("customerSession.ttlMinutes", 120)
	v.SetDefault("customerSession.refreshThresholdMinutes", 30)
	v.SetDefault("storage.default", "local")
	v.SetDefault("storage.maxUploadSizeMB", 20)
	v.SetDefault("storage.local.root", "data/storage")
	v.SetDefault("storage.local.baseUrl", "/storage")
	v.SetDefault("vectorDB.type", "qdrant")
	v.SetDefault("vectorDB.qdrant.host", "127.0.0.1")
	v.SetDefault("vectorDB.qdrant.grpcPort", 6334)
	v.SetDefault("mcp.enabled", true)
	v.SetDefault("discord.clientId", "")
	v.SetDefault("discord.clientSecret", "")
	v.SetDefault("discord.botToken", "")
	v.SetDefault("discord.publicKey", "")
	v.SetDefault("slack.clientId", "")
	v.SetDefault("slack.clientSecret", "")
	v.SetDefault("slack.botToken", "")
	v.SetDefault("slack.signingSecret", "")
}

func bindEnvironmentAliases(v *viper.Viper) {
	// Prefixed AGENT_DESK_* aliases are listed first so that ambient legacy
	// variables (PORT, DATABASE_URL, ...) cannot silently override the
	// documented configuration.
	_ = v.BindEnv("server.port", "AGENT_DESK_SERVER_PORT", "PORT", "SERVER_PORT")
	_ = v.BindEnv("server.companyName", "AGENT_DESK_SERVER_COMPANYNAME", "COMPANY_NAME", "NEXT_PUBLIC_COMPANY_NAME", "BRAND_NAME", "BRAND_COMPANY_NAME")
	_ = v.BindEnv("server.companyLogoUrl", "AGENT_DESK_SERVER_COMPANYLOGOURL", "COMPANY_LOGO_URL", "NEXT_PUBLIC_COMPANY_LOGO_URL", "BRAND_LOGO_URL")
	_ = v.BindEnv("server.trustedProxies", "AGENT_DESK_SERVER_TRUSTEDPROXIES", "TRUSTED_PROXIES")
	_ = v.BindEnv("server.trustedPlatform", "AGENT_DESK_SERVER_TRUSTEDPLATFORM", "TRUSTED_PLATFORM")
	_ = v.BindEnv("server.rateLimit.enabled", "AGENT_DESK_SERVER_RATELIMIT_ENABLED", "RATE_LIMIT_ENABLED")
	_ = v.BindEnv("server.rateLimit.windowSeconds", "AGENT_DESK_SERVER_RATELIMIT_WINDOWSECONDS", "RATE_LIMIT_WINDOW_SECONDS")
	_ = v.BindEnv("db.type", "AGENT_DESK_DB_TYPE", "DATABASE_TYPE", "DB_TYPE")
	_ = v.BindEnv("db.dsn", "AGENT_DESK_DB_DSN", "DATABASE_URL", "DB_DSN")
	_ = v.BindEnv("auth.passwordLoginEnabled", "AGENT_DESK_AUTH_PASSWORDLOGINENABLED", "PASSWORD_LOGIN_ENABLED")
	_ = v.BindEnv("auth.tokenTTLHours", "AGENT_DESK_AUTH_TOKENTTLHOURS", "AUTH_TOKEN_TTL_HOURS")
	_ = v.BindEnv("customerSession.secret", "AGENT_DESK_CUSTOMERSESSION_SECRET", "CUSTOMER_SESSION_SECRET", "SESSION_SECRET", "JWT_SECRET")
	_ = v.BindEnv("storage.default", "AGENT_DESK_STORAGE_DEFAULT", "STORAGE_DEFAULT", "STORAGE_TYPE")
	_ = v.BindEnv("storage.local.root", "AGENT_DESK_STORAGE_LOCAL_ROOT", "STORAGE_LOCAL_ROOT")
	_ = v.BindEnv("storage.local.baseUrl", "AGENT_DESK_STORAGE_LOCAL_BASEURL", "STORAGE_LOCAL_BASE_URL")
	_ = v.BindEnv("vectorDB.type", "AGENT_DESK_VECTORDB_TYPE", "VECTOR_DB_TYPE")
	_ = v.BindEnv("vectorDB.qdrant.host", "AGENT_DESK_VECTORDB_QDRANT_HOST", "QDRANT_HOST")
	_ = v.BindEnv("vectorDB.qdrant.grpcPort", "AGENT_DESK_VECTORDB_QDRANT_GRPCPORT", "QDRANT_GRPC_PORT", "QDRANT_PORT")
	_ = v.BindEnv("vectorDB.qdrant.apiKey", "AGENT_DESK_VECTORDB_QDRANT_APIKEY", "QDRANT_API_KEY")
	_ = v.BindEnv("oidc.enabled", "AGENT_DESK_OIDC_ENABLED", "OIDC_ENABLED")
	_ = v.BindEnv("oidc.issuer", "AGENT_DESK_OIDC_ISSUER", "OIDC_ISSUER")
	_ = v.BindEnv("oidc.clientId", "AGENT_DESK_OIDC_CLIENTID", "OIDC_CLIENT_ID", "CUSTOM_OAUTH_CLIENT_ID")
	_ = v.BindEnv("oidc.clientSecret", "AGENT_DESK_OIDC_CLIENTSECRET", "OIDC_CLIENT_SECRET", "CUSTOM_OAUTH_CLIENT_SECRET")
	_ = v.BindEnv("oidc.redirectUrl", "AGENT_DESK_OIDC_REDIRECTURL", "OIDC_REDIRECT_URL", "CUSTOM_OAUTH_REDIRECT_URI")
	_ = v.BindEnv("webhook.orgSyncSecret", "AGENT_DESK_WEBHOOK_ORGSYNCSECRET", "ORG_SYNC_SECRET", "WEBHOOK_SECRET")
	_ = v.BindEnv("discord.clientId", "AGENT_DESK_DISCORD_CLIENTID", "DISCORD_CLIENT_ID")
	_ = v.BindEnv("discord.clientSecret", "AGENT_DESK_DISCORD_CLIENTSECRET", "DISCORD_CLIENT_SECRET")
	_ = v.BindEnv("discord.botToken", "AGENT_DESK_DISCORD_BOTTOKEN", "DISCORD_BOT_TOKEN")
	_ = v.BindEnv("discord.publicKey", "AGENT_DESK_DISCORD_PUBLICKEY", "DISCORD_PUBLIC_KEY")
	_ = v.BindEnv("slack.clientId", "AGENT_DESK_SLACK_CLIENTID", "SLACK_CLIENT_ID")
	_ = v.BindEnv("slack.clientSecret", "AGENT_DESK_SLACK_CLIENTSECRET", "SLACK_CLIENT_SECRET")
	_ = v.BindEnv("slack.botToken", "AGENT_DESK_SLACK_BOTTOKEN", "SLACK_BOT_TOKEN")
	_ = v.BindEnv("slack.signingSecret", "AGENT_DESK_SLACK_SIGNINGSECRET", "SLACK_SIGNING_SECRET")
}

func normalizeLoadedConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.DB.Type == "sqlite" && (strings.HasPrefix(cfg.DB.DSN, "postgres://") || strings.HasPrefix(cfg.DB.DSN, "postgresql://")) {
		cfg.DB.Type = "postgres"
	} else if cfg.DB.Type == "sqlite" && strings.Contains(cfg.DB.DSN, "@tcp(") {
		cfg.DB.Type = "mysql"
	}
}
