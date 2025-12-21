package handlers

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type CountryRequestText struct{}

func (h CountryRequestText) Name() string { return "country_request_text" }

func (h CountryRequestText) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	if u.Message.IsCommand() {
		return false
	}
	return s.State == "AWAIT_COUNTRY_REQUEST_TEXT"
}

func (h CountryRequestText) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Напиши текстом страну/запрос.")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if err := d.App.TelegramCreateCountryToAdd(ctx, s.TgUserID, text); err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог сохранить запрос: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
	msg := tgbotapi.NewMessage(s.ChatID, "Ок, записал. Мы добавим и сообщим.")
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
