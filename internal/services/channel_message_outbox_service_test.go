package services

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"agent-desk/internal/wxwork"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupChannelMessageOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
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
	if err := db.AutoMigrate(&models.ChannelMessageOutbox{}, &models.Message{}); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqls.SetDB(db)
	return db
}

func createOutboxRecordForTest(t *testing.T, db *gorm.DB, item *models.ChannelMessageOutbox) {
	t.Helper()
	if err := repositories.ChannelMessageOutboxRepository.Create(db, item); err != nil {
		t.Fatalf("create outbox error = %v", err)
	}
}

// TestChannelMessageOutboxListPendingEligibility 校验 ListPending 只取"现在可以尝试发送"的记录：
// failed 达到最大重试次数（next_retry_at 为空）后必须停止自动重试，退避窗口内的记录不能占用批次。
func TestChannelMessageOutboxListPendingEligibility(t *testing.T) {
	db := setupChannelMessageOutboxTestDB(t)

	now := time.Now()
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)

	cases := []struct {
		name       string
		channel    string
		status     enums.ChannelMessageOutboxStatus
		retryCount int
		nextAt     *time.Time
		expected   bool
	}{
		{"pending 无重试时间", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusPending, 0, nil, true},
		{"failed 已到期", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusFailed, 1, &past, true},
		{"failed 退避窗口内", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusFailed, 1, &future, false},
		{"failed 达上限无重试时间", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusFailed, 6, nil, false},
		{"pending 未来重试时间", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusPending, 0, &future, false},
		{"sending 状态", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusSending, 0, nil, false},
		{"sent 状态", enums.ChannelTypeWxWorkKF, enums.ChannelMessageOutboxStatusSent, 0, nil, false},
		{"其他渠道", enums.ChannelTypeTelegram, enums.ChannelMessageOutboxStatusPending, 0, nil, false},
	}
	for i, tc := range cases {
		createOutboxRecordForTest(t, db, &models.ChannelMessageOutbox{
			ChannelType: tc.channel,
			MessageID:   int64(i + 1),
			Payload:     "{}",
			SendStatus:  string(tc.status),
			RetryCount:  tc.retryCount,
			NextRetryAt: tc.nextAt,
		})
	}

	items := ChannelMessageOutboxService.ListPending(enums.ChannelTypeWxWorkKF, 20)
	included := make(map[int64]bool, len(items))
	for _, item := range items {
		included[item.MessageID] = true
	}
	var mismatches []string
	for i, tc := range cases {
		messageID := int64(i + 1)
		if included[messageID] != tc.expected {
			mismatches = append(mismatches,
				"case "+tc.name+": 期望 included="+boolText(tc.expected)+", 实际 included="+boolText(included[messageID]))
		}
	}
	if len(mismatches) > 0 {
		t.Fatalf("ListPending 资格判定不符合预期:\n%s", strings.Join(mismatches, "\n"))
	}
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// TestTelegramDispatchSkipsBackoffBatch 回归守卫：整批记录都处于退避窗口时投递必须零进度终止，
// 而不是把 skip 当作成功导致外层 drain 循环空转。
func TestTelegramDispatchSkipsBackoffBatch(t *testing.T) {
	db := setupChannelMessageOutboxTestDB(t)
	future := time.Now().Add(time.Hour)
	for i := 1; i <= 3; i++ {
		nextAt := future
		createOutboxRecordForTest(t, db, &models.ChannelMessageOutbox{
			ChannelType: enums.ChannelTypeTelegram,
			MessageID:   int64(i),
			Payload:     "{}",
			SendStatus:  string(enums.ChannelMessageOutboxStatusFailed),
			RetryCount:  1,
			NextRetryAt: &nextAt,
		})
	}

	if got := TelegramOutboundService.doDispatchPendingOutbox(20); got != 0 {
		t.Fatalf("全退避批次应零进度终止, got %d", got)
	}
}

// TestWxWorkExhaustedOutboxNotAutoRetried 回归守卫：达到最大重试次数的记录不参与自动投递，
// 状态与重试次数保持不变，等待管理端人工处置。
func TestWxWorkExhaustedOutboxNotAutoRetried(t *testing.T) {
	db := setupChannelMessageOutboxTestDB(t)
	enableWxWorkForTest(t)
	createOutboxRecordForTest(t, db, &models.ChannelMessageOutbox{
		ChannelType: enums.ChannelTypeWxWorkKF,
		MessageID:   1,
		Payload:     "{}",
		SendStatus:  string(enums.ChannelMessageOutboxStatusFailed),
		RetryCount:  wxWorkKFOutboxMaxRetry,
	})

	if got := WxWorkKFOutboundService.doDispatchPendingOutbox(20); got != 0 {
		t.Fatalf("达上限记录不应被自动重试, got %d", got)
	}
	item := repositories.ChannelMessageOutboxRepository.Get(db, 1)
	if item == nil {
		t.Fatal("outbox record not found")
	}
	if item.SendStatus != string(enums.ChannelMessageOutboxStatusFailed) || item.RetryCount != wxWorkKFOutboxMaxRetry {
		t.Fatalf("达上限记录应保持 failed 且重试次数不变, got status=%s retryCount=%d", item.SendStatus, item.RetryCount)
	}
}

// enableWxWorkForTest 在测试期间启用 wxwork（doDispatchPendingOutbox 有 Enabled 门），结束后复位。
func enableWxWorkForTest(t *testing.T) {
	t.Helper()
	config.SetCurrent(&config.Config{
		WxWork: config.WxWorkConfig{
			Enabled:    true,
			CorpID:     "corp-test",
			CorpSecret: "secret-test",
		},
	})
	wxwork.Init()
	t.Cleanup(func() {
		config.SetCurrent(&config.Config{})
		wxwork.Init()
	})
}

// TestRetryWxWorkFailureRevivesExhaustedRecord 校验人工重试可以把达上限记录复活为 pending。
func TestRetryWxWorkFailureRevivesExhaustedRecord(t *testing.T) {
	db := setupChannelMessageOutboxTestDB(t)
	createOutboxRecordForTest(t, db, &models.ChannelMessageOutbox{
		ChannelType: enums.ChannelTypeWxWorkKF,
		MessageID:   1,
		Payload:     "{}",
		SendStatus:  string(enums.ChannelMessageOutboxStatusFailed),
		RetryCount:  wxWorkKFOutboxMaxRetry,
	})

	if err := ChannelMessageOutboxService.RetryWxWorkFailure(1, &dto.AuthPrincipal{}); err != nil {
		t.Fatalf("retry error = %v", err)
	}

	items := ChannelMessageOutboxService.ListPending(enums.ChannelTypeWxWorkKF, 20)
	if len(items) != 1 {
		t.Fatalf("人工重试后应重新可投递, got %d", len(items))
	}
	if items[0].SendStatus != string(enums.ChannelMessageOutboxStatusPending) {
		t.Fatalf("人工重试后状态应为 pending, got %s", items[0].SendStatus)
	}
}
