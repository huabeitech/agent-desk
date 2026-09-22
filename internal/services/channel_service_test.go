package services

import (
	"strings"
	"testing"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestChannelServiceRejectsAgentWithoutPublishedRevision(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	agent := createChannelServiceTestAgent(t, db, 0)

	_, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb,
		AIAgentID:   agent.ID,
		Name:        "官网客服",
		Status:      int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err == nil {
		t.Fatalf("expected channel creation to reject unpublished ai agent")
	}
}

func TestChannelServiceAllowsAgentWithPublishedRevision(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	agent := createChannelServiceTestAgent(t, db, 1001)

	item, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb,
		AIAgentID:   agent.ID,
		Name:        "官网客服",
		Status:      int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if item == nil || item.AIAgentID != agent.ID {
		t.Fatalf("unexpected channel: %#v", item)
	}
}

func TestChannelServiceStoresAIAgentRolloutPercent(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	agent := createChannelServiceTestAgent(t, db, 1001)
	item, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb, AIAgentID: agent.ID, AIAgentRolloutPercent: 25,
		Name: "灰度渠道", Status: int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil || item == nil || item.AIAgentRolloutPercent != 25 {
		t.Fatalf("expected persisted rollout percent, item=%#v err=%v", item, err)
	}
	if _, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb, AIAgentID: agent.ID, AIAgentRolloutPercent: 101,
		Name: "错误灰度渠道", Status: int(enums.StatusOk),
	}, channelServiceTestOperator()); err == nil {
		t.Fatal("expected invalid rollout percent to be rejected")
	}
}

func TestChannelServiceRollsBackPreviousAIAgentRolloutPercent(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	agent := createChannelServiceTestAgent(t, db, 1001)
	channel, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		ChannelType: enums.ChannelTypeWeb, AIAgentID: agent.ID, AIAgentRolloutPercent: 20,
		Name: "渠道灰度回滚", Status: int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if err := db.Model(&models.Channel{}).Where("id = ?", channel.ID).Update("previous_ai_agent_rollout_percent", 100).Error; err != nil {
		t.Fatalf("set previous rollout: %v", err)
	}
	operator := channelServiceTestOperator()
	if err := ChannelService.RollbackChannelAIAgentRollout(channel.ID, operator); err != nil {
		t.Fatalf("RollbackChannelAIAgentRollout: %v", err)
	}
	updated := ChannelService.Get(channel.ID)
	if updated == nil || updated.AIAgentRolloutPercent != 100 || updated.PreviousAIAgentRolloutPercent != 20 {
		t.Fatalf("unexpected channel rollout rollback: %#v", updated)
	}
	if err := ChannelService.RollbackChannelAIAgentRollout(channel.ID, operator); err != nil {
		t.Fatalf("second RollbackChannelAIAgentRollout: %v", err)
	}
	updated = ChannelService.Get(channel.ID)
	if updated == nil || updated.AIAgentRolloutPercent != 20 || updated.PreviousAIAgentRolloutPercent != 100 {
		t.Fatalf("unexpected channel rollout redo: %#v", updated)
	}
}

func setupChannelServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.AIAgent{}, &models.AgentRevision{}, &models.AIAgentWorkflowBinding{}, &models.Channel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createChannelServiceTestAgent(t *testing.T, db *gorm.DB, publishedRevisionID int64) models.AIAgent {
	t.Helper()
	item := models.AIAgent{
		Name:                "测试 AI",
		Status:              enums.StatusOk,
		PublishedRevisionID: publishedRevisionID,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create ai agent: %v", err)
	}
	return item
}

func channelServiceTestOperator() *dto.AuthPrincipal {
	return &dto.AuthPrincipal{UserID: 1, Username: "admin"}
}

func TestGetEnabledEmailChannelByAddress_MultiTenantSlugResolution(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	agent := createChannelServiceTestAgent(t, db, 1001)

	// Create Channel 1 for DOS (help@dos.crove.io)
	dosCfg := `{"emailAddress":"help@dos.crove.io","forwardingAddress":"help@dos.crove.io"}`
	_, err := ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "DOS Support Channel",
		ChannelType:           enums.ChannelTypeEmail,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            dosCfg,
		Status:                int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create dos channel failed: %v", err)
	}

	// Create Channel 2 for Acme (help@acme.on.crove.email)
	acmeCfg := `{"emailAddress":"support@acme.com","forwardingAddress":"help@acme.on.crove.email"}`
	_, err = ChannelService.CreateChannel(request.CreateChannelRequest{
		Name:                  "Acme Support Channel",
		ChannelType:           enums.ChannelTypeEmail,
		AIAgentID:             agent.ID,
		AIAgentRolloutPercent: 100,
		ConfigJSON:            acmeCfg,
		Status:                int(enums.StatusOk),
	}, channelServiceTestOperator())
	if err != nil {
		t.Fatalf("create acme channel failed: %v", err)
	}

	// Test 1: Exact match on EmailAddress
	c1 := ChannelService.GetEnabledEmailChannelByAddress("help@dos.crove.io")
	if c1 == nil || c1.Name != "DOS Support Channel" {
		t.Fatalf("expected DOS Support Channel, got: %+v", c1)
	}

	// Test 2: Subdomain / Slug match (e.g. any sender addressing support@dos.crove.io)
	c2 := ChannelService.GetEnabledEmailChannelByAddress("support@dos.crove.io")
	if c2 == nil || c2.Name != "DOS Support Channel" {
		t.Fatalf("expected DOS Support Channel from subdomain slug, got: %+v", c2)
	}

	// Test 3: Match on forwardingAddress (help@acme.on.crove.email)
	c3 := ChannelService.GetEnabledEmailChannelByAddress("help@acme.on.crove.email")
	if c3 == nil || c3.Name != "Acme Support Channel" {
		t.Fatalf("expected Acme Support Channel, got: %+v", c3)
	}

	// Test 4: Slug match on Acme (sales@acme.on.crove.email)
	c4 := ChannelService.GetEnabledEmailChannelByAddress("sales@acme.on.crove.email")
	if c4 == nil || c4.Name != "Acme Support Channel" {
		t.Fatalf("expected Acme Support Channel from on.crove.email slug, got: %+v", c4)
	}
}
