package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

type supportNavigationMenuValidator struct{}

type supportAICustomerServiceConfigValidator struct{}

func (supportNavigationMenuValidator) Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error) {
	var input []request.SupportNavigationMenuItemRequest
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, []response.ConfigFieldError{configFieldError("", "invalid_json", "error.supportConfig.navigationInvalidJSON")}, nil
	}
	items, fieldErrors := normalizeSupportNavigationMenu(input)
	if len(fieldErrors) > 0 {
		return nil, fieldErrors, nil
	}
	normalized, err := json.Marshal(items)
	if err != nil {
		return nil, nil, err
	}
	return normalized, nil, nil
}

func (supportAICustomerServiceConfigValidator) Validate(raw json.RawMessage) (json.RawMessage, []response.ConfigFieldError, error) {
	var input request.SupportAICustomerServiceConfigRequest
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, []response.ConfigFieldError{configFieldError("aiCustomerService", "invalid_json", "error.supportConfig.aiCustomerServiceInvalidJSON")}, nil
	}
	cfg, fieldErrors := normalizeSupportAICustomerServiceConfig(input)
	if len(fieldErrors) > 0 {
		return nil, fieldErrors, nil
	}
	normalized, err := json.Marshal(cfg)
	if err != nil {
		return nil, nil, err
	}
	return normalized, nil, nil
}

func normalizeSupportAICustomerServiceConfig(input request.SupportAICustomerServiceConfigRequest) (response.SupportAICustomerServiceConfigResponse, []response.ConfigFieldError) {
	cfg := response.SupportAICustomerServiceConfigResponse{
		Enabled:   input.Enabled,
		ChannelID: strings.TrimSpace(input.ChannelID),
	}
	if !cfg.Enabled {
		return cfg, nil
	}
	if cfg.ChannelID == "" {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "required", "error.supportConfig.aiCustomerServiceChannelRequired")}
	}
	channel := repositories.ChannelRepository.GetByChannelID(sqls.DB(), cfg.ChannelID)
	if channel == nil || channel.Status == enums.StatusDeleted {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "not_found", "error.supportConfig.aiCustomerServiceChannelNotFound")}
	}
	if channel.ChannelType != enums.ChannelTypeWeb {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "type_invalid", "error.supportConfig.aiCustomerServiceChannelTypeInvalid")}
	}
	if channel.Status != enums.StatusOk {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "disabled", "error.supportConfig.aiCustomerServiceChannelDisabled")}
	}
	aiAgent := repositories.AIAgentRepository.Get(sqls.DB(), channel.AIAgentID)
	if aiAgent == nil || aiAgent.Status != enums.StatusOk {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "agent_disabled", "error.supportConfig.aiCustomerServiceAgentDisabled")}
	}
	if aiAgent.PublishedRevisionID <= 0 {
		return cfg, []response.ConfigFieldError{configFieldError("aiCustomerService.channelId", "agent_unpublished", "error.supportConfig.aiCustomerServiceAgentUnpublished")}
	}
	return cfg, nil
}

func normalizeSupportNavigationMenu(input []request.SupportNavigationMenuItemRequest) ([]response.SupportNavigationMenuItemResponse, []response.ConfigFieldError) {
	if len(input) == 0 {
		return nil, []response.ConfigFieldError{configFieldError("navigationMenu", "required", "error.supportConfig.navigationRequired")}
	}
	if len(input) > 20 {
		return nil, []response.ConfigFieldError{configFieldError("navigationMenu", "too_many", "error.supportConfig.navigationTooMany")}
	}
	seenIDs := make(map[string]int)
	items, visibleCount, fieldErrors := normalizeSupportNavigationItems(input, "navigationMenu", 1, seenIDs)
	if len(fieldErrors) > 0 {
		return nil, fieldErrors
	}
	if visibleCount == 0 {
		return nil, []response.ConfigFieldError{configFieldError("navigationMenu", "visible_required", "error.supportConfig.navigationVisibleRequired")}
	}
	return items, nil
}

func normalizeSupportNavigationItems(input []request.SupportNavigationMenuItemRequest, path string, depth int, seenIDs map[string]int) ([]response.SupportNavigationMenuItemResponse, int, []response.ConfigFieldError) {
	items := make([]response.SupportNavigationMenuItemResponse, 0, len(input))
	visibleCount := 0
	for idx, raw := range input {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		title := strings.TrimSpace(raw.Title)
		if title == "" {
			return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".title", "required", "error.supportConfig.navigationTitleRequired")}
		}
		if len([]rune(title)) > 64 {
			return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".title", "too_long", "error.supportConfig.navigationTitleTooLong")}
		}
		link := strings.TrimSpace(raw.URL)
		if link == "" {
			return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".url", "required", "error.supportConfig.navigationURLRequired")}
		}
		if !isAllowedSupportNavigationURL(link) {
			return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".url", "invalid_url", "error.supportConfig.navigationURLInvalid")}
		}
		id := normalizeSupportNavigationMenuID(raw.ID)
		if id == "" {
			id = "nav-" + strings.ReplaceAll(strs.UUID(), "-", "")[:12]
		}
		if count := seenIDs[id]; count > 0 {
			id = id + "-" + strings.ReplaceAll(strs.UUID(), "-", "")[:6]
		}
		seenIDs[id]++
		visible := true
		if raw.Visible != nil {
			visible = *raw.Visible
		}
		children := []response.SupportNavigationMenuItemResponse(nil)
		if len(raw.Children) > 0 {
			if depth >= 2 {
				return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".children", "too_deep", "error.supportConfig.navigationTooDeep")}
			}
			if len(raw.Children) > 20 {
				return nil, 0, []response.ConfigFieldError{configFieldError(itemPath+".children", "too_many", "error.supportConfig.navigationChildrenTooMany")}
			}
			nextChildren, nextVisibleCount, fieldErrors := normalizeSupportNavigationItems(raw.Children, itemPath+".children", depth+1, seenIDs)
			if len(fieldErrors) > 0 {
				return nil, 0, fieldErrors
			}
			children = nextChildren
			if nextVisibleCount > 0 && visible {
				visibleCount += nextVisibleCount
			}
		}
		if visible {
			visibleCount++
		}
		items = append(items, response.SupportNavigationMenuItemResponse{
			ID:              id,
			Title:           title,
			URL:             link,
			OpenInNewWindow: raw.OpenInNewWindow,
			Visible:         visible,
			SortNo:          (idx + 1) * 10,
			Children:        children,
		})
	}
	return items, visibleCount, nil
}

func isAllowedSupportNavigationURL(value string) bool {
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func normalizeSupportNavigationMenuID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-_")
}

func configFieldError(path, code, key string) response.ConfigFieldError {
	return response.ConfigFieldError{
		Path:       path,
		Code:       code,
		MessageKey: key,
		Message:    i18nx.Get(key),
	}
}

func localizeConfigFieldErrors(errors []response.ConfigFieldError, locale string) []response.ConfigFieldError {
	if len(errors) == 0 {
		return nil
	}
	ret := make([]response.ConfigFieldError, 0, len(errors))
	for _, item := range errors {
		if item.MessageKey != "" {
			item.Message = i18nx.Getf(locale, item.MessageKey)
		}
		ret = append(ret, item)
	}
	return ret
}

func defaultSupportNavigationMenu() []response.SupportNavigationMenuItemResponse {
	return []response.SupportNavigationMenuItemResponse{
		{ID: "home", Title: i18nx.Get("systemConfig.support.navigationMenu.default.home"), URL: "/support", SortNo: 10, Visible: true},
		{ID: "docs", Title: i18nx.Get("systemConfig.support.navigationMenu.default.docs"), URL: "/support/docs", SortNo: 20, Visible: true},
		{ID: "community", Title: i18nx.Get("systemConfig.support.navigationMenu.default.community"), URL: "/support/community/posts", SortNo: 30, Visible: true},
	}
}

func defaultSupportAICustomerServiceConfig() response.SupportAICustomerServiceConfigResponse {
	return response.SupportAICustomerServiceConfigResponse{}
}
