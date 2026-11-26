package telegram

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/ptickz/expenses-monitor-golang/internal/core"
)

type route int

const (
	routeIgnored route = iota
	routeDraft
	routeStaleCallback
	routeNotConfigured
	routeSetupFirst
	routeStart
	routeAdminStart
	routeSetup
	routeFalseSetup
	routeUnknownInput
)

type sender struct {
	userID  int32
	isAdmin bool
}

func (t *Bot) dispatch(ctx context.Context, _ *bot.Bot, u *models.Update) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	in := t.parse(u)

	if !t.allowedChat(in) {
		return
	}

	s, ok, err := t.resolveSender(ctx, in)
	if err != nil {
		t.logger.ErrorContext(ctx, "resolve sender", slog.Any("err", err))
		return
	}
	if !ok {
		return
	}
	configured, err := t.core.Configured(ctx)
	if err != nil {
		t.logger.ErrorContext(ctx, "configured check", slog.Any("err", err))
		return
	}

	d := t.drafts.get(in.chatID, in.userID)

	switch pickRoute(in, s, configured, (d != nil)) {
	case routeIgnored:
		return
	case routeSetup:
		t.handleSetup(ctx, in, s)
	case routeDraft:
		if in.kind == kindCallback {
			t.answerCallback(ctx, in.callbackID, "")
		}
		done, err := d.flow.handle(ctx, t, d, in)
		if err != nil {
			t.logger.ErrorContext(ctx, "handle draft", slog.Any("err", err))
		}
		if done || (err != nil) {
			t.drafts.finish(in.chatID, in.userID)
		}
	case routeStaleCallback:
		t.answerCallback(ctx, in.callbackID, "Кнопка устарела")
	}
}

func (t *Bot) allowedChat(in incoming) bool {
	if in.kind == kindOther {
		return false
	}
	if in.chatType == models.ChatTypePrivate {
		return true
	}
	if in.chatType == models.ChatTypeGroup || in.chatType == models.ChatTypeSupergroup {
		return in.chatID == t.groupChatID
	}
	return false
}

func (t *Bot) resolveSender(ctx context.Context, in incoming) (s sender, ok bool, err error) {
	externalID := strconv.FormatInt(in.userID, 10)

	switch in.chatType {
	case models.ChatTypeGroup, models.ChatTypeSupergroup:
		id, err := t.core.RegisterUser(ctx, core.SourceTelegram, externalID, in.userName)
		if err != nil {
			return sender{}, false, err
		}
		return sender{
			userID:  id,
			isAdmin: false,
		}, true, nil
	case models.ChatTypePrivate:
		id, found, err := t.core.FindUser(ctx, core.SourceTelegram, externalID)
		if err != nil {
			return sender{}, false, err
		}
		if !found {
			return sender{}, false, nil
		}
		isAdmin, err := t.core.IsAdmin(ctx, id)
		if err != nil {
			return sender{}, false, err
		}
		if !isAdmin {
			return sender{}, false, nil
		}
		return sender{
			userID:  id,
			isAdmin: true,
		}, true, nil
	default:
		return sender{}, false, nil
	}
}

func pickRoute(in incoming, s sender, configured, hasDraft bool) route {
	if hasDraft {
		return routeDraft
	}
	if in.kind == kindCallback {
		return routeStaleCallback
	}
	if in.chatType == models.ChatTypePrivate {
		if !s.isAdmin {
			return routeIgnored
		}
		switch in.command {
		case "/start":
			return routeAdminStart
		case "/setup":
			if !configured {
				return routeSetup
			} else {
				return routeFalseSetup
			}
		}
		if !configured {
			return routeSetupFirst
		}
		return routeUnknownInput
	}
	if !configured {
		return routeNotConfigured
	}
	if in.command == "/start" {
		return routeStart
	}

	return routeUnknownInput
}
