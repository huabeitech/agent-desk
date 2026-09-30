package wxwork

import (
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/i18nx"
	"strings"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/work"
	"github.com/silenceper/wechat/v2/work/addresslist"
	wxconfig "github.com/silenceper/wechat/v2/work/config"
	"github.com/silenceper/wechat/v2/work/oauth"
)

var (
	defaultWork *work.Work
	appWorks    map[string]*work.Work
	wxCfg       config.WxWorkConfig
)

type LoginUser struct {
	CorpID         string                       `json:"corpId"`
	UserID         string                       `json:"userId"`
	OpenID         string                       `json:"openId,omitempty"`
	ExternalUserID string                       `json:"externalUserId,omitempty"`
	UserTicket     string                       `json:"userTicket,omitempty"`
	Name           string                       `json:"name,omitempty"`
	Avatar         string                       `json:"avatar,omitempty"`
	Mobile         string                       `json:"mobile,omitempty"`
	Email          string                       `json:"email,omitempty"`
	BizMail        string                       `json:"bizMail,omitempty"`
	UserInfo       *oauth.GetUserInfoResponse   `json:"userInfo,omitempty"`
	UserDetail     *oauth.GetUserDetailResponse `json:"userDetail,omitempty"`
	UserProfile    *addresslist.UserGetResponse `json:"userProfile,omitempty"`
}

func Init() {
	defaultWork = nil
	appWorks = nil
	wxCfg = config.WxWorkConfig{}
	cfg := config.Current()
	if !cfg.WxWork.Enabled {
		return
	}
	wxCfg = cfg.WxWork
	if strings.TrimSpace(wxCfg.CorpID) == "" {
		return
	}
	apps := wxCfg.NormalizedAPIApps()
	if len(apps) == 0 {
		return
	}

	// 每个应用使用各自的 corpSecret 构建独立客户端，
	// SDK 令牌缓存随客户端实例隔离，不同应用的 access_token 天然分开缓存。
	appWorks = make(map[string]*work.Work, len(apps))
	for i := range apps {
		app := apps[i]
		cli := work.NewWork(&wxconfig.Config{
			CorpID:         wxCfg.CorpID,
			CorpSecret:     app.CorpSecret,
			AgentID:        app.AgentID,
			RasPrivateKey:  wxCfg.RSAPrivateKey,
			Token:          wxCfg.Token,
			EncodingAESKey: wxCfg.EncodingAESKey,
			Cache:          cache.NewMemory(),
		})
		appWorks[app.AgentID] = cli
		if defaultWork == nil {
			defaultWork = cli
		}
	}
}

func Enabled() bool {
	return defaultWork != nil && wxCfg.Enabled
}

func StateSecret() string {
	if secret := strings.TrimSpace(wxCfg.StateSecret); secret != "" {
		return secret
	}
	if secret := strings.TrimSpace(wxCfg.CorpSecret); secret != "" {
		return secret
	}
	if apps := wxCfg.NormalizedAPIApps(); len(apps) > 0 {
		return strings.TrimSpace(apps[0].CorpSecret)
	}
	return ""
}

func GetWorkCli() *work.Work {
	return defaultWork
}

// GetWorkCliByAgentID 按 agentId 返回对应应用的企微客户端；
// 各应用客户端持有独立的 corpSecret 与 token 缓存，不能混用。
func GetWorkCliByAgentID(agentID string) (*work.Work, error) {
	agentID = strings.TrimSpace(agentID)
	if cli := appWorks[agentID]; cli != nil {
		return cli, nil
	}
	return nil, i18nx.Errorf("error.wxwork.appNotConfigured", agentID)
}

// DefaultAgentID 返回默认（第一个已配置）应用的 agentId；
// 供 OAuth 登录等不区分具体应用的流程使用。
func DefaultAgentID() string {
	if agentID := strings.TrimSpace(wxCfg.AgentID); agentID != "" {
		return agentID
	}
	if apps := wxCfg.NormalizedAPIApps(); len(apps) > 0 {
		return strings.TrimSpace(apps[0].AgentID)
	}
	return ""
}
