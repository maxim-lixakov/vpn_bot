package handlers

import (
	"context"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/router"
)

type DailyStats struct{}

func (h DailyStats) Name() string { return "daily_stats" }

func (h DailyStats) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	return strings.TrimSpace(u.Message.Text) == "/daily_stats"
}

func (h DailyStats) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	adminTgUserIDStr := os.Getenv("BACKUP_ADMIN_TG_USER_ID")
	if adminTgUserIDStr == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Админ не настроен")
		_, _ = d.Bot.Send(msg)
		return nil
	}

	adminTgUserID, err := strconv.ParseInt(adminTgUserIDStr, 10, 64)
	if err != nil || s.TgUserID != adminTgUserID {
		msg := tgbotapi.NewMessage(s.ChatID, "У вас нет прав для выполнения этой команды")
		_, _ = d.Bot.Send(msg)
		return nil
	}

	resp, err := d.App.DailyStats(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Ошибка при получении статистики: "+err.Error())
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if !resp.Success {
		msg := tgbotapi.NewMessage(s.ChatID, "Ошибка: "+resp.Error)
		_, _ = d.Bot.Send(msg)
		return nil
	}

	msg := tgbotapi.NewMessage(s.ChatID, "Статистика отправлена")
	_, _ = d.Bot.Send(msg)
	return nil
}
