package handlers

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/payments"
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
	return s.State == "CHOOSE_VPN_COUNTRY" || s.State == "CHOOSE_VPN_COUNTRY_PROMOCODE"
}

func sendActiveSubscriptionMessage(bot *tgbotapi.BotAPI, chatID int64, country string, activeUntil appclient.TelegramCountryStatusResp) {
	msg := tgbotapi.NewMessage(
		chatID,
		fmt.Sprintf("У Вас уже есть подписка на %s. Активна до: %s",
			country,
			activeUntil.ActiveUntil.Format("2006-01-02 15:04"),
		),
	)
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = bot.Send(msg)
}

func (h CountryChosen) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	country := strings.TrimPrefix(u.CallbackQuery.Data, "country:")
	_, _ = d.Bot.Request(tgbotapi.NewCallback(u.CallbackQuery.ID, "Ок"))

	// Проверяем, есть ли уже активная подписка на эту страну
	st, err := d.App.TelegramCountryStatus(ctx, s.TgUserID, country)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог проверить подписку: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return nil
	}

	if st.Active {
		// Уже есть активная подписка на эту страну
		// Если это выбор страны после промокода - откатываем промокод (без указания кода - откатим последний)
		if s.State == "CHOOSE_VPN_COUNTRY_PROMOCODE" {
			_ = d.App.TelegramPromocodeRollback(ctx, s.TgUserID, "")
		}
		sendActiveSubscriptionMessage(d.Bot, s.ChatID, country, st)
		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return nil
	}

	// Если это выбор страны после промокода - обновляем подписку и выдаём ключ
	if s.State == "CHOOSE_VPN_COUNTRY_PROMOCODE" {
		// Нет активной подписки - обновляем подписку от промокода и выдаём ключ
		if err := d.App.TelegramUpdatePromocodeSubscription(ctx, s.TgUserID, country); err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Не смог обновить подписку: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
			return nil
		}

		ss := s
		ss.SelectedCountry = &country
		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return IssueKeyNow(ctx, ss, d)
	}

	// 2) no active subscription -> payment 150
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_VPN_PAYMENT", &country)

	err = payments.SendVPNInvoice(
		d.Bot,
		s.ChatID,
		d.Cfg.Payments.ProviderToken,
		d.Cfg.Payments.Currency,
		d.Cfg.Payments.VPNTtitle,
		d.Cfg.Payments.VPNDescription,
		d.Cfg.Payments.VPNPayload,
		d.Cfg.Payments.VPNPriceMinor,
	)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог отправить invoice: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	return nil
}
