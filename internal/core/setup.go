package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ptickz/expenses-monitor-golang/internal/period"
	"github.com/ptickz/expenses-monitor-golang/internal/store/db"
)

const SavingsAccountName = "Сберегательный"

var (
	ErrAlreadyConfigured = errors.New("already configured")
	ErrInvalidSetup      = errors.New("invalid setup input")
)

type SetupInput struct {
	AdminID      int32
	CurrencyID   int32
	AccountTitle string
	StartDay     int
	WeekMode     period.WeekMode
	TimeZone     string
	WeeklyAmount int64
}

func (c *Core) Setup(ctx context.Context, i SetupInput, now time.Time) error {
	if !period.ValidateStartDay(i.StartDay) {
		return fmt.Errorf("%w: wrong start day: %d", ErrInvalidSetup, i.StartDay)
	}
	DBWeekmode := db.WeekMode(i.WeekMode)
	if !DBWeekmode.Valid() {
		return fmt.Errorf("%w: wrong week mode: %v", ErrInvalidSetup, i.WeekMode)
	}
	loc, err := time.LoadLocation(i.TimeZone)
	if err != nil {
		return fmt.Errorf("%w: wrong location: %w", ErrInvalidSetup, err)
	}
	if i.TimeZone == "" || i.TimeZone == "Local" {
		return fmt.Errorf("%w: wrong location: %v", ErrInvalidSetup, i.TimeZone)
	}
	if i.WeeklyAmount < 0 {
		return fmt.Errorf("%w: wrong weekly amount: %d", ErrInvalidSetup, i.WeeklyAmount)
	}

	i.AccountTitle = strings.TrimSpace(i.AccountTitle)
	if i.AccountTitle == "" || strings.EqualFold(i.AccountTitle, SavingsAccountName) {
		return fmt.Errorf("%w: wrong account title: %v", ErrInvalidSetup, i.AccountTitle)
	}

	y, m, d := now.In(loc).Date()
	effectiveFrom := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	return c.st.InTx(ctx, func(q *db.Queries) error {
		check, err := q.IsConfigured(ctx)
		if err != nil {
			return fmt.Errorf("check configured: %w", err)
		}
		if check {
			return ErrAlreadyConfigured
		}

		err = q.ActivateCurrency(ctx, i.CurrencyID)
		if err != nil {
			return fmt.Errorf("activate currency: %w", err)
		}

		savingAcc, err := q.EnsureSavingsAccount(ctx, SavingsAccountName)
		if err != nil {
			return fmt.Errorf("create saving account: %w", err)
		}

		err = q.EnsureUnion(ctx, db.EnsureUnionParams{
			AccountID:  savingAcc.ID,
			CurrencyID: i.CurrencyID,
		})
		if err != nil {
			return fmt.Errorf("create saving account union: %w", err)
		}

		account, err := q.CreateAccount(ctx, i.AccountTitle)
		if err != nil {
			return fmt.Errorf("create account: %w", err)
		}

		union, err := q.CreateUnion(ctx, db.CreateUnionParams{
			AccountID:  account.ID,
			CurrencyID: i.CurrencyID,
		})
		if err != nil {
			return fmt.Errorf("create account union: %w", err)
		}

		periodSettings, err := q.CreatePeriodSettings(ctx, db.CreatePeriodSettingsParams{
			StartDay:      int32(i.StartDay),
			WeekMode:      DBWeekmode,
			TimeZone:      i.TimeZone,
			EffectiveFrom: effectiveFrom,
			CreatedBy:     i.AdminID,
		})
		if err != nil {
			return fmt.Errorf("create period settings: %w", err)
		}

		err = q.AddPeriodSettingsUnion(ctx, db.AddPeriodSettingsUnionParams{
			PeriodSettingsID: periodSettings.ID,
			UnionID:          union.ID,
		})
		if err != nil {
			return fmt.Errorf("create period settings union: %w", err)
		}

		_, err = q.CreateBudgetSettings(ctx, db.CreateBudgetSettingsParams{
			WeeklyAmount:  i.WeeklyAmount,
			EffectiveFrom: now,
			CreatedBy:     i.AdminID,
		})
		if err != nil {
			return fmt.Errorf("create budget settings: %w", err)
		}

		return nil
	})
}
