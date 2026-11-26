package telegram

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/ptickz/expenses-monitor-golang/internal/logctx"
)

func (t *Bot) logUpdates(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		in := t.parse(u)
		ctx = logctx.With(ctx, in.logAttrs()...)
		if in.text != "" || in.args != "" {
			t.logger.DebugContext(ctx, "update text", slog.String("text", in.text), slog.String("args", in.args))
		}
		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				t.logger.ErrorContext(ctx, "request panicked",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())))
			}
			t.logger.InfoContext(ctx, "request handled",
				slog.Int64("duration_ms", time.Since(start).Milliseconds()))
		}()

		next(ctx, b, u)
	}
}
