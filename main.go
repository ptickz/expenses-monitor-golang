package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/ptickz/expenses-monitor-golang/internal/config"
	"github.com/ptickz/expenses-monitor-golang/internal/core"
	"github.com/ptickz/expenses-monitor-golang/internal/logctx"
	"github.com/ptickz/expenses-monitor-golang/internal/store"
	"github.com/ptickz/expenses-monitor-golang/internal/transport/telegram"
)

func main() {
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf, err := config.Load()
	if err != nil {
		log.Fatalf("config read: %v", err)
	}

	var base slog.Handler
	opts := &slog.HandlerOptions{Level: conf.LogLevel}

	switch conf.LogFormat {
	case "text":
		base = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		base = slog.NewJSONHandler(os.Stdout, opts)
	default:
		log.Fatalf("unknown LOG_FORMAT %q, want text or json", conf.LogFormat)
	}

	logger := slog.New(logctx.NewHandler(base))
	slog.SetDefault(logger)

	if err := run(ctx, conf, logger); err != nil {
		logger.Error("run failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, conf config.Config, logger *slog.Logger) error {
	pool, err := pgxpool.New(ctx, conf.DBDSN)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}

	st := store.New(pool)

	app := core.New(logger, st)

	err = app.BootstrapAdmin(ctx, core.SourceTelegram, strconv.FormatInt(conf.AdminTelegramID, 10))
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}

	tgBot, err := telegram.NewBot(ctx, logger, app, conf.TelegramBotToken, conf.GroupChatID)
	if err != nil {
		return fmt.Errorf("bot init: %w", err)
	}

	logger.Info("bot started", slog.String("log_level", conf.LogLevel.String()))
	tgBot.Listen(ctx)
	logger.Info("bot stopped")

	return nil
}
