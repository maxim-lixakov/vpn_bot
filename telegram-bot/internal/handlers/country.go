package handlers

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type CountryChosen struct{}

func (h CountryChosen) Name() string { return "country" }

func (h CountryChosen) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.CallbackQuery == nil {
		return false
	}
	if !strings.HasPrefix(u.CallbackQuery.Data, "country:") {
		return false
	}
	return s.State == "CHOOSE_VPN_COUNTRY"
}

func (h CountryChosen) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	country := strings.TrimPrefix(u.CallbackQuery.Data, "country:")
	_, _ = d.Bot.Request(tgbotapi.NewCallback(u.CallbackQuery.ID, "Ок"))

	// 1) check active subscription for this country
	st, err := d.App.TelegramCountryStatus(ctx, s.TgUserID, country)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог проверить подписку: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if st.Active {
		msg := tgbotapi.NewMessage(s.ChatID, fmt.Sprintf("У тебя уже есть подписка на %s. Активна до: %s", country, st.ActiveUntil.Format("2006-01-02 15:04")))
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)

		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return nil
	}

	// 2) no active subscription -> payment 100 (skip by env for now)
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_VPN_PAYMENT", &country)

	if d.Cfg.Payments.ProviderToken == "" {
		_, _ = d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "vpn",
			CountryCode: &country,
			AmountMinor: d.Cfg.Payments.VPNPriceMinor,
			Currency:    d.Cfg.Payments.Currency,

			TelegramPaymentChargeID: "dev-bypass",
			ProviderPaymentChargeID: "dev-bypass",
		})

		msg := tgbotapi.NewMessage(s.ChatID, "Оплата пропущена (dev). Выдаю ключ…")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)

		ss := s
		ss.SelectedCountry = &country
		return issueKeyNow(ctx, ss, d)
	}

	// later: send invoice immediately (real payments)
	// (можно вынести в отдельный метод, но пока так)
	price := tgbotapi.LabeledPrice{Label: "VPN 1 month", Amount: int(d.Cfg.Payments.VPNPriceMinor)}
	inv := tgbotapi.NewInvoice(
		s.ChatID,
		d.Cfg.Payments.VPNTtitle,
		d.Cfg.Payments.VPNDescription+"\nCountry: "+country,
		d.Cfg.Payments.VPNPayload,
		d.Cfg.Payments.ProviderToken,
		"",
		d.Cfg.Payments.Currency,
		[]tgbotapi.LabeledPrice{price},
	)

	_, err = d.Bot.Send(inv)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог отправить invoice: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	return nil
}
