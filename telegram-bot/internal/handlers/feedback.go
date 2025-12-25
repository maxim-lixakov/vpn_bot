package handlers

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type SendFeedback struct{}

func (h SendFeedback) Name() string { return "send_feedback" }

func (h SendFeedback) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.Message != nil && strings.EqualFold(strings.TrimSpace(u.Message.Text), menu.BtnFeedback)
}

func (h SendFeedback) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_FEEDBACK", nil)

	msg := tgbotapi.NewMessage(s.ChatID, "Напишите ваш отзыв:")
	msg.ReplyMarkup = menu.Keyboard()
	_, err := d.Bot.Send(msg)
	return err
}

type FeedbackText struct{}

func (h FeedbackText) Name() string { return "feedback_text" }

func (h FeedbackText) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	if u.Message.IsCommand() {
		return false
	}
	return s.State == "AWAIT_FEEDBACK"
}

func (h FeedbackText) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Отзыв не может быть пустым. Напишите ваш отзыв:")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if err := d.App.TelegramFeedback(ctx, s.TgUserID, text); err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог отправить отзыв: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	_ = d.App.TelegramSetState(ctx, s.TgUserID, "MENU", nil)
	msg := tgbotapi.NewMessage(s.ChatID, "Ваш отзыв успешно отправлен. Спасибо за обратную связь!")
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
