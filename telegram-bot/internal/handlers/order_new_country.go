package handlers

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/payments"
	"vpn-bot/internal/router"
)

type OrderNewCountry struct{}

func (h OrderNewCountry) Name() string { return "order_new_country" }

func (h OrderNewCountry) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && strings.EqualFold(strings.TrimSpace(u.Message.Text), menu.BtnOrderCountry)
}

func (h OrderNewCountry) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {

	if d.Cfg.Payments.ProviderToken == "" {
		_, _ = d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "country_request",
			CountryCode: nil,

			AmountMinor: d.Cfg.Payments.NewCountryPriceMinor,
			Currency:    d.Cfg.Payments.Currency,

			TelegramPaymentChargeID: "dev-bypass",
			ProviderPaymentChargeID: "dev-bypass",
		})

		_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_COUNTRY_REQUEST_TEXT", nil)

		msg := tgbotapi.NewMessage(s.ChatID, "Какую страну ты бы хотел добавить?")
		msg.ReplyMarkup = menu.Keyboard()
		_, err := d.Bot.Send(msg)
		return err
	}

	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_NEW_COUNTRY_PAYMENT", nil)

	err := payments.SendNewCountryInvoice(
		d.Bot,
		s.ChatID,
		d.Cfg.Payments.ProviderToken,
		d.Cfg.Payments.Currency,
		d.Cfg.Payments.NewCountryTitle,
		d.Cfg.Payments.NewCountryDescription,
		d.Cfg.Payments.NewCountryPayload,
		d.Cfg.Payments.NewCountryPriceMinor,
	)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог отправить invoice: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	return nil
}
