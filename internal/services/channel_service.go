package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"
	"agent-desk/internal/wxwork"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/silenceper/wechat/v2/work"
	"github.com/silenceper/wechat/v2/work/kf"
)

var ChannelService = newChannelService()

func newChannelService() *channelService {
	return &channelService{}
}

type channelService struct {
}

func (s *channelService) Get(id int64) *models.Channel {
	return repositories.ChannelRepository.Get(sqls.DB(), id)
}

func (s *channelService) Take(where ...interface{}) *models.Channel {
	return repositories.ChannelRepository.Take(sqls.DB(), where...)
}

func (s *channelService) Find(cnd *sqls.Cnd) []models.Channel {
	return repositories.ChannelRepository.Find(sqls.DB(), cnd)
}

func (s *channelService) FindOne(cnd *sqls.Cnd) *models.Channel {
	return repositories.ChannelRepository.FindOne(sqls.DB(), cnd)
}

func (s *channelService) FindPageByParams(params *params.QueryParams) (list []models.Channel, paging *sqls.Paging) {
	return repositories.ChannelRepository.FindPageByParams(sqls.DB(), params)
}

func (s *channelService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Channel, paging *sqls.Paging) {
	return repositories.ChannelRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *channelService) Count(cnd *sqls.Cnd) int64 {
	return repositories.ChannelRepository.Count(sqls.DB(), cnd)
}

func (s *channelService) Create(t *models.Channel) error {
	return repositories.ChannelRepository.Create(sqls.DB(), t)
}

func (s *channelService) Update(t *models.Channel) error {
	return repositories.ChannelRepository.Update(sqls.DB(), t)
}

func (s *channelService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.ChannelRepository.Updates(sqls.DB(), id, columns)
}

func (s *channelService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.ChannelRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *channelService) CreateChannel(req request.CreateChannelRequest, operator *dto.AuthPrincipal) (*models.Channel, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item, err := s.buildChannelModel(0, req)
	if err != nil {
		return nil, err
	}
	item.AuditFields = utils.BuildAuditFields(operator)
	if err := repositories.ChannelRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *channelService) UpdateChannel(req request.UpdateChannelRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	current := s.Get(req.ID)
	if current == nil || current.Status == enums.StatusDeleted {
		return errorsx.InvalidParamI18n("error.e0208")
	}
	item, err := s.buildChannelModel(req.ID, req.CreateChannelRequest)
	if err != nil {
		return err
	}
	columns := map[string]any{
		"channel_type":             item.ChannelType,
		"channel_id":               item.ChannelID,
		"ai_agent_id":              item.AIAgentID,
		"ai_agent_rollout_percent": item.AIAgentRolloutPercent,
		"name":                     item.Name,
		"config_json":              item.ConfigJSON,
		"status":                   item.Status,
		"remark":                   item.Remark,
		"update_user_id":           operator.UserID,
		"update_user_name":         operator.Username,
		"updated_at":               time.Now(),
	}
	if item.AIAgentRolloutPercent != current.AIAgentRolloutPercent {
		columns["previous_ai_agent_rollout_percent"] = current.AIAgentRolloutPercent
	}
	return repositories.ChannelRepository.Updates(sqls.DB(), req.ID, columns)
}

// RollbackChannelAIAgentRollout restores the last channel-level rollout value
// and swaps it into history so the action itself is reversible.
func (s *channelService) RollbackChannelAIAgentRollout(id int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if id <= 0 {
		return errorsx.InvalidParam("channel id is required")
	}
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		channel := repositories.ChannelRepository.Get(ctx.Tx, id)
		if channel == nil || channel.Status == enums.StatusDeleted {
			return errorsx.InvalidParamI18n("error.e0208")
		}
		if channel.PreviousAIAgentRolloutPercent < 1 || channel.PreviousAIAgentRolloutPercent > 100 {
			return errorsx.InvalidParam("channel rollout has no previous value to restore")
		}
		return repositories.ChannelRepository.Updates(ctx.Tx, channel.ID, map[string]any{
			"ai_agent_rollout_percent":          channel.PreviousAIAgentRolloutPercent,
			"previous_ai_agent_rollout_percent": channel.AIAgentRolloutPercent,
			"update_user_id":                    operator.UserID,
			"update_user_name":                  operator.Username,
			"updated_at":                        time.Now(),
		})
	})
}

func (s *channelService) UpdateStatus(id int64, status int, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil || item.Status == enums.StatusDeleted {
		return errorsx.InvalidParamI18n("error.e0208")
	}
	if status != int(enums.StatusOk) && status != int(enums.StatusDisabled) {
		return errorsx.InvalidParamI18n("error.e0254")
	}
	return s.Updates(id, map[string]any{
		"status":           status,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	})
}

func (s *channelService) DeleteChannel(id int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil || item.Status == enums.StatusDeleted {
		return errorsx.InvalidParamI18n("error.e0208")
	}
	return s.Updates(id, map[string]any{
		"status":           enums.StatusDeleted,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	})
}

func (s *channelService) ParseWxWorkKFChannelConfig(raw string) (*dto.WxWorkKFChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &dto.WxWorkKFChannelConfig{}, nil
	}
	cfg := &dto.WxWorkKFChannelConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, err
	}
	cfg.OpenKfID = strings.TrimSpace(cfg.OpenKfID)
	cfg.AgentID = strings.TrimSpace(cfg.AgentID)
	return cfg, nil
}

// GetWxWorkCliByChannel 返回渠道绑定 agentId 对应的企微客户端。
// 不同渠道可绑定不同应用，客户端与 access_token 均按应用隔离。
func (s *channelService) GetWxWorkCliByChannel(channel *models.Channel) (*work.Work, error) {
	if channel == nil {
		return nil, errorsx.InvalidParamI18n("error.wxwork.appNotConfigured", "")
	}
	cfg, err := s.ParseWxWorkKFChannelConfig(channel.ConfigJSON)
	if err != nil {
		return nil, err
	}
	return wxwork.GetWorkCliByAgentID(cfg.AgentID)
}

// ListWxWorkApiApps 返回配置文件中可用的企业微信应用（agentId 列表），供渠道表单选择。
func (s *channelService) ListWxWorkApiApps() []response.WxWorkApiAppResponse {
	apps := config.Current().WxWork.NormalizedAPIApps()
	ret := make([]response.WxWorkApiAppResponse, 0, len(apps))
	for _, app := range apps {
		ret = append(ret, response.WxWorkApiAppResponse{AgentID: app.AgentID})
	}
	return ret
}

func (s *channelService) ListWxWorkKFAccounts() ([]response.WxWorkKFAccountResponse, error) {
	if !wxwork.Enabled() || wxwork.GetWorkCli() == nil {
		return nil, errorsx.BusinessErrorI18n(1, "error.wxwork.configIncomplete")
	}
	cli, err := wxwork.GetWorkCli().GetKF()
	if err != nil {
		return nil, err
	}

	const limit = 100
	accounts := make([]response.WxWorkKFAccountResponse, 0)
	for offset := 0; ; offset += limit {
		result, err := cli.AccountPaging(&kf.AccountPagingRequest{
			Offset: offset,
			Limit:  limit,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range result.AccountList {
			openKfID := strings.TrimSpace(item.OpenKFID)
			if openKfID == "" {
				continue
			}
			accounts = append(accounts, response.WxWorkKFAccountResponse{
				OpenKfID:        openKfID,
				Name:            strings.TrimSpace(item.Name),
				Avatar:          strings.TrimSpace(item.Avatar),
				ManagePrivilege: item.ManagePrivilege,
			})
		}
		if len(result.AccountList) < limit {
			break
		}
	}
	return accounts, nil
}

func (s *channelService) ParseWebChannelConfig(raw string) (*dto.WebChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	cfg := &dto.WebChannelConfig{
		Title:      "Support",
		Subtitle:   "How can we help?",
		ThemeColor: "#2563eb",
		Position:   "right",
		Width:      "380px",
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	cfg.Title = strings.TrimSpace(cfg.Title)
	if cfg.Title == "" {
		cfg.Title = "Support"
	}
	cfg.Subtitle = strings.TrimSpace(cfg.Subtitle)
	cfg.ThemeColor = strings.TrimSpace(cfg.ThemeColor)
	if cfg.ThemeColor == "" {
		cfg.ThemeColor = "#2563eb"
	}
	cfg.Position = strings.TrimSpace(cfg.Position)
	if cfg.Position == "" {
		cfg.Position = "right"
	}
	if cfg.Position != "left" && cfg.Position != "right" {
		return nil, errorsx.InvalidParamI18n("error.e0059")
	}
	cfg.Width = strings.TrimSpace(cfg.Width)
	if cfg.Width == "" {
		cfg.Width = "380px"
	}
	cfg.UserTokenSecret = strings.TrimSpace(cfg.UserTokenSecret)
	return cfg, nil
}

func (s *channelService) ParseWechatMPChannelConfig(raw string) (*dto.WechatMPChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	cfg := &dto.WechatMPChannelConfig{
		Title:      "Official Account Support",
		Subtitle:   "How can we help?",
		ThemeColor: "#2563eb",
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	cfg.Title = strings.TrimSpace(cfg.Title)
	if cfg.Title == "" {
		cfg.Title = "Official Account Support"
	}
	cfg.Subtitle = strings.TrimSpace(cfg.Subtitle)
	cfg.ThemeColor = strings.TrimSpace(cfg.ThemeColor)
	if cfg.ThemeColor == "" {
		cfg.ThemeColor = "#2563eb"
	}
	cfg.UserTokenSecret = strings.TrimSpace(cfg.UserTokenSecret)
	return cfg, nil
}

func (s *channelService) ParseTelegramChannelConfig(raw string) (*dto.TelegramChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	cfg := &dto.TelegramChannelConfig{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	cfg.BotToken = strings.TrimSpace(cfg.BotToken)
	cfg.BotUsername = strings.TrimSpace(cfg.BotUsername)
	cfg.WebhookSecret = strings.TrimSpace(cfg.WebhookSecret)
	cfg.WelcomeMessage = strings.TrimSpace(cfg.WelcomeMessage)
	return cfg, nil
}

func (s *channelService) ParseZaloOAChannelConfig(raw string) (*dto.ZaloOAChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	cfg := &dto.ZaloOAChannelConfig{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	cfg.AppID = strings.TrimSpace(cfg.AppID)
	cfg.OAID = strings.TrimSpace(cfg.OAID)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.AccessToken = strings.TrimSpace(cfg.AccessToken)
	cfg.RefreshToken = strings.TrimSpace(cfg.RefreshToken)
	cfg.WebhookSecret = strings.TrimSpace(cfg.WebhookSecret)
	cfg.WelcomeMessage = strings.TrimSpace(cfg.WelcomeMessage)
	return cfg, nil
}

func (s *channelService) ParseDiscordChannelConfig(raw string) (*dto.DiscordChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	cfg := &dto.DiscordChannelConfig{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	cfg.GuildID = strings.TrimSpace(cfg.GuildID)
	cfg.GuildName = strings.TrimSpace(cfg.GuildName)
	cfg.ChannelScope = strings.TrimSpace(cfg.ChannelScope)
	cfg.BotToken = strings.TrimSpace(cfg.BotToken)
	cfg.ApplicationID = strings.TrimSpace(cfg.ApplicationID)
	cfg.PublicKey = strings.TrimSpace(cfg.PublicKey)
	cfg.WebhookSecret = strings.TrimSpace(cfg.WebhookSecret)
	cfg.WelcomeMessage = strings.TrimSpace(cfg.WelcomeMessage)
	return cfg, nil
}

func (s *channelService) GetUserTokenSecret(channel *models.Channel) string {
	if channel == nil {
		return ""
	}
	switch channel.ChannelType {
	case enums.ChannelTypeWeb:
		cfg, err := s.ParseWebChannelConfig(channel.ConfigJSON)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(cfg.UserTokenSecret)
	case enums.ChannelTypeWechatMP:
		cfg, err := s.ParseWechatMPChannelConfig(channel.ConfigJSON)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(cfg.UserTokenSecret)
	default:
		return ""
	}
}

func (s *channelService) ResetUserTokenSecret(channelID int64, operator *dto.AuthPrincipal) (string, error) {
	if operator == nil {
		return "", errorsx.UnauthorizedI18n("error.auth.expired")
	}
	channel := s.Get(channelID)
	if channel == nil || channel.Status == enums.StatusDeleted {
		return "", errorsx.InvalidParamI18n("error.e0208")
	}
	if channel.ChannelType != enums.ChannelTypeWeb && channel.ChannelType != enums.ChannelTypeWechatMP {
		return "", errorsx.InvalidParamI18n("error.e0196")
	}
	secret, err := generateUserTokenSecret()
	if err != nil {
		return "", err
	}
	var configJSON string
	switch channel.ChannelType {
	case enums.ChannelTypeWeb:
		cfg, err := s.ParseWebChannelConfig(channel.ConfigJSON)
		if err != nil {
			return "", err
		}
		cfg.UserTokenSecret = secret
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", err
		}
		configJSON = string(raw)
	case enums.ChannelTypeWechatMP:
		cfg, err := s.ParseWechatMPChannelConfig(channel.ConfigJSON)
		if err != nil {
			return "", err
		}
		cfg.UserTokenSecret = secret
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", err
		}
		configJSON = string(raw)
	}
	if err := repositories.ChannelRepository.Updates(sqls.DB(), channelID, map[string]any{
		"config_json":      configJSON,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
		"updated_at":       time.Now(),
	}); err != nil {
		return "", err
	}
	return secret, nil
}

func generateUserTokenSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *channelService) GetEnabledWxWorkKFChannelByOpenKfID(openKfID string) *models.Channel {
	openKfID = strings.TrimSpace(openKfID)
	if openKfID == "" {
		return nil
	}
	channels := s.Find(sqls.NewCnd().
		Eq("channel_type", enums.ChannelTypeWxWorkKF).
		Eq("status", enums.StatusOk).
		Asc("id"))
	for i := range channels {
		cfg, err := s.ParseWxWorkKFChannelConfig(channels[i].ConfigJSON)
		if err != nil {
			continue
		}
		if cfg != nil && cfg.OpenKfID == openKfID {
			return &channels[i]
		}
	}
	return nil
}

func (s *channelService) GetEnabledChannel(ctx *gin.Context) *models.Channel {
	channelID := httpx.GetChannelID(ctx)
	channel := repositories.ChannelRepository.GetByChannelID(sqls.DB(), channelID)
	if channel == nil {
		return nil
	}
	if channel.Status != enums.StatusOk {
		return nil
	}
	return channel
}

func (s *channelService) buildChannelModel(id int64, req request.CreateChannelRequest) (*models.Channel, error) {
	channelType := strings.TrimSpace(req.ChannelType)
	if channelType != enums.ChannelTypeWeb && channelType != enums.ChannelTypeWechatMP && channelType != enums.ChannelTypeWxWorkKF && channelType != enums.ChannelTypeTelegram && channelType != enums.ChannelTypeZaloOA && channelType != enums.ChannelTypeDiscord {
		return nil, errorsx.InvalidParamI18n("error.e0250")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParamI18n("error.e0247")
	}
	if req.AIAgentID <= 0 {
		return nil, errorsx.InvalidParamI18n("error.e0321")
	}
	if req.AIAgentRolloutPercent == 0 {
		req.AIAgentRolloutPercent = 100
	}
	if req.AIAgentRolloutPercent < 1 || req.AIAgentRolloutPercent > 100 {
		return nil, errorsx.InvalidParam("channel ai agent rollout percent must be between 1 and 100")
	}
	if req.AIReplyTimeoutSeconds < 0 || req.AIReplyTimeoutSeconds > models.MaxAIReplyTimeoutSeconds {
		return nil, errorsx.InvalidParamI18n("error.e0349")
	}
	aiAgent := AIAgentService.Get(req.AIAgentID)
	if aiAgent == nil || aiAgent.Status != enums.StatusOk {
		return nil, errorsx.InvalidParamI18n("error.e0004")
	}
	if aiAgent.PublishedRevisionID <= 0 {
		return nil, errorsx.InvalidParam("ai agent must be published before binding channel")
	}
	status := enums.Status(req.Status)
	if req.Status == 0 {
		status = enums.StatusOk
	}
	if status != enums.StatusOk && status != enums.StatusDisabled {
		return nil, errorsx.InvalidParamI18n("error.e0249")
	}

	channelID := ""
	if id > 0 {
		current := s.Get(id)
		if current == nil || current.Status == enums.StatusDeleted {
			return nil, errorsx.InvalidParamI18n("error.e0208")
		}
		channelID = strings.TrimSpace(current.ChannelID)
	}
	configJSON := strings.TrimSpace(req.ConfigJSON)
	switch channelType {
	case enums.ChannelTypeWeb:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseWebChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParamI18n("error.e0060")
		}
		if strings.TrimSpace(cfg.UserTokenSecret) == "" {
			secret, err := generateUserTokenSecret()
			if err != nil {
				return nil, err
			}
			cfg.UserTokenSecret = secret
		}
		configBytes, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		configJSON = string(configBytes)
	case enums.ChannelTypeWechatMP:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseWechatMPChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParamI18n("error.e0201")
		}
		if strings.TrimSpace(cfg.UserTokenSecret) == "" {
			secret, err := generateUserTokenSecret()
			if err != nil {
				return nil, err
			}
			cfg.UserTokenSecret = secret
		}
		configBytes, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		configJSON = string(configBytes)
	case enums.ChannelTypeWxWorkKF:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseWxWorkKFChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParamI18n("error.e0102")
		}
		if cfg == nil || cfg.OpenKfID == "" {
			return nil, errorsx.InvalidParamI18n("error.e0103")
		}
		if cfg.AgentID == "" {
			return nil, errorsx.InvalidParamI18n("error.wxwork.agentIdRequired")
		}
		if _, err := wxwork.GetWorkCliByAgentID(cfg.AgentID); err != nil {
			return nil, err
		}
		if channel := s.GetEnabledWxWorkKFChannelByOpenKfID(cfg.OpenKfID); channel != nil && channel.ID != id {
			return nil, errorsx.InvalidParamI18n("error.e0069")
		}
	case enums.ChannelTypeTelegram:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseTelegramChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParam("invalid telegram configuration")
		}
		if cfg == nil || cfg.BotToken == "" {
			return nil, errorsx.InvalidParam("telegram botToken is required")
		}
		if cfg.WebhookSecret == "" {
			if secret, err := generateUserTokenSecret(); err == nil {
				cfg.WebhookSecret = secret
			}
		}
		configBytes, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		configJSON = string(configBytes)
	case enums.ChannelTypeZaloOA:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseZaloOAChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParam("invalid zalo oa configuration")
		}
		if cfg == nil || cfg.AccessToken == "" {
			return nil, errorsx.InvalidParam("zalo oa accessToken is required")
		}
		configBytes, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		configJSON = string(configBytes)
	case enums.ChannelTypeDiscord:
		if channelID == "" {
			channelID = strs.UUID()
		}
		if exists := s.Take("channel_id = ? AND status <> ? AND id <> ?", channelID, enums.StatusDeleted, id); exists != nil {
			return nil, errorsx.InvalidParamI18n("error.e0248")
		}
		cfg, err := s.ParseDiscordChannelConfig(configJSON)
		if err != nil {
			return nil, errorsx.InvalidParam("invalid discord configuration")
		}
		// A channel may rely on the deployment-wide bot token instead of carrying
		// its own, so the token is not required here the way Telegram's is.
		if cfg.ChannelScope != "" && cfg.ChannelScope != "all" && cfg.ChannelScope != "dm_only" {
			return nil, errorsx.InvalidParam("discord channelScope must be all or dm_only")
		}
		if cfg.WebhookSecret == "" {
			if secret, err := generateUserTokenSecret(); err == nil {
				cfg.WebhookSecret = secret
			}
		}
		configBytes, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		configJSON = string(configBytes)
	}

	return &models.Channel{
		ChannelType:           channelType,
		ChannelID:             channelID,
		AIAgentID:             req.AIAgentID,
		AIAgentRolloutPercent: req.AIAgentRolloutPercent,
		AIReplyPlaceholder:    strings.TrimSpace(req.AIReplyPlaceholder),
		AIReplyTimeoutSeconds: req.AIReplyTimeoutSeconds,
		AIReplyTimeoutNotice:  strings.TrimSpace(req.AIReplyTimeoutNotice),
		Name:                  name,
		ConfigJSON:            configJSON,
		Status:                status,
		Remark:                strings.TrimSpace(req.Remark),
	}, nil
}
