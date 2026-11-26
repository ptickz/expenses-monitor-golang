package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/ptickz/expenses-monitor-golang/internal/core"
	"github.com/ptickz/expenses-monitor-golang/internal/money"
	"github.com/ptickz/expenses-monitor-golang/internal/period"
)

const startupMessage string = "Настройка бота. Займёт пару минут, 6 вопросов:\n" +
	"1. Валюта — в ней бот будет вести бюджет. Код из 3–4 букв: GEL, USD, USDT. " +
	"Автоматически будете заведен сберегательный счет в этой валюте. " +
	"Дополнительные валюты вы можете активировать позже.\n" +
	"2. Счёт — где лежат деньги на бюджет: банк, карта, наличные. " +
	"Бот будет считать недели и свободные деньги по этому счёту в выбранной валюте. " +
	"Другие счета можно добавить позже.\n" +
	"3. День начала периода — число месяца, с которого начинается расчётный " +
	"период (например, день зарплаты). Если в месяце меньше дней — последний день месяца.\n" +
	"4. Режим недель — как делить период на недели:\n" +
	"• от начала периода — по 7 дней от дня начала, последняя неделя короче;\n" +
	"• календарные Пн–Вс — неполные недели в начале и конце периода делят одну недельную сумму.\n" +
	"5. Часовой пояс — в формате IANA, например Europe/Moscow, Asia/Tbilisi.\n" +
	"6. Сумма на неделю — сколько выделяется на одну неделю.\n\n" +
	"Настройки начнут действовать с сегодняшнего дня. " +
	"В конце будет выведена сводка — можно сохранить или отменить. " +
	"Если не отвечать 10 минут, настройка сбросится."

const (
	IncorrectInput        string = "Некорректный ввод"
	CallbackWeekPrefix    string = "setup:week:"
	CallbackConfirmPrefix string = "setup:confirm:"
)

const (
	stepCurrency setupStep = iota
	stepAccount
	stepStartDay
	stepWeekMode
	stepTimeZone
	stepWeeklyAmount
	stepConfirm
)

type setupStep int

type setupFlow struct {
	step         setupStep
	adminID      int32
	currencyID   int32
	currencyCode string
	minorUnits   int16
	accountTitle string
	startDay     int
	weekMode     period.WeekMode
	timeZone     string
	weeklyAmount int64
}

func (t *Bot) handleSetup(ctx context.Context, in incoming, s sender) {
	wctx := context.WithoutCancel(ctx)
	f := &setupFlow{adminID: s.userID}
	d := &draft{authorID: in.userID, flow: f}
	t.reply(ctx, in.chatID, startupMessage, nil)
	d.messageID = t.reply(ctx, in.chatID, f.question(), nil)
	t.drafts.start(
		in.chatID,
		in.userID,
		d,
		func() {
			t.reply(wctx, in.chatID, "Время вышло, начните /setup заново", nil)
		},
	)
}

func (f *setupFlow) handle(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	switch f.step {
	case stepCurrency:
		return f.onCurrency(ctx, t, d, in)
	case stepAccount:
		return f.onAccount(ctx, t, d, in)
	case stepStartDay:
		return f.onStartDay(ctx, t, d, in)
	case stepWeekMode:
		return f.onWeekMode(ctx, t, d, in)
	case stepTimeZone:
		return f.onTimeZone(ctx, t, d, in)
	case stepWeeklyAmount:
		return f.onWeeklyAmount(ctx, t, d, in)
	case stepConfirm:
		return f.onConfirm(ctx, t, d, in)
	default:
		return false, fmt.Errorf("unknown setup step %d", f.step)
	}
}

func (f *setupFlow) onCurrency(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindText {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	currency, check, err := t.core.FindCurrency(ctx, in.text)
	if err != nil {
		return false, err
	}
	if !check {
		return f.reject(ctx, t, d, in, "Валюты с таким кодом не найдено, перепроверьте корректность ввода")
	}
	f.currencyID = currency.ID
	f.currencyCode = currency.Code
	f.minorUnits = currency.MinorUnits
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onAccount(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindText {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	if !core.ValidateAccountName(in.text) {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	f.accountTitle = in.text
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onStartDay(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindText {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	n, err := strconv.Atoi(in.text)
	if err != nil || !period.ValidateStartDay(n) {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	f.startDay = n
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onWeekMode(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindCallback || in.messageID != d.messageID {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	modeInput, check := strings.CutPrefix(in.callbackData, CallbackWeekPrefix)
	m := period.WeekMode(modeInput)
	if !check || !m.Valid() {
		return false, fmt.Errorf("unexpected week mode callback %q", in.callbackData)
	}
	f.weekMode = m
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onTimeZone(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindText {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	loc, ok := core.ParseTimeZone(in.text)
	if !ok {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	f.timeZone = loc.String()
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onWeeklyAmount(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindText {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	amount, err := money.ParseAmount(in.text, f.minorUnits)
	if err != nil {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	f.weeklyAmount = amount
	return f.next(ctx, t, d, in)
}

func (f *setupFlow) onConfirm(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	if in.kind != kindCallback || in.messageID != d.messageID {
		return f.reject(ctx, t, d, in, IncorrectInput)
	}
	t.removeKeyboard(ctx, in.chatID, in.messageID)
	confirmInput, check := strings.CutPrefix(in.callbackData, CallbackConfirmPrefix)
	if !check {
		return false, fmt.Errorf("unexpected confirm callback %q", in.callbackData)
	}
	switch confirmInput {
	case "true":
		return f.save(ctx, t, in)
	case "false":
		t.reply(ctx, in.chatID, "Настройка отменена", nil)
		return true, nil
	default:
		return false, fmt.Errorf("unexpected confirm callback value %q", in.callbackData)
	}
}

func (f *setupFlow) save(ctx context.Context, t *Bot, in incoming) (bool, error) {
	err := t.core.Setup(ctx, core.SetupInput{
		AdminID:      f.adminID,
		CurrencyID:   f.currencyID,
		AccountTitle: f.accountTitle,
		StartDay:     f.startDay,
		WeekMode:     f.weekMode,
		TimeZone:     f.timeZone,
		WeeklyAmount: f.weeklyAmount,
	}, time.Now())
	if errors.Is(err, core.ErrAlreadyConfigured) {
		t.reply(ctx, in.chatID, "Бот уже был настроен", nil)
		return true, nil
	}
	if errors.Is(err, core.ErrInvalidSetup) {
		t.reply(ctx, in.chatID, "Данные не прошли проверку, перепроверьте и начните /setup заново", nil)
		return true, nil
	}
	if err != nil {
		t.reply(ctx, in.chatID, "Не удалось сохранить, попробуйте позже", nil)
		return true, err
	}
	t.reply(ctx, in.chatID, "✅ Настройка сохранена", nil)
	return true, nil
}

func (f *setupFlow) reject(ctx context.Context, t *Bot, d *draft, in incoming, text string) (bool, error) {
	t.reply(ctx, in.chatID, text, nil)
	return f.ask(ctx, t, d, in)
}

func (f *setupFlow) next(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	f.step++
	return f.ask(ctx, t, d, in)
}

func (f *setupFlow) ask(ctx context.Context, t *Bot, d *draft, in incoming) (bool, error) {
	messageID := t.reply(ctx, in.chatID, f.question(), f.keyboard())
	if messageID == 0 {
		return false, fmt.Errorf("send setup question")
	}
	t.removeKeyboard(ctx, in.chatID, d.messageID)
	d.messageID = messageID
	return false, nil
}

func (f *setupFlow) question() string {
	switch f.step {
	case stepCurrency:
		return "Введите код валюты. Пример: GEL, USD, USDT."
	case stepAccount:
		return "Введите название счёта (банка или платформы)."
	case stepStartDay:
		return "Введите число месяца начала расчётного периода в диапазоне от 1 до 31."
	case stepWeekMode:
		return "Выберите расчётный режим для периода"
	case stepTimeZone:
		return "В каком часовом поясе находитесь? Нужен формат IANA, например Asia/Tbilisi."
	case stepWeeklyAmount:
		return fmt.Sprintf("Какую сумму закладываете на неделю расчётного периода в %s?", f.currencyCode)
	case stepConfirm:
		return fmt.Sprintf(
			"Проверьте данные:\n"+
				"Валюта: %s\n"+
				"Счёт: %s\n"+
				"День начала: %d\n"+
				"Режим недель: %s\n"+
				"Часовой пояс: %s\n"+
				"Сумма на неделю: %s %s",
			f.currencyCode, f.accountTitle, f.startDay,
			weekModeLabel(f.weekMode), f.timeZone,
			money.FormatAmount(f.weeklyAmount, f.minorUnits), f.currencyCode,
		)
	default:
		return ""
	}
}

func weekModeLabel(m period.WeekMode) string {
	switch m {
	case period.FromPeriodStart:
		return "С начала расчётного периода по 7 дней"
	case period.CalendarWeeks:
		return "По календарным неделям ПН-ВС"
	default:
		return ""
	}
}

func (f *setupFlow) keyboard() models.ReplyMarkup {
	switch f.step {
	case stepWeekMode:
		return &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: weekModeLabel(period.FromPeriodStart), CallbackData: CallbackWeekPrefix + string(period.FromPeriodStart)}},
				{{Text: weekModeLabel(period.CalendarWeeks), CallbackData: CallbackWeekPrefix + string(period.CalendarWeeks)}},
			},
		}
	case stepConfirm:
		return &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "✅ Сохранить", CallbackData: CallbackConfirmPrefix + "true"},
					{Text: "❌ Отмена", CallbackData: CallbackConfirmPrefix + "false"},
				},
			},
		}
	}
	return nil
}
