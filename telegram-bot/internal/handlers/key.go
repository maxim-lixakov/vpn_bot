package handlers

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/state"
)

type KeyIssuer struct{}

func (h KeyIssuer) Name() string { return "key" }

func (h KeyIssuer) CanHandle(u tgbotapi.Update, s state.Session) bool {
	// example: user writes "key" while in ISSUE_KEY
	if u.Message != nil && s.State == "ISSUE_KEY" && !u.Message.IsCommand() {
		return true
	}
	return false
}

func (h KeyIssuer) Handle(ctx context.Context, u tgbotapi.Update, s state.Session, d state.Deps) error {
	if s.SelectedCountry == nil {
		_, err := d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не выбрана страна. Нажми /start"))
		return err
	}

	resp, err := d.App.IssueKey(ctx, s.TgUserID, *s.SelectedCountry)
	if err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Ошибка выдачи ключа: "+err.Error()))
		return nil
	}

	text := fmt.Sprintf("Сервер: %s\n\nКлюч:\n%s\n\n%s",
		resp.ServerName, resp.AccessURL, "Скачать Outline Client:\n"+officialLinks(),
	)
	_, err = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, text))
	return err
}
