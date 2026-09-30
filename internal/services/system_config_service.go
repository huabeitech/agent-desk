package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/logx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

var SystemConfigService = newSystemConfigService()

func newSystemConfigService() *systemConfigService {
	return &systemConfigService{}
}

type systemConfigService struct {
}

const (
	systemConfigGroupSupportCenter          = "support"
	systemConfigKeySupportNavMenu           = "navigationMenu"
	systemConfigKeySupportAICustomerService = "aiCustomerService"

	systemConfigGroupSystem             = "system"
	systemConfigKeyLogLevel             = "logLevel"
	systemConfigDefaultLevel            = "warn"
	systemConfigKeyConvIdleTimeout      = "conversationIdleTimeout"
	systemConfigDefaultConvIdleTimeout  = 30
	systemConfigMinConvIdleTimeout      = 5
	systemConfigMaxConvIdleTimeout      = 1440
	systemConfigKeyConvIdleReminder     = "conversationIdleReminderMessage"
	systemConfigDefaultConvIdleReminder = "您已长时间没有响应，会话将在3分钟后结束。如有其它疑问，请及时响应哦"
)

// allowedLogLevels 为允许配置的日志最低级别。
var allowedLogLevels = []string{"debug", "info", "warn", "error"}

// conversationIdleReminderValidator 校验超时提醒文案（非空字符串）。
type conversationIdleReminderValidator struct{}

func (conversationIdleReminderValidator) Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, []response.ConfigFieldError{configFieldError("conversationIdleReminderMessage", "invalid_json", "error.systemConfig.conversationIdleReminderInvalid")}, nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, []response.ConfigFieldError{configFieldError("conversationIdleReminderMessage", "empty", "error.systemConfig.conversationIdleReminderInvalid")}, nil
	}
	normalized, err := json.Marshal(value)
	return normalized, nil, err
}

// conversationIdleTimeoutValidator 校验会话自动关闭超时（分钟）。
type conversationIdleTimeoutValidator struct{}

func (conversationIdleTimeoutValidator) Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error) {
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, []response.ConfigFieldError{configFieldError("conversationIdleTimeout", "invalid_json", "error.systemConfig.conversationIdleTimeoutInvalid")}, nil
	}
	if value < systemConfigMinConvIdleTimeout || value > systemConfigMaxConvIdleTimeout {
		return nil, []response.ConfigFieldError{configFieldError("conversationIdleTimeout", "out_of_range", "error.systemConfig.conversationIdleTimeoutInvalid")}, nil
	}
	normalized, err := json.Marshal(value)
	return normalized, nil, err
}

// logLevelValidator 校验日志级别取值并归一为小写。
type logLevelValidator struct{}

func (logLevelValidator) Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, []response.ConfigFieldError{configFieldError("logLevel", "invalid_json", "error.systemConfig.logLevelInvalid")}, nil
	}
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowedLogLevels {
		if value == candidate {
			normalized, err := json.Marshal(value)
			return normalized, nil, err
		}
	}
	return nil, []response.ConfigFieldError{configFieldError("logLevel", "invalid", "error.systemConfig.logLevelInvalid")}, nil
}

type configValidator interface {
	Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error)
}

type systemConfigDefinition struct {
	GroupCode      string
	Key            string
	TitleKey       string
	DescriptionKey string
	DefaultValue   any
	Validator      configValidator
}

type SystemConfigValidationError struct {
	errors []response.ConfigFieldError
}

func (e *SystemConfigValidationError) Error() string {
	return e.Message(i18nx.DefaultLocale)
}

func (e *SystemConfigValidationError) Message(locale string) string {
	if len(e.errors) == 0 {
		return i18nx.Getf(locale, "error.supportConfig.validationFailed")
	}
	fieldErrorMessage := e.errors[0].Message
	if e.errors[0].MessageKey != "" {
		fieldErrorMessage = i18nx.Getf(locale, e.errors[0].MessageKey)
	}
	return fmt.Sprintf("%s: %s", i18nx.Getf(locale, "error.supportConfig.validationFailed"), fieldErrorMessage)
}

func (e *SystemConfigValidationError) FieldErrors() []response.ConfigFieldError {
	if e == nil {
		return nil
	}
	return e.errors
}

func (e *SystemConfigValidationError) FieldErrorsLocale(locale string) []response.ConfigFieldError {
	if e == nil {
		return nil
	}
	return localizeConfigFieldErrors(e.errors, locale)
}

var systemConfigDefinitions = map[string]map[string]systemConfigDefinition{
	systemConfigGroupSupportCenter: {
		systemConfigKeySupportNavMenu: {
			GroupCode:      systemConfigGroupSupportCenter,
			Key:            systemConfigKeySupportNavMenu,
			TitleKey:       "systemConfig.support.navigationMenu.title",
			DescriptionKey: "systemConfig.support.navigationMenu.description",
			DefaultValue:   defaultSupportNavigationMenu(),
			Validator:      supportNavigationMenuValidator{},
		},
		systemConfigKeySupportAICustomerService: {
			GroupCode:      systemConfigGroupSupportCenter,
			Key:            systemConfigKeySupportAICustomerService,
			TitleKey:       "systemConfig.support.aiCustomerService.title",
			DescriptionKey: "systemConfig.support.aiCustomerService.description",
			DefaultValue:   defaultSupportAICustomerServiceConfig(),
			Validator:      supportAICustomerServiceConfigValidator{},
		},
	},
	systemConfigGroupSystem: {
		systemConfigKeyLogLevel: {
			GroupCode:      systemConfigGroupSystem,
			Key:            systemConfigKeyLogLevel,
			TitleKey:       "systemConfig.system.logLevel.title",
			DescriptionKey: "systemConfig.system.logLevel.description",
			DefaultValue:   systemConfigDefaultLevel,
			Validator:      logLevelValidator{},
		},
		systemConfigKeyConvIdleTimeout: {
			GroupCode:      systemConfigGroupSystem,
			Key:            systemConfigKeyConvIdleTimeout,
			TitleKey:       "systemConfig.system.conversationIdleTimeout.title",
			DescriptionKey: "systemConfig.system.conversationIdleTimeout.description",
			DefaultValue:   systemConfigDefaultConvIdleTimeout,
			Validator:      conversationIdleTimeoutValidator{},
		},
		systemConfigKeyConvIdleReminder: {
			GroupCode:      systemConfigGroupSystem,
			Key:            systemConfigKeyConvIdleReminder,
			TitleKey:       "systemConfig.system.conversationIdleReminderMessage.title",
			DescriptionKey: "systemConfig.system.conversationIdleReminderMessage.description",
			DefaultValue:   systemConfigDefaultConvIdleReminder,
			Validator:      conversationIdleReminderValidator{},
		},
	},
}

func (s *systemConfigService) Get(id int64) *models.SystemConfig {
	return repositories.SystemConfigRepository.Get(sqls.DB(), id)
}

func (s *systemConfigService) Find(cnd *sqls.Cnd) []models.SystemConfig {
	return repositories.SystemConfigRepository.Find(sqls.DB(), cnd)
}

func (s *systemConfigService) FindByGroupCode(groupCode string) []models.SystemConfig {
	return repositories.SystemConfigRepository.FindByGroupCode(sqls.DB(), groupCode)
}

func (s *systemConfigService) GetByGroupAndKey(groupCode, key string) *models.SystemConfig {
	return repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), groupCode, key)
}

func (s *systemConfigService) FindOne(cnd *sqls.Cnd) *models.SystemConfig {
	return repositories.SystemConfigRepository.FindOne(sqls.DB(), cnd)
}

func (s *systemConfigService) FindPageByCnd(cnd *sqls.Cnd) (list []models.SystemConfig, paging *sqls.Paging) {
	return repositories.SystemConfigRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *systemConfigService) GetPublicSupportConfig() response.PublicSupportConfigResponse {
	return response.PublicSupportConfigResponse{
		NavigationMenu:    s.enabledSupportNavigationMenu(),
		AICustomerService: s.publicSupportAICustomerServiceConfig(),
	}
}

func (s *systemConfigService) GetDashboardSupportConfig() response.DashboardSupportConfigResponse {
	return response.DashboardSupportConfigResponse{
		NavigationMenu:    s.supportNavigationMenu(),
		AICustomerService: s.supportAICustomerServiceConfig(),
	}
}

func (s *systemConfigService) GetPublicSupportAICustomerServiceChannel() *models.Channel {
	cfg := s.publicSupportAICustomerServiceConfig()
	if !cfg.Enabled || strings.TrimSpace(cfg.ChannelID) == "" {
		return nil
	}
	return repositories.ChannelRepository.GetByChannelID(sqls.DB(), cfg.ChannelID)
}

// GetDashboardSystemConfig 返回运营侧系统配置。
func (s *systemConfigService) GetDashboardSystemConfig() response.SystemConfigResponse {
	return response.SystemConfigResponse{
		LogLevel:                        s.LogLevel(),
		ConversationIdleTimeout:         s.ConversationIdleTimeout(),
		ConversationIdleReminderMessage: s.ConversationIdleReminderMessage(),
	}
}

// LogLevel 返回已配置的日志最低级别，未配置或非法时返回默认值。
func (s *systemConfigService) LogLevel() string {
	item := repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), systemConfigGroupSystem, systemConfigKeyLogLevel)
	if item == nil {
		return systemConfigDefaultLevel
	}
	// ConfigValue 存的是 JSON 编码字符串（如 `"info"`），先解码为原始值。
	var value string
	if err := json.Unmarshal([]byte(item.ConfigValue), &value); err != nil {
		value = item.ConfigValue
	}
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowedLogLevels {
		if value == candidate {
			return value
		}
	}
	return systemConfigDefaultLevel
}

// ConversationIdleTimeout 返回会话自动关闭超时（分钟），未配置或非法时返回默认值。
func (s *systemConfigService) ConversationIdleTimeout() int {
	item := repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), systemConfigGroupSystem, systemConfigKeyConvIdleTimeout)
	if item == nil {
		return systemConfigDefaultConvIdleTimeout
	}
	var value int
	if err := json.Unmarshal([]byte(item.ConfigValue), &value); err != nil {
		return systemConfigDefaultConvIdleTimeout
	}
	if value < systemConfigMinConvIdleTimeout || value > systemConfigMaxConvIdleTimeout {
		return systemConfigDefaultConvIdleTimeout
	}
	return value
}

// ConversationIdleReminderMessage 返回超时提醒文案，未配置时返回默认值。
func (s *systemConfigService) ConversationIdleReminderMessage() string {
	item := repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), systemConfigGroupSystem, systemConfigKeyConvIdleReminder)
	if item == nil {
		return systemConfigDefaultConvIdleReminder
	}
	var value string
	if err := json.Unmarshal([]byte(item.ConfigValue), &value); err != nil {
		return systemConfigDefaultConvIdleReminder
	}
	if strings.TrimSpace(value) == "" {
		return systemConfigDefaultConvIdleReminder
	}
	return value
}

// SaveSupportConfig 保存支持中心配置。
func (s *systemConfigService) SaveSupportConfig(payload map[string]json.RawMessage, operator *dto.AuthPrincipal) (response.DashboardSupportConfigResponse, error) {
	if err := s.SaveGroupConfig(systemConfigGroupSupportCenter, payload, operator); err != nil {
		return response.DashboardSupportConfigResponse{}, err
	}
	return s.GetDashboardSupportConfig(), nil
}

// SaveSystemConfig 保存系统配置并立即应用日志级别。
func (s *systemConfigService) SaveSystemConfig(payload map[string]json.RawMessage, operator *dto.AuthPrincipal) (response.SystemConfigResponse, error) {
	if err := s.SaveGroupConfig(systemConfigGroupSystem, payload, operator); err != nil {
		return response.SystemConfigResponse{}, err
	}
	logx.SetDBLevel(logx.ParseLevel(s.LogLevel()))
	return s.GetDashboardSystemConfig(), nil
}

func (s *systemConfigService) SaveGroupConfig(groupCode string, payload map[string]json.RawMessage, operator *dto.AuthPrincipal) error {
	definitions := systemConfigDefinitions[groupCode]
	if len(definitions) == 0 {
		return errorsx.InvalidParamI18n("error.supportConfig.groupUnsupported")
	}
	if len(payload) == 0 {
		return errorsx.InvalidParamI18n("error.supportConfig.emptyPayload")
	}

	values := make(map[string]json.RawMessage, len(payload))
	for key, raw := range payload {
		definition, ok := definitions[key]
		if !ok {
			return errorsx.InvalidParamI18n("error.supportConfig.keyUnsupported", key)
		}
		normalized := raw
		if definition.Validator != nil {
			next, fieldErrors, err := definition.Validator.Validate(raw)
			if err != nil {
				return err
			}
			if len(fieldErrors) > 0 {
				return &SystemConfigValidationError{errors: fieldErrors}
			}
			normalized = next
		}
		values[key] = normalized
	}

	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		now := time.Now()
		auditFields := utils.BuildAuditFields(operator)
		for key, raw := range values {
			definition := definitions[key]
			existing := repositories.SystemConfigRepository.FindByGroupAndKey(ctx.Tx, groupCode, key)
			if existing == nil {
				item := &models.SystemConfig{
					ConfigKey:   key,
					ConfigValue: string(raw),
					GroupCode:   groupCode,
					Title:       definition.Title(),
					Description: definition.Description(),
					Status:      enums.StatusOk,
					AuditFields: auditFields,
				}
				if err := repositories.SystemConfigRepository.Create(ctx.Tx, item); err != nil {
					return err
				}
				continue
			}
			columns := map[string]any{
				"config_value":     string(raw),
				"group_code":       groupCode,
				"title":            definition.Title(),
				"description":      definition.Description(),
				"status":           enums.StatusOk,
				"updated_at":       now,
				"update_user_id":   auditFields.UpdateUserID,
				"update_user_name": auditFields.UpdateUserName,
			}
			if err := repositories.SystemConfigRepository.Updates(ctx.Tx, existing.ID, columns); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *systemConfigService) UpdateSupportNavigationMenu(items []request.SupportNavigationMenuItemRequest, operator *dto.AuthPrincipal) ([]response.SupportNavigationMenuItemResponse, error) {
	raw, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	_, err = s.SaveSupportConfig(map[string]json.RawMessage{
		systemConfigKeySupportNavMenu: raw,
	}, operator)
	if err != nil {
		return nil, err
	}
	return s.supportNavigationMenu(), nil
}

func (s *systemConfigService) enabledSupportNavigationMenu() []response.SupportNavigationMenuItemResponse {
	items := s.supportNavigationMenu()
	enabled := make([]response.SupportNavigationMenuItemResponse, 0, len(items))
	for _, item := range items {
		if item.Visible {
			item.Children = visibleSupportNavigationChildren(item.Children)
			enabled = append(enabled, item)
		}
	}
	if len(enabled) == 0 {
		return defaultSupportNavigationMenu()
	}
	return enabled
}

func (s *systemConfigService) supportNavigationMenu() []response.SupportNavigationMenuItemResponse {
	item := repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), systemConfigGroupSupportCenter, systemConfigKeySupportNavMenu)
	if item == nil || strings.TrimSpace(item.ConfigValue) == "" {
		return defaultSupportNavigationMenu()
	}
	var list []response.SupportNavigationMenuItemResponse
	if err := json.Unmarshal([]byte(item.ConfigValue), &list); err != nil {
		return defaultSupportNavigationMenu()
	}
	if len(list) == 0 {
		return defaultSupportNavigationMenu()
	}
	return sortSupportNavigationMenu(list)
}

func (s *systemConfigService) publicSupportAICustomerServiceConfig() response.SupportAICustomerServiceConfigResponse {
	cfg := s.supportAICustomerServiceConfig()
	if !cfg.Enabled {
		return response.SupportAICustomerServiceConfigResponse{}
	}
	channel := repositories.ChannelRepository.GetByChannelID(sqls.DB(), cfg.ChannelID)
	if channel == nil || channel.Status != enums.StatusOk || channel.ChannelType != enums.ChannelTypeWeb {
		return response.SupportAICustomerServiceConfigResponse{}
	}
	aiAgent := repositories.AIAgentRepository.Get(sqls.DB(), channel.AIAgentID)
	if aiAgent == nil || aiAgent.Status != enums.StatusOk || aiAgent.PublishedRevisionID <= 0 {
		return response.SupportAICustomerServiceConfigResponse{}
	}
	return cfg
}

func (s *systemConfigService) supportAICustomerServiceConfig() response.SupportAICustomerServiceConfigResponse {
	item := repositories.SystemConfigRepository.FindByGroupAndKey(sqls.DB(), systemConfigGroupSupportCenter, systemConfigKeySupportAICustomerService)
	if item == nil || strings.TrimSpace(item.ConfigValue) == "" {
		return defaultSupportAICustomerServiceConfig()
	}
	var cfg response.SupportAICustomerServiceConfigResponse
	if err := json.Unmarshal([]byte(item.ConfigValue), &cfg); err != nil {
		return defaultSupportAICustomerServiceConfig()
	}
	cfg.ChannelID = strings.TrimSpace(cfg.ChannelID)
	return cfg
}

func sortSupportNavigationMenu(items []response.SupportNavigationMenuItemResponse) []response.SupportNavigationMenuItemResponse {
	ret := append([]response.SupportNavigationMenuItemResponse(nil), items...)
	for i := 0; i < len(ret)-1; i++ {
		for j := i + 1; j < len(ret); j++ {
			if ret[j].SortNo < ret[i].SortNo || (ret[j].SortNo == ret[i].SortNo && ret[j].ID < ret[i].ID) {
				ret[i], ret[j] = ret[j], ret[i]
			}
		}
	}
	return ret
}

func visibleSupportNavigationChildren(items []response.SupportNavigationMenuItemResponse) []response.SupportNavigationMenuItemResponse {
	if len(items) == 0 {
		return nil
	}
	visible := make([]response.SupportNavigationMenuItemResponse, 0, len(items))
	for _, item := range sortSupportNavigationMenu(items) {
		if item.Visible {
			visible = append(visible, item)
		}
	}
	return visible
}

func (d systemConfigDefinition) Title() string {
	return i18nx.Get(d.TitleKey)
}

func (d systemConfigDefinition) Description() string {
	return i18nx.Get(d.DescriptionKey)
}
