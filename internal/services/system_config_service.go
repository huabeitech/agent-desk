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
)

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

func (s *systemConfigService) SaveSupportConfig(payload map[string]json.RawMessage, operator *dto.AuthPrincipal) (response.DashboardSupportConfigResponse, error) {
	if err := s.SaveGroupConfig(systemConfigGroupSupportCenter, payload, operator); err != nil {
		return response.DashboardSupportConfigResponse{}, err
	}
	return s.GetDashboardSupportConfig(), nil
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
