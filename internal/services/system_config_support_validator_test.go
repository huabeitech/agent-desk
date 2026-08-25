package services

import (
	"encoding/json"
	"testing"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
)

func TestSystemConfigValidationErrorLocalizesFieldErrors(t *testing.T) {
	_, fieldErrors, err := supportNavigationMenuValidator{}.Validate(json.RawMessage(`not-json`))
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	validationErr := &SystemConfigValidationError{errors: fieldErrors}

	if got := validationErr.Message(i18nx.LocaleEnUS); got != "Config validation failed: Navigation menu config must be a valid JSON array" {
		t.Fatalf("Message(en-US) = %q", got)
	}
	localized := validationErr.FieldErrorsLocale(i18nx.LocaleEnUS)
	if len(localized) != 1 {
		t.Fatalf("FieldErrorsLocale() length = %d, want 1", len(localized))
	}
	if localized[0].Message != "Navigation menu config must be a valid JSON array" {
		t.Fatalf("localized message = %q", localized[0].Message)
	}
	if localized[0].MessageKey != "error.supportConfig.navigationInvalidJSON" {
		t.Fatalf("message key = %q", localized[0].MessageKey)
	}
}

func TestSupportAICustomerServiceConfigValidatesWebChannel(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("migrate system config: %v", err)
	}
	agent := createChannelServiceTestAgent(t, db, 1001)
	channel, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb,
		AIAgentID:   agent.ID,
		Name:        "支持中心 AI 客服",
		Status:      int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	config, err := SystemConfigService.SaveSupportConfig(map[string]json.RawMessage{
		systemConfigKeySupportAICustomerService: json.RawMessage(`{"enabled":true,"channelId":"` + channel.ChannelID + `"}`),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("SaveSupportConfig() error = %v", err)
	}
	if !config.AICustomerService.Enabled || config.AICustomerService.ChannelID != channel.ChannelID {
		t.Fatalf("unexpected dashboard config: %#v", config.AICustomerService)
	}
	publicConfig := SystemConfigService.GetPublicSupportConfig()
	if !publicConfig.AICustomerService.Enabled || publicConfig.AICustomerService.ChannelID != channel.ChannelID {
		t.Fatalf("unexpected public config: %#v", publicConfig.AICustomerService)
	}
}

func TestPublicSupportAICustomerServiceHidesDisabledChannel(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("migrate system config: %v", err)
	}
	agent := createChannelServiceTestAgent(t, db, 1001)
	channel, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb,
		AIAgentID:   agent.ID,
		Name:        "支持中心 AI 客服",
		Status:      int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if _, err := SystemConfigService.SaveSupportConfig(map[string]json.RawMessage{
		systemConfigKeySupportAICustomerService: json.RawMessage(`{"enabled":true,"channelId":"` + channel.ChannelID + `"}`),
	}, channelServiceTestOperator()); err != nil {
		t.Fatalf("SaveSupportConfig() error = %v", err)
	}
	if err := ChannelService.UpdateStatus(channel.ID, int(enums.StatusDisabled), channelServiceTestOperator()); err != nil {
		t.Fatalf("disable channel: %v", err)
	}

	publicConfig := SystemConfigService.GetPublicSupportConfig()
	if publicConfig.AICustomerService.Enabled || publicConfig.AICustomerService.ChannelID != "" {
		t.Fatalf("disabled channel should be hidden from public config: %#v", publicConfig.AICustomerService)
	}
}

func TestSupportAICustomerServiceConfigAllowsDisabledConfigWithStaleChannel(t *testing.T) {
	_, fieldErrors, err := supportAICustomerServiceConfigValidator{}.Validate(json.RawMessage(`{"enabled":false,"channelId":"stale"}`))
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(fieldErrors) != 0 {
		t.Fatalf("disabled config should not validate stale channel: %#v", fieldErrors)
	}
}
