package cronx

import (
	"agent-desk/internal/services"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

func Init() {
	// Recover：任务 panic 不拖垮整个 cron；
	// SkipIfStillRunning：上一轮未结束前跳过本轮触发，避免任务慢于间隔时 goroutine 无限堆积。
	c := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))

	addFunc(c, "0 4 ? * *", func() {
		fmt.Println("cron test")
	})

	addFunc(c, "@every 30s", func() {
		if _, err := services.ConversationDispatchService.DispatchPendingConversations(0); err != nil {
			slog.Warn("dispatch pending conversations loop failed", "error", err)
		}
	})

	addFunc(c, "@every 5s", func() {
		count := services.WxWorkKFOutboundService.DispatchPendingOutbox()
		if count > 0 {
			slog.Info("wxwork kf outbox dispatched", "count", count)
		}
		tgCount := services.TelegramOutboundService.DispatchPendingOutbox()
		if tgCount > 0 {
			slog.Info("telegram outbox dispatched", "count", tgCount)
		}
		zaloCount := services.ZaloOAOutboundService.DispatchPendingOutbox()
		if zaloCount > 0 {
			slog.Info("zalo oa outbox dispatched", "count", zaloCount)
		}
		discordCount := services.DiscordOutboundService.DispatchPendingOutbox()
		if discordCount > 0 {
			slog.Info("discord outbox dispatched", "count", discordCount)
		}
	})

	// 每天凌晨 3 点清理一个月以前的系统日志。
	addFunc(c, "0 3 * * *", func() {
		before := time.Now().AddDate(0, -1, 0)
		deleted := services.SystemLogService.DeleteOlderThan(before)
		if deleted > 0 {
			slog.Info("system log cleanup completed", "deleted", deleted, "before", before.Format(time.DateTime))
		}
	})

	c.Start()
}

func addFunc(c *cron.Cron, sepc string, cmd func()) {
	if _, err := c.AddFunc(sepc, cmd); err != nil {
		slog.Error("add cron func error", slog.Any("err", err))
	}
}
