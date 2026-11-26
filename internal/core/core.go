package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/ptickz/expenses-monitor-golang/internal/store"
	"github.com/ptickz/expenses-monitor-golang/internal/store/db"
)

type UserSource string

const (
	SourceTelegram UserSource = "TELEGRAM"
	SourceHTTP     UserSource = "HTTP"
)

type Core struct {
	logger *slog.Logger
	st     *store.Store
}

func New(l *slog.Logger, st *store.Store) *Core {
	l = l.With("component", "core")

	return &Core{
		logger: l,
		st:     st,
	}
}

func (c *Core) BootstrapAdmin(ctx context.Context, src UserSource, externalID string) error {
	u, err := c.st.Queries().EnsureUser(ctx, db.EnsureUserParams{
		Source:     db.UserSource(src),
		ExternalID: externalID,
		Name:       "none",
	})
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}

	err = c.st.Queries().GrantAdmin(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("grant admin: %w", err)
	}
	return nil
}

func (c *Core) RegisterUser(ctx context.Context, src UserSource, externalID, name string) (int32, error) {
	u, err := c.st.Queries().UpsertUser(ctx, db.UpsertUserParams{
		Source:     db.UserSource(src),
		ExternalID: externalID,
		Name:       name,
	})
	if err != nil {
		return 0, fmt.Errorf("upsert user: %w", err)
	}

	return u.ID, nil
}

func (c *Core) FindUser(ctx context.Context, src UserSource, externalID string) (id int32, found bool, err error) {
	u, err := c.st.Queries().GetUserByExternalID(ctx, db.GetUserByExternalIDParams{
		Source:     db.UserSource(src),
		ExternalID: externalID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("find user: %w", err)
	}

	return u.ID, true, nil
}

func (c *Core) IsAdmin(ctx context.Context, userID int32) (bool, error) {
	check, err := c.st.Queries().IsAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("is admin: %w", err)
	}

	return check, nil
}

func (c *Core) Configured(ctx context.Context) (bool, error) {
	check, err := c.st.Queries().IsConfigured(ctx)
	if err != nil {
		return false, fmt.Errorf("is configured: %w", err)
	}

	return check, nil
}

func (c *Core) FindCurrency(ctx context.Context, code string) (db.Currency, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	currency, err := c.st.Queries().GetCurrencyByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Currency{}, false, nil
		}
		return db.Currency{}, false, fmt.Errorf("find currency: %w", err)
	}
	return currency, true, nil
}

func ParseTimeZone(s string) (*time.Location, bool) {
	if s == "" || s == "Local" {
		return nil, false
	} else {
		tz, err := time.LoadLocation(s)
		if err != nil {
			return nil, false
		} else {
			return tz, true
		}
	}
}

func ValidateAccountName(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && !strings.EqualFold(s, SavingsAccountName) && utf8.RuneCountInString(s) <= 64
}
