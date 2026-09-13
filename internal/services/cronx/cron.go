package cronx

import (
	"agent-desk/internal/services"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

func Init() {
	c := cron.New()

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
		slackCount := services.SlackOutboundService.DispatchPendingOutbox()
		if slackCount > 0 {
			slog.Info("slack outbox dispatched", "count", slackCount)
		}
	})

	c.Start()
}

func addFunc(c *cron.Cron, sepc string, cmd func()) {
	if _, err := c.AddFunc(sepc, cmd); err != nil {
		slog.Error("add cron func error", slog.Any("err", err))
	}
}
