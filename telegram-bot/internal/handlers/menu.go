package handlers

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type Menu struct{}

func (h Menu) Name() string { return "menu" }

func (h Menu) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.CallbackQuery != nil && u.CallbackQuery.Data == "menu" {
		return true
	}
	if u.Message != nil {
		txt := strings.TrimSpace(strings.ToLower(u.Message.Text))
		return txt == "меню" || (u.Message.IsCommand() && u.Message.Command() == "menu")
	}
	return false
}

func (h Menu) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)

	if u.CallbackQuery != nil {
		_, _ = d.Bot.Request(tgbotapi.NewCallback(u.CallbackQuery.ID, "Ок"))
	}

	msg := tgbotapi.NewMessage(s.ChatID, "Меню:")
	msg.ReplyMarkup = menu.Keyboard()
	_, err := d.Bot.Send(msg)
	return err
}
