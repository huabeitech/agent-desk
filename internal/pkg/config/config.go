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

// WxWorkApiAppConfig 定义一个企业微信自建应用（agentId + corpSecret）。
// 不同应用的 corpSecret 换取各自独立的 access_token，必须按应用分开缓存与使用。
type WxWorkApiAppConfig struct {
	// AgentID 为企业微信自建应用 AgentID。
	AgentID string `yaml:"agentId"`
	// CorpSecret 为该应用的 Secret，用于换取该应用的 access_token。
	CorpSecret string `yaml:"corpSecret"`
}

type ServerConfig struct {
	Port           int        `yaml:"port"`
	CompanyName    string     `yaml:"companyName"`
	CompanyLogoURL string     `yaml:"companyLogoUrl"`
	CORS           CORSConfig `yaml:"cors"`
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
	CredentialLockMinute int   `yaml:"credentialLockMinute"`
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
	// 仅用于单应用的旧配置；多应用场景请使用 APIApps。
	CorpSecret string `yaml:"corpSecret"`
	// AgentID 为企业微信自建应用 AgentID。
	// 仅用于单应用的旧配置；多应用场景请使用 APIApps。
	AgentID string `yaml:"agentId"`
	// APIApps 为可用的企业微信自建应用列表，每项包含 agentId 与对应 corpSecret。
	// 接入渠道按 agentId 选择应用，使用该应用的 corpSecret 换取独立 access_token。
	APIApps []WxWorkApiAppConfig `yaml:"apiApps"`
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

// NormalizedAPIApps 返回去除空白并过滤掉不完整项后的 apiApps。
// 当 apiApps 为空且配置了顶层 corpSecret 时，回退为单应用列表，兼容旧配置。
func (c WxWorkConfig) NormalizedAPIApps() []WxWorkApiAppConfig {
	apps := make([]WxWorkApiAppConfig, 0, len(c.APIApps))
	for _, app := range c.APIApps {
		agentID := strings.TrimSpace(app.AgentID)
		corpSecret := strings.TrimSpace(app.CorpSecret)
		if agentID == "" || corpSecret == "" {
			continue
		}
		apps = append(apps, WxWorkApiAppConfig{AgentID: agentID, CorpSecret: corpSecret})
	}
	if len(apps) == 0 {
		if corpSecret := strings.TrimSpace(c.CorpSecret); corpSecret != "" {
			apps = append(apps, WxWorkApiAppConfig{
				AgentID:    strings.TrimSpace(c.AgentID),
				CorpSecret: corpSecret,
			})
		}
	}
	return apps
}

// FindAPIApp 按 agentID 查找已配置的应用。
func (c WxWorkConfig) FindAPIApp(agentID string) (WxWorkApiAppConfig, bool) {
	agentID = strings.TrimSpace(agentID)
	for _, app := range c.NormalizedAPIApps() {
		if app.AgentID == agentID {
			return app, true
		}
	}
	return WxWorkApiAppConfig{}, false
}

type WebhookConfig struct {
	OrgSyncSecret    string `yaml:"orgSyncSecret"`
	DOSOrgSyncSecret string `yaml:"dosOrgSyncSecret"`
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
}

func bindEnvironmentAliases(v *viper.Viper) {
	_ = v.BindEnv("server.port", "PORT", "SERVER_PORT", "AGENT_DESK_SERVER_PORT")
	_ = v.BindEnv("server.companyName", "COMPANY_NAME", "NEXT_PUBLIC_COMPANY_NAME", "BRAND_NAME", "BRAND_COMPANY_NAME", "AGENT_DESK_SERVER_COMPANYNAME")
	_ = v.BindEnv("server.companyLogoUrl", "COMPANY_LOGO_URL", "NEXT_PUBLIC_COMPANY_LOGO_URL", "BRAND_LOGO_URL", "AGENT_DESK_SERVER_COMPANYLOGOURL")
	_ = v.BindEnv("db.type", "DATABASE_TYPE", "DB_TYPE", "AGENT_DESK_DB_TYPE")
	_ = v.BindEnv("db.dsn", "DATABASE_URL", "DB_DSN", "AGENT_DESK_DB_DSN")
	_ = v.BindEnv("auth.passwordLoginEnabled", "PASSWORD_LOGIN_ENABLED", "AGENT_DESK_AUTH_PASSWORDLOGINENABLED")
	_ = v.BindEnv("auth.tokenTTLHours", "AUTH_TOKEN_TTL_HOURS", "AGENT_DESK_AUTH_TOKENTTLHOURS")
	_ = v.BindEnv("customerSession.secret", "CUSTOMER_SESSION_SECRET", "SESSION_SECRET", "JWT_SECRET", "AGENT_DESK_CUSTOMERSESSION_SECRET")
	_ = v.BindEnv("storage.default", "STORAGE_DEFAULT", "STORAGE_TYPE", "AGENT_DESK_STORAGE_DEFAULT")
	_ = v.BindEnv("storage.local.root", "STORAGE_LOCAL_ROOT", "AGENT_DESK_STORAGE_LOCAL_ROOT")
	_ = v.BindEnv("storage.local.baseUrl", "STORAGE_LOCAL_BASE_URL", "AGENT_DESK_STORAGE_LOCAL_BASEURL")
	_ = v.BindEnv("vectorDB.type", "VECTOR_DB_TYPE", "AGENT_DESK_VECTORDB_TYPE")
	_ = v.BindEnv("vectorDB.qdrant.host", "QDRANT_HOST", "AGENT_DESK_VECTORDB_QDRANT_HOST")
	_ = v.BindEnv("vectorDB.qdrant.grpcPort", "QDRANT_GRPC_PORT", "QDRANT_PORT", "AGENT_DESK_VECTORDB_QDRANT_GRPCPORT")
	_ = v.BindEnv("vectorDB.qdrant.apiKey", "QDRANT_API_KEY", "AGENT_DESK_VECTORDB_QDRANT_APIKEY")
	_ = v.BindEnv("oidc.enabled", "OIDC_ENABLED", "AGENT_DESK_OIDC_ENABLED")
	_ = v.BindEnv("oidc.issuer", "OIDC_ISSUER", "AGENT_DESK_OIDC_ISSUER")
	_ = v.BindEnv("oidc.clientId", "OIDC_CLIENT_ID", "CUSTOM_OAUTH_CLIENT_ID", "AGENT_DESK_OIDC_CLIENTID")
	_ = v.BindEnv("oidc.clientSecret", "OIDC_CLIENT_SECRET", "CUSTOM_OAUTH_CLIENT_SECRET", "AGENT_DESK_OIDC_CLIENTSECRET")
	_ = v.BindEnv("oidc.redirectUrl", "OIDC_REDIRECT_URL", "CUSTOM_OAUTH_REDIRECT_URI", "AGENT_DESK_OIDC_REDIRECTURL")
	_ = v.BindEnv("webhook.orgSyncSecret", "ORG_SYNC_SECRET", "WEBHOOK_SECRET", "AGENT_DESK_WEBHOOK_ORGSYNCSECRET")
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
