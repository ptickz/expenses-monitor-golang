package telegram

import (
	"log/slog"
	"strings"

	"github.com/go-telegram/bot/models"
)

type kind int

const (
	kindOther kind = iota
	kindCommand
	kindText
	kindCallback
)

func (k kind) String() string {
	switch k {
	case kindCommand:
		return "command"
	case kindText:
		return "text"
	case kindCallback:
		return "callback"
	default:
		return "other"
	}
}

type incoming struct {
	updateID     int64
	chatID       int64
	chatType     models.ChatType
	userID       int64
	userName     string
	messageID    int
	kind         kind
	command      string
	args         string
	text         string
	callbackID   string
	callbackData string
}

func (t *Bot) parse(u *models.Update) incoming {
	in := incoming{updateID: u.ID}
	switch {
	case u.Message != nil:
		m := u.Message
		if m.From != nil {
			in.userID = m.From.ID
			in.userName = m.From.FirstName
		}
		in.chatID = m.Chat.ID
		in.chatType = m.Chat.Type
		in.messageID = m.ID
		if m.Text == "" {
			in.kind = kindOther
			return in
		}
		if strings.HasPrefix(m.Text, "/") {
			fields := strings.Fields(m.Text)
			cmd, target, found := strings.Cut(fields[0], "@")
			if found && !strings.EqualFold(target, t.botName) {
				in.kind = kindOther
				return in
			}
			in.command = cmd
			in.args = strings.Join(fields[1:], " ")
			in.kind = kindCommand
		} else {
			in.text = m.Text
			in.kind = kindText
		}
	case u.CallbackQuery != nil:
		q := u.CallbackQuery
		if chat := callbackChat(q); chat != nil {
			in.chatID = chat.ID
			in.chatType = chat.Type
		}
		if q.Message.Message != nil {
			in.messageID = q.Message.Message.ID
		}
		in.userID = q.From.ID
		in.userName = q.From.FirstName
		in.callbackID = q.ID
		in.callbackData = q.Data
		in.kind = kindCallback
	}

	return in
}

func (in incoming) logAttrs() []slog.Attr {
	attrs := []slog.Attr{slog.Int64("update_id", in.updateID)}
	attrs = append(
		attrs,
		slog.Int64("chat_id", in.chatID),
		slog.String("chat_type", string(in.chatType)),
		slog.Int64("user_id", in.userID),
		slog.String("kind", in.kind.String()),
	)
	if in.command != "" {
		attrs = append(attrs, slog.String("command", in.command))
	}
	if in.callbackData != "" {
		attrs = append(attrs, slog.String("callback", in.callbackData))
	}

	return attrs
}

func callbackChat(q *models.CallbackQuery) *models.Chat {
	switch {
	case q.Message.Message != nil:
		return &q.Message.Message.Chat
	case q.Message.InaccessibleMessage != nil:
		return &q.Message.InaccessibleMessage.Chat
	default:
		return nil
	}
}
