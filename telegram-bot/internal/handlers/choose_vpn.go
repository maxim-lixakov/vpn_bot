package handlers

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/countries"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type ChooseVPN struct{}

func (h ChooseVPN) Name() string { return "choose_vpn" }

func (h ChooseVPN) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && strings.EqualFold(strings.TrimSpace(u.Message.Text), menu.BtnChooseVPN)
}

func (h ChooseVPN) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	if err := d.App.TelegramSetState(ctx, s.TgUserID, "CHOOSE_VPN_COUNTRY", nil); err != nil {
		log.Printf("TelegramSetState failed: %v", err)
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог переключить состояние (ошибка сервера). Попробуй ещё раз /start")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	msg := tgbotapi.NewMessage(s.ChatID, "Выбери страну VPN:")
	msg.ReplyMarkup = countries.CountryKeyboard()
	_, err := d.Bot.Send(msg)
	return err
}
