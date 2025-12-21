package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type PaymentFlow struct{}

func (h PaymentFlow) Name() string { return "payment" }

func (h PaymentFlow) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.PreCheckoutQuery != nil || (u.Message != nil && u.Message.SuccessfulPayment != nil)
}

func (h PaymentFlow) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	if u.PreCheckoutQuery != nil {
		pc := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: u.PreCheckoutQuery.ID,
			OK:                 true,
		}
		_, err := d.Bot.Request(pc)
		return err
	}

	sp := u.Message.SuccessfulPayment
	payload := sp.InvoicePayload

	switch payload {
	case d.Cfg.Payments.VPNPayload:
		// must have selected country saved in session
		if s.SelectedCountry == nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Не выбрана страна. Нажми /start")
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "vpn",
			CountryCode: s.SelectedCountry,
			AmountMinor: int64(sp.TotalAmount),
			Currency:    sp.Currency,

			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: sp.ProviderPaymentChargeID,
		})
		if err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог сохранить подписку: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		msg := tgbotapi.NewMessage(s.ChatID, "Оплата успешна. Выдаю ключ…")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)

		return issueKeyNow(ctx, s, d)

	case d.Cfg.Payments.NewCountryPayload:
		_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "country_request",
			CountryCode: nil,
			AmountMinor: int64(sp.TotalAmount),
			Currency:    sp.Currency,

			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: sp.ProviderPaymentChargeID,
		})
		if err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог сохранить: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_COUNTRY_REQUEST_TEXT", nil)

		msg := tgbotapi.NewMessage(s.ChatID, "Какую страну ты бы хотел добавить?")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// неизвестный payload
	msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но payload не распознан.")
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
