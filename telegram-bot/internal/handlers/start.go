package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/flows"
	"vpn-bot/internal/state"
)

type Start struct{}

func (h Start) Name() string { return "start" }

func (h Start) CanHandle(u tgbotapi.Update, s state.Session) bool {
	return u.Message != nil && u.Message.IsCommand() && u.Message.Command() == "start"
}

func (h Start) Handle(ctx context.Context, u tgbotapi.Update, s state.Session, d state.Deps) error {
	// reset state to first step
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "CHOOSE_COUNTRY", nil)

	msg := tgbotapi.NewMessage(s.ChatID, "Выбери страну сервера:")
	msg.ReplyMarkup = flows.CountryKeyboard()
	_, err := d.Bot.Send(msg)
	return err
}
