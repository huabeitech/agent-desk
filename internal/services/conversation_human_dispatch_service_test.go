package services_test

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/services"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestConversationHumanDispatchAIHandoffOffHoursKeepsAIServingAndSendsNotice(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "1")
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusAIServing)

	result, err := services.ConversationHumanDispatchService.HandoffByAI(conversation.ID, aiAgent, "用户要求转人工")
	if err != nil {
		t.Fatalf("HandoffByAI() error = %v", err)
	}
	if result == nil || result.Decision != services.HandoffDecisionOffHours {
		t.Fatalf("expected off_hours decision, got %+v", result)
	}

	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusAIServing {
		t.Fatalf("expected conversation to stay AI serving, got status=%d", current.Status)
	}
	if current.HandoffAt != nil {
		t.Fatalf("expected handoffAt to stay nil, got %v", current.HandoffAt)
	}

	var lead models.SalesLead
	if err := db.Where("conversation_id = ?", conversation.ID).First(&lead).Error; err != nil {
		t.Fatalf("expected off-hours handoff lead: %v", err)
	}
	if lead.OwnerUserID != 0 || lead.NextFollowUpAt == nil || lead.SourceChannel != "off_hours_handoff" {
		t.Fatalf("unexpected off-hours lead: %+v", lead)
	}
	if lead.Status != enums.SalesLeadStatusNew || lead.IntentLevel != enums.SalesLeadIntentMedium {
		t.Fatalf("expected new medium-intent follow-up lead, got %+v", lead)
	}

	message := services.MessageService.FindOne(sqls.NewCnd().Eq("conversation_id", conversation.ID).Desc("id"))
	if message == nil {
		t.Fatalf("expected off-hours notice message")
	}
	if message.SenderType != enums.IMSenderTypeAI ||
		(!strings.Contains(message.Content, "Human support is currently outside service hours") &&
			!strings.Contains(message.Content, "当前不在人工客服服务时间内")) {
		t.Fatalf("unexpected off-hours message: %+v", message)
	}
}

func TestConversationHumanDispatchAIHandoffAssignsAvailableAgent(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "1")
	createHumanDispatchTeam(t, db, 1, "售后支持组")
	createHumanDispatchActiveSchedule(t, db, 1)
	createHumanDispatchAgentProfile(t, db, 101, 1, enums.ServiceStatusIdle, 3, true, enums.StatusOk)
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusAIServing)

	result, err := services.ConversationHumanDispatchService.HandoffByAI(conversation.ID, aiAgent, "用户要求转人工")
	if err != nil {
		t.Fatalf("HandoffByAI() error = %v", err)
	}
	if result == nil || result.Decision != services.HandoffDecisionAssigned {
		t.Fatalf("expected assigned decision, got %+v", result)
	}

	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusActive {
		t.Fatalf("expected active conversation, got status=%d", current.Status)
	}
	if current.CurrentAssigneeID != 101 || current.CurrentTeamID != 1 {
		t.Fatalf("unexpected assignment: assignee=%d team=%d", current.CurrentAssigneeID, current.CurrentTeamID)
	}
	if current.HandoffAt == nil || current.HandoffReason != "用户要求转人工" {
		t.Fatalf("expected handoff metadata, got at=%v reason=%q", current.HandoffAt, current.HandoffReason)
	}
}

func TestConversationHumanDispatchAIHandoffFallsBackToFirstScheduledTeam(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "3,1,2")
	createHumanDispatchTeam(t, db, 1, "售后支持组")
	createHumanDispatchTeam(t, db, 2, "VIP支持组")
	createHumanDispatchTeam(t, db, 3, "非值班组")
	createHumanDispatchActiveSchedule(t, db, 1)
	createHumanDispatchActiveSchedule(t, db, 2)
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusAIServing)

	result, err := services.ConversationHumanDispatchService.HandoffByAI(conversation.ID, aiAgent, "用户要求转人工")
	if err != nil {
		t.Fatalf("HandoffByAI() error = %v", err)
	}
	if result == nil || result.Decision != services.HandoffDecisionTeamPool {
		t.Fatalf("expected team_pool decision, got %+v", result)
	}

	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusPending {
		t.Fatalf("expected pending conversation, got status=%d", current.Status)
	}
	if current.CurrentTeamID != 1 || current.CurrentAssigneeID != 0 {
		t.Fatalf("expected fallback team 1 with no assignee, got team=%d assignee=%d", current.CurrentTeamID, current.CurrentAssigneeID)
	}
}

func TestConversationHumanDispatchHumanOnlyCreateOffHoursUsesGlobalPendingPool(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeHumanOnly, "1")

	conversation, err := services.ConversationService.Create(openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceGuest,
		ExternalID:     "guest-human-only-off-hours",
		ExternalName:   "非服务时间访客",
	}, 1, aiAgent.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if conversation.Status != enums.IMConversationStatusPending {
		t.Fatalf("expected pending conversation, got status=%d", conversation.Status)
	}
	if conversation.CurrentTeamID != 0 || conversation.CurrentAssigneeID != 0 {
		t.Fatalf("expected global pending pool, got team=%d assignee=%d", conversation.CurrentTeamID, conversation.CurrentAssigneeID)
	}

	message := services.MessageService.FindOne(sqls.NewCnd().Eq("conversation_id", conversation.ID).Desc("id"))
	if message == nil || message.Content != services.HandoffWaitingMessage {
		t.Fatalf("expected waiting message, got %+v", message)
	}
}

func TestConversationHumanDispatchHumanOnlyCreateAssignsAvailableAgent(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeHumanOnly, "1")
	createHumanDispatchTeam(t, db, 1, "售后支持组")
	createHumanDispatchActiveSchedule(t, db, 1)
	createHumanDispatchAgentProfile(t, db, 101, 1, enums.ServiceStatusIdle, 3, true, enums.StatusOk)

	conversation, err := services.ConversationService.Create(openidentity.ExternalUser{
		ExternalSource: enums.ExternalSourceGuest,
		ExternalID:     "guest-human-only-assigned",
		ExternalName:   "服务时间访客",
	}, 1, aiAgent.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if conversation.Status != enums.IMConversationStatusActive {
		t.Fatalf("expected active conversation, got status=%d", conversation.Status)
	}
	if conversation.CurrentAssigneeID != 101 || conversation.CurrentTeamID != 1 {
		t.Fatalf("unexpected assignment: assignee=%d team=%d", conversation.CurrentAssigneeID, conversation.CurrentTeamID)
	}
}

func TestConversationAutoAssignManualDispatchOffHoursReturnsBusinessMessage(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "1")
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusPending)

	err := services.ConversationService.AutoAssignConversation(conversation.ID, testHumanDispatchOperator())
	if err == nil {
		t.Fatalf("expected off-hours manual dispatch to fail")
	}
	if !strings.Contains(err.Error(), "当前暂不在人工客服服务时间内") {
		t.Fatalf("expected off-hours error, got %v", err)
	}
}

func TestConversationAutoAssignManualDispatchFallsBackToTeamPool(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "1")
	createHumanDispatchTeam(t, db, 1, "售后支持组")
	createHumanDispatchActiveSchedule(t, db, 1)
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusPending)

	err := services.ConversationService.AutoAssignConversation(conversation.ID, testHumanDispatchOperator())
	if err != nil {
		t.Fatalf("AutoAssignConversation() error = %v", err)
	}
	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusPending || current.CurrentTeamID != 1 || current.CurrentAssigneeID != 0 {
		t.Fatalf("expected team-pool pending conversation, got %+v", current)
	}
}

func TestConversationResumeAIConversationReturnsActiveConversationToAI(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeAIFirst, "1")
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusActive)
	if err := db.Model(&models.Conversation{}).Where("id = ?", conversation.ID).Updates(map[string]any{
		"current_assignee_id": int64(101),
		"current_team_id":     int64(1),
	}).Error; err != nil {
		t.Fatalf("update conversation assignment: %v", err)
	}
	if err := db.Create(&models.ConversationAssignment{
		ConversationID: conversation.ID,
		ToUserID:       101,
		AssignType:     string(enums.IMAssignmentTypeAssign),
		Reason:         "人工接管",
		Status:         enums.IMAssignmentStatusActive,
		CreatedAt:      time.Now(),
		OperatorID:     101,
	}).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	err := services.ConversationService.ResumeAIConversation(conversation.ID, "处理完成", &dto.AuthPrincipal{UserID: 101, Username: "agent"})
	if err != nil {
		t.Fatalf("ResumeAIConversation() error = %v", err)
	}
	current := services.ConversationService.Get(conversation.ID)
	if current.Status != enums.IMConversationStatusAIServing || current.CurrentAssigneeID != 0 || current.CurrentTeamID != 0 {
		t.Fatalf("expected ai-serving conversation without assignee, got %+v", current)
	}
	var assignment models.ConversationAssignment
	if err := db.Where("conversation_id = ?", conversation.ID).First(&assignment).Error; err != nil {
		t.Fatalf("load assignment: %v", err)
	}
	if assignment.Status != enums.IMAssignmentStatusInactive || assignment.FinishedAt == nil {
		t.Fatalf("expected finished assignment, got %+v", assignment)
	}
	var event models.ConversationEventLog
	if err := db.Where("conversation_id = ? AND content = ?", conversation.ID, "恢复 AI 接待").First(&event).Error; err != nil {
		t.Fatalf("expected resume ai event: %v", err)
	}
}

func TestConversationResumeAIConversationRejectsHumanOnlyConversation(t *testing.T) {
	db := setupConversationHumanDispatchTestDB(t)
	aiAgent := createHumanDispatchAIAgent(t, db, enums.IMConversationServiceModeHumanOnly, "1")
	conversation := createHumanDispatchConversation(t, db, aiAgent.ID, enums.IMConversationStatusActive)
	if err := db.Model(&models.Conversation{}).Where("id = ?", conversation.ID).Updates(map[string]any{
		"service_mode":        enums.IMConversationServiceModeHumanOnly,
		"current_assignee_id": int64(101),
	}).Error; err != nil {
		t.Fatalf("update conversation service mode: %v", err)
	}

	err := services.ConversationService.ResumeAIConversation(conversation.ID, "处理完成", &dto.AuthPrincipal{UserID: 101, Username: "agent"})
	if err == nil {
		t.Fatalf("expected human-only conversation to reject resume ai")
	}
}

func setupConversationHumanDispatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.CustomerIdentity{},
		&models.AIAgent{},
		&models.AgentTeam{},
		&models.AgentTeamSchedule{},
		&models.AgentProfile{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.ConversationAssignment{},
		&models.ConversationEventLog{},
		&models.ConversationReadState{},
		&models.Message{},
		&models.SalesLead{},
		&models.ChannelMessageOutbox{},
	); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createHumanDispatchAIAgent(t *testing.T, db *gorm.DB, mode enums.IMConversationServiceMode, teamIDs string) models.AIAgent {
	t.Helper()
	item := models.AIAgent{
		Name:        "测试AI",
		ServiceMode: mode,
		TeamIDs:     teamIDs,
		Status:      enums.StatusOk,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create ai agent error = %v", err)
	}
	return item
}

func createHumanDispatchTeam(t *testing.T, db *gorm.DB, id int64, name string) {
	t.Helper()
	if err := db.Create(&models.AgentTeam{ID: id, Name: name, Status: enums.StatusOk}).Error; err != nil {
		t.Fatalf("create team error = %v", err)
	}
}

func createHumanDispatchActiveSchedule(t *testing.T, db *gorm.DB, teamID int64) {
	t.Helper()
	now := time.Now()
	if err := db.Create(&models.AgentTeamSchedule{
		TeamID:  teamID,
		StartAt: now.Add(-time.Hour),
		EndAt:   now.Add(time.Hour),
		Status:  enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create schedule error = %v", err)
	}
}

func createHumanDispatchAgentProfile(t *testing.T, db *gorm.DB, userID, teamID int64, serviceStatus enums.ServiceStatus, maxConcurrent int, autoAssign bool, status enums.Status) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "agent",
		Nickname: "客服",
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create user error = %v", err)
	}
	if err := db.Create(&models.AgentProfile{
		UserID:             userID,
		TeamID:             teamID,
		AgentCode:          "A001",
		DisplayName:        "客服",
		ServiceStatus:      serviceStatus,
		MaxConcurrentCount: maxConcurrent,
		AutoAssignEnabled:  autoAssign,
		Status:             status,
	}).Error; err != nil {
		t.Fatalf("create profile error = %v", err)
	}
}

func createHumanDispatchConversation(t *testing.T, db *gorm.DB, aiAgentID int64, status enums.IMConversationStatus) models.Conversation {
	t.Helper()
	now := time.Now()
	item := models.Conversation{
		AIAgentID:     aiAgentID,
		ChannelID:     1,
		CustomerID:    1,
		CustomerName:  "测试访客",
		Status:        status,
		ServiceMode:   enums.IMConversationServiceModeAIFirst,
		LastMessageAt: now,
		LastActiveAt:  now,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create conversation error = %v", err)
	}
	return item
}

func testHumanDispatchOperator() *dto.AuthPrincipal {
	return &dto.AuthPrincipal{UserID: 9, Username: "dispatcher", Nickname: "调度员"}
}
