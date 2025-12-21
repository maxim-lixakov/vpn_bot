package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type Start struct{}

func (h Start) Name() string { return "start" }

func (h Start) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && u.Message.IsCommand() && u.Message.Command() == "start"
}

func (h Start) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)

	msg := tgbotapi.NewMessage(s.ChatID, "Меню:")
	msg.ReplyMarkup = menu.Keyboard()
	_, err := d.Bot.Send(msg)
	return err
}
