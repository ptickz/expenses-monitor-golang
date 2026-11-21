package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/ptickz/expenses-monitor-golang/src/config"
	"github.com/ptickz/expenses-monitor-golang/src/core"
	"github.com/ptickz/expenses-monitor-golang/src/transport"
)

func main() {
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf, err := config.Load()
	if err != nil {
		log.Fatalf("config read: %v", err)
	}
	pool, err := pgxpool.New(ctx, conf.DBDSN)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	app := core.New(pool)

	tgBot, err := transport.NewBot(app, conf.TelegramBotToken)
	if err != nil {
		log.Fatalf("bot init: %v", err)
	}
	tgBot.Listen(ctx)
}
