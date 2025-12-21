package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type MySubscriptions struct{}

func (h MySubscriptions) Name() string { return "my_subs" }

func (h MySubscriptions) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && strings.EqualFold(strings.TrimSpace(u.Message.Text), menu.BtnMySubs)
}

func (h MySubscriptions) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	resp, err := d.App.TelegramSubscriptions(ctx, s.TgUserID)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог получить подписки: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if len(resp.Items) == 0 {
		msg := tgbotapi.NewMessage(s.ChatID, "Подписок нет.")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	var b strings.Builder
	b.WriteString("Твои подписки:\n")
	now := time.Now()

	for _, it := range resp.Items {
		cc := "—"
		if it.CountryCode != nil {
			cc = *it.CountryCode
		}
		until := "—"
		if it.ActiveUntil != nil {
			until = it.ActiveUntil.Format("2006-01-02 15:04")
		}
		status := "expired"
		if it.IsActive && (it.ActiveUntil == nil || it.ActiveUntil.After(now)) {
			status = "active"
		}
		b.WriteString(fmt.Sprintf("- kind=%s country=%s until=%s (%s)\n", it.Kind, cc, until, status))
	}

	msg := tgbotapi.NewMessage(s.ChatID, b.String())
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
