package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"agent-desk/internal/bootstrap"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/logx"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	if err := bootstrap.Init(*configPath); err != nil {
		slog.Error("bootstrap init failed", "error", err)
		return
	}

	cfg := config.Current()

	app, err := bootstrap.NewServer()
	if err != nil {
		slog.Error("bootstrap server failed", "error", err)
		return
	}

	// 监听退出信号，确保 db sink 中缓冲的日志在进程退出前 flush 到数据库。
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		_ = logx.Close()
		os.Exit(0)
	}()

	if err := app.Run(cfg.Server.Address()); err != nil {
		slog.Error("start server failed", "error", err)
		return
	}
}
