package transport

import (
	"context"
	"log"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/ptickz/expenses-monitor-golang/internal/core"
)

type Bot struct {
	logger      *slog.Logger
	api         *bot.Bot
	core        *core.Core
	adminID     int64
	groupChatID int64
}

func NewBot(l *slog.Logger, core *core.Core, token string, adminID, groupChatID int64) (*Bot, error) {
	l = l.With("component", "bot")

	t := &Bot{logger: l, core: core, adminID: adminID, groupChatID: groupChatID}

	api, err := bot.New(token, bot.WithDefaultHandler(t.defaultHandler))
	if err != nil {
		return t, err
	}

	t.api = api

	t.api.RegisterHandler(bot.HandlerTypeMessageText, "stuff", bot.MatchTypeCommand, t.stuffHandler)

	return t, nil
}

func (t *Bot) Listen(ctx context.Context) {
	t.api.Start(ctx)
}

func (t *Bot) stuffHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	t.core.DoStuff()
	t.reply(ctx, update.Message.Chat.ID, "caught stuff")
}

func (t *Bot) reply(ctx context.Context, chatID int64, text string) {
	_, err := t.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("send message: %v", err)
	}
}

func (t *Bot) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	t.reply(ctx, update.Message.Chat.ID, "unknown command")
}
