package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"streamagent/internal/api"
	"streamagent/internal/config"
	"streamagent/internal/scheduler"
	"streamagent/internal/tasks"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	configPath := flag.String("c", "", "config file path")
	flag.StringVar(configPath, "config", "", "config file path")
	flag.Parse()
	if strings.TrimSpace(*configPath) == "" {
		logger.Fatal("config path is required, use -c <path>")
	}

	// 收到 SIGINT/SIGTERM 时优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(strings.TrimSpace(*configPath))
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	apiClient := api.New(cfg.API, cfg.Debug, logger.Printf)

	// 根据 mode 选择 client 或 server 任务。
	var job func(context.Context) error
	switch cfg.Mode {
	case "client":
		job = tasks.NewClientTask(cfg, apiClient, logger.Printf).Run
	case "server":
		job = tasks.NewServerTask(cfg, apiClient, logger.Printf).Run
	default:
		logger.Fatalf("unsupported mode: %s", cfg.Mode)
	}

	// scheduler 负责启动后立即执行一次，然后按 cron 定时重复执行。
	service, err := scheduler.New(cfg.Scheduler, config.DefaultTimeout, job, logger.Printf)
	if err != nil {
		logger.Fatalf("create scheduler: %v", err)
	}

	if err := service.Start(ctx, true); err != nil {
		logger.Fatalf("start scheduler: %v", err)
	}

	<-ctx.Done()
	<-service.Stop()
	logger.Println("agent stopped")
}
