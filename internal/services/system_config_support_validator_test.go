package services

import (
	"encoding/json"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/openidentity"

	"github.com/golang-jwt/jwt/v5"
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

func TestSignSupportUserTokenUsesInternalUserSource(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	config.SetCurrent(&config.Config{
		CustomerSession: config.CustomerSessionConfig{Secret: "customer-session-secret"},
	})
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
	user := &models.User{ID: 88, Username: "support-user", Nickname: "支持中心用户", Status: enums.StatusOk}

	result, err := CustomerSessionService.SignSupportUserToken(channel, user)
	if err != nil {
		t.Fatalf("SignSupportUserToken() error = %v", err)
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(result.UserToken, claims, func(token *jwt.Token) (any, error) {
		return []byte(config.Current().CustomerSession.Secret), nil
	}, jwt.WithExpirationRequired())
	if err != nil || token == nil || !token.Valid {
		t.Fatalf("parse signed token: token=%#v err=%v", token, err)
	}
	if claims["typ"] != openidentity.SupportUserTokenType {
		t.Fatalf("typ claim = %#v", claims["typ"])
	}
	if claims["userId"] != "88" {
		t.Fatalf("userId claim = %#v", claims["userId"])
	}
	if claims["name"] != "支持中心用户" {
		t.Fatalf("name claim = %#v", claims["name"])
	}
	if expiresAt, err := time.Parse(time.DateTime, result.ExpiresAt); err != nil || time.Until(expiresAt) <= 0 {
		t.Fatalf("invalid expiresAt %q: %v", result.ExpiresAt, err)
	}
}
