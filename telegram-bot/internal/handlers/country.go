package handlers

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/state"
)

type CountryChosen struct{}

func (h CountryChosen) Name() string { return "country" }

func (h CountryChosen) CanHandle(u tgbotapi.Update, s state.Session) bool {
	if u.CallbackQuery == nil {
		return false
	}
	return strings.HasPrefix(u.CallbackQuery.Data, "country:")
}

func (h CountryChosen) Handle(ctx context.Context, u tgbotapi.Update, s state.Session, d state.Deps) error {
	country := strings.TrimPrefix(u.CallbackQuery.Data, "country:")
	if _, err := d.Bot.Request(tgbotapi.NewCallback(u.CallbackQuery.ID, "Готово")); err != nil {
		// можно залогировать
	}

	// move to payment step
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_PAYMENT", &country)

	if d.Cfg.Payments.ProviderToken == "" {
		// DEV BYPASS
		_, _ = d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:                s.TgUserID,
			AmountMinor:             d.Cfg.Payments.PriceMinor,
			Currency:                d.Cfg.Payments.Currency,
			TelegramPaymentChargeID: "dev-bypass",
			ProviderPaymentChargeID: "dev-bypass",
		})

		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Оплата временно пропущена (dev mode). Выдаю ключ…"))
		// важно: state.SelectedCountry в Session ещё старый, поэтому сформируй новую сессию локально:
		ss := s
		ss.SelectedCountry = &country
		return issueKeyNow(ctx, ss, d)
	}

	// delegate to payment instructions (send invoice or message)
	text := "Далее — оплата подписки 100 руб/мес."
	_, err := d.Bot.Send(tgbotapi.NewMessage(s.ChatID, text))
	return err
}
