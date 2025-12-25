package handlers

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/countries"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type UsePromocode struct{}

func (h UsePromocode) Name() string { return "use_promocode" }

func (h UsePromocode) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && strings.EqualFold(strings.TrimSpace(u.Message.Text), menu.BtnUsePromocode)
}

func (h UsePromocode) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_PROMOCODE", nil)

	msg := tgbotapi.NewMessage(s.ChatID, "Введите промокод:")
	msg.ReplyMarkup = menu.Keyboard()
	_, err := d.Bot.Send(msg)
	return err
}

type PromocodeText struct{}

func (h PromocodeText) Name() string { return "promocode_text" }

func (h PromocodeText) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	if u.Message.IsCommand() {
		return false
	}
	return s.State == "AWAIT_PROMOCODE"
}

func (h PromocodeText) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	code := strings.TrimSpace(u.Message.Text)
	if code == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Промокод не может быть пустым. Введите промокод:")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	resp, err := d.App.TelegramPromocodeUse(ctx, s.TgUserID, code)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Ошибка при проверке промокода: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if !resp.Valid {
		msg := tgbotapi.NewMessage(s.ChatID, resp.Message)
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return nil
	}

	// Промокод валиден - создаём подписку на указанное количество месяцев
	// Используем dev-bypass для оплаты (0 рублей)
	_, err = d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
		TgUserID:    s.TgUserID,
		Kind:        "vpn",
		CountryCode: nil, // Промокод даёт подписку без привязки к стране
		AmountMinor: 0,
		Currency:    d.Cfg.Payments.Currency,
		Months:      resp.Months,

		TelegramPaymentChargeID: "promocode",
		ProviderPaymentChargeID: "promocode",
	})
	if err != nil {
		// Откатываем использование промокода при ошибке создания подписки
		_ = d.App.TelegramPromocodeRollback(ctx, s.TgUserID, code)
		msg := tgbotapi.NewMessage(s.ChatID, "Промокод применён, но не удалось создать подписку: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
		return nil
	}

	monthsText := "месяц"
	if resp.Months > 1 && resp.Months < 5 {
		monthsText = "месяца"
	} else if resp.Months >= 5 {
		monthsText = "месяцев"
	}

	// Сообщаем об успешном применении промокода
	msg := tgbotapi.NewMessage(s.ChatID, resp.Message+". Подписка активирована на "+fmt.Sprintf("%d %s", resp.Months, monthsText)+". Выбери страну для VPN:")
	msg.ReplyMarkup = countries.CountryKeyboard()
	_, _ = d.Bot.Send(msg)

	// Переводим в состояние выбора страны для промокода
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "CHOOSE_VPN_COUNTRY_PROMOCODE", nil)
	return nil
}
