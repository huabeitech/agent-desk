package cronx

import (
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/services"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	})

	addDailyBusinessReportJob(c)

	c.Start()
}

func addFunc(c *cron.Cron, sepc string, cmd func()) {
	if _, err := c.AddFunc(sepc, cmd); err != nil {
		slog.Error("add cron func error", slog.Any("err", err))
	}
}

func addDailyBusinessReportJob(c *cron.Cron) {
	cfg := config.Current()
	dailyCfg := cfg.Notify.DailyReport
	if !dailyCfg.Enabled {
		return
	}
	spec := strings.TrimSpace(dailyCfg.Cron)
	if spec == "" {
		spec = "0 9 * * *"
	}
	addFunc(c, spec, func() {
		reportDate := time.Now().AddDate(0, 0, dailyCfg.DateOffsetDays).Format(time.DateOnly)
		resp, err := services.DashboardService.SendScheduledDailyBusinessReportWebhook(reportDate, cfg.LanguageOrDefault())
		if err != nil {
			slog.Error("send scheduled daily business report failed", "error", err, "reportDate", reportDate)
			return
		}
		if resp.Sent {
			slog.Info("scheduled daily business report sent", "reportDate", reportDate)
		} else {
			slog.Warn("scheduled daily business report skipped", "reportDate", reportDate, "message", resp.Message)
		}
	})
}
