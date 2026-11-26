package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/ptickz/expenses-monitor-golang/internal/core"
)

type Bot struct {
	botName     string
	logger      *slog.Logger
	api         *bot.Bot
	core        *core.Core
	groupChatID int64
	drafts      *drafts
}

func NewBot(ctx context.Context, l *slog.Logger, core *core.Core, token string, groupChatID int64) (*Bot, error) {
	l = l.With("component", "bot")

	t := &Bot{logger: l, core: core, groupChatID: groupChatID, drafts: newDrafts()}

	opts := []bot.Option{
		bot.WithDefaultHandler(t.dispatch),
		bot.WithMiddlewares(t.logUpdates),
		bot.WithErrorsHandler(func(err error) { t.logger.Warn("telegram error", slog.Any("err", err)) }),
	}

	api, err := bot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create telegram client: %w", err)
	}

	t.api = api

	n, err := t.api.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("get bot info: %w", err)
	}

	t.botName = n.Username

	return t, nil
}

func (t *Bot) Listen(ctx context.Context) {
	t.api.Start(ctx)
}

func (t *Bot) reply(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup) int {
	m, err := t.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	})
	if err != nil {
		t.logger.WarnContext(ctx, "send message", slog.Any("err", err))
		return 0
	}

	return m.ID
}

func (t *Bot) answerCallback(ctx context.Context, callbackID, text string) {
	_, err := t.api.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
	if err != nil {
		t.logger.WarnContext(ctx, "answer callback", slog.Any("err", err))
	}
}

func (t *Bot) removeKeyboard(ctx context.Context, chatID int64, messageID int) {
	_, err := t.api.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	if err != nil {
		t.logger.WarnContext(ctx, "remove keyboard", slog.Any("err", err))
	}
}
